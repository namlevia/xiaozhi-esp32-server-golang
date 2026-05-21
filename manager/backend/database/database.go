package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/models"
	"xiaozhi/manager/backend/services/configprovider"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Init(cfg config.DatabaseConfig) *gorm.DB {
	var db *gorm.DB
	var err error

	storageType := cfg.GetStorageType()

	if storageType == "sqlite" {
		if cfg.SQLite == nil {
			log.Println("Cấu hình SQLite trống, sẽ chạy ở chế độ fallback (xác thực người dùng hard-code)")
			return nil
		}
		// Đảm bảo thư mục chứa file cơ sở dữ liệu tồn tại để SQLite không báo unable to open database file
		dir := filepath.Dir(cfg.SQLite.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Tạo thư mục cơ sở dữ liệu thất bại %s: %v", dir, err)
			return nil
		}
		log.Println("Đang dùng cơ sở dữ liệu SQLite:", cfg.SQLite.FilePath)
		db, err = gorm.Open(sqlite.Open(cfg.SQLite.FilePath), &gorm.Config{})
	} else {
		if cfg.MySQL == nil {
			log.Println("Cấu hình MySQL trống, sẽ chạy ở chế độ fallback (xác thực người dùng hard-code)")
			return nil
		}
		// Kết nối cơ sở dữ liệu MySQL
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.MySQL.Username, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		log.Println("Kết nối cơ sở dữ liệu thất bại:", err)
		log.Println("Sẽ chạy ở chế độ fallback (xác thực người dùng hard-code)")
		return nil
	}

	log.Println("Kết nối cơ sở dữ liệu thành công")

	// Tự động migrate cấu trúc bảng cơ sở dữ liệu
	log.Println("Bắt đầu tự động migrate cấu trúc bảng cơ sở dữ liệu...")
	err = db.AutoMigrate(
		&models.User{},
		&models.APIToken{},
		&models.Device{},
		&models.Agent{},
		&models.KnowledgeBase{},
		&models.KnowledgeBaseDocument{},
		&models.AgentKnowledgeBase{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.Role{}, // Bảng vai trò hợp nhất
		&models.ChatMessage{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.VoiceClone{},
		&models.VoiceCloneAudio{},
		&models.VoiceCloneTask{},
		&models.UserVoiceCloneQuota{},
	)
	if err != nil {
		log.Printf("Migrate cấu trúc bảng cơ sở dữ liệu thất bại: %v", err)
		log.Println("Sẽ chạy ở chế độ fallback (xác thực người dùng hard-code)")
		return nil
	}
	log.Println("Migrate cấu trúc bảng cơ sở dữ liệu thành công")

	if err := dropDeprecatedAgentStatusColumn(db); err != nil {
		log.Printf("Xóa cột trạng thái trợ lý cũ thất bại: %v", err)
	}

	// Migrate dữ liệu vai trò toàn cục hiện có sang bảng roles mới
	log.Println("Kiểm tra có cần migrate dữ liệu vai trò toàn cục không...")
	if err := migrateGlobalRolesToRoles(db); err != nil {
		log.Printf("Migrate dữ liệu vai trò toàn cục thất bại: %v", err)
		// Migrate thất bại không ảnh hưởng quá trình khởi động, chỉ chưa migrate dữ liệu
	}
	if err := repairConfigProviders(db); err != nil {
		log.Printf("Sửa provider cấu hình thất bại: %v", err)
	}
	if err := ensureDefaultVADConfig(db); err != nil {
		log.Printf("Tạo cấu hình VAD mặc định thất bại: %v", err)
	}
	if err := ensureDefaultLLMConfig(db); err != nil {
		log.Printf("Tạo cấu hình LLM mặc định thất bại: %v", err)
	}
	if err := ensureDefaultASRConfig(db); err != nil {
		log.Printf("Tạo cấu hình ASR mặc định thất bại: %v", err)
	}
	if err := ensureDefaultTTSConfig(db); err != nil {
		log.Printf("Tạo cấu hình TTS mặc định thất bại: %v", err)
	}
	if err := ensureDefaultPiperTTSConfig(db); err != nil {
		log.Printf("Tạo cấu hình Piper TTS offline thất bại: %v", err)
	}
	if err := ensureDefaultMemoryConfig(db); err != nil {
		log.Printf("Tạo cấu hình Memory mặc định thất bại: %v", err)
	}
	if err := ensureDefaultKnowledgeSearchConfig(db); err != nil {
		log.Printf("Tạo cấu hình truy xuất kho tri thức mặc định thất bại: %v", err)
	}
	return db
}

func dropDeprecatedAgentStatusColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Agent{}) {
		return nil
	}
	hasColumn, err := hasDatabaseColumn(db, "agents", "status")
	if err != nil {
		return err
	}
	if !hasColumn {
		return nil
	}
	err = db.Exec("ALTER TABLE agents DROP COLUMN status").Error
	if err != nil {
		return err
	}
	log.Println("Đã xóa cột trạng thái trợ lý cũ agents.status")
	return nil
}

func hasDatabaseColumn(db *gorm.DB, tableName, columnName string) (bool, error) {
	switch db.Dialector.Name() {
	case "sqlite":
		var columns []struct {
			Name string `gorm:"column:name"`
		}
		if err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).Scan(&columns).Error; err != nil {
			return false, err
		}
		for _, column := range columns {
			if column.Name == columnName {
				return true, nil
			}
		}
		return false, nil
	case "mysql":
		var count int64
		if err := db.Raw(
			"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
			tableName,
			columnName,
		).Scan(&count).Error; err != nil {
			return false, err
		}
		return count > 0, nil
	default:
		return db.Migrator().HasColumn(tableName, columnName), nil
	}
}

func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Println("Lấy kết nối cơ sở dữ liệu thất bại:", err)
		return
	}
	sqlDB.Close()
}

// migrateGlobalRolesToRoles migrate dữ liệu vai trò toàn cục hiện có sang bảng roles mới
func migrateGlobalRolesToRoles(db *gorm.DB) error {
	// Kiểm tra bảng roles đã có dữ liệu chưa
	var count int64
	if err := db.Table("roles").Count(&count).Error; err != nil {
		return fmt.Errorf("Kiểm tra bảng roles thất bại: %w", err)
	}

	// Nếu bảng roles đã có dữ liệu thì bỏ qua migrate
	if count > 0 {
		log.Println("Bảng roles đã có dữ liệu, bỏ qua migrate")
		return nil
	}

	// Kiểm tra bảng global_roles có dữ liệu không
	var globalRoleCount int64
	if err := db.Table("global_roles").Count(&globalRoleCount).Error; err != nil {
		// Bảng global_roles có thể không tồn tại, đây không phải lỗi
		log.Println("Bảng global_roles không tồn tại, bỏ qua migrate")
		return nil
	}

	if globalRoleCount == 0 {
		log.Println("Bảng global_roles không có dữ liệu, bỏ qua migrate")
		return nil
	}

	log.Printf("Bắt đầu migrate %d vai trò toàn cục sang bảng roles...", globalRoleCount)

	// Truy vấn tất cả vai trò toàn cục
	var globalRoles []models.GlobalRole
	if err := db.Table("global_roles").Find(&globalRoles).Error; err != nil {
		return fmt.Errorf("Truy vấn global_roles thất bại: %w", err)
	}

	// Chuyển đổi và insert vào bảng roles
	for _, gr := range globalRoles {
		role := models.Role{
			UserID:      nil, // Vai trò toàn cục có user_id là NULL
			Name:        gr.Name,
			Description: gr.Description,
			Prompt:      gr.Prompt,
			RoleType:    "global",
			Status:      "active",
			SortOrder:   0,
			IsDefault:   gr.IsDefault,
			CreatedAt:   gr.CreatedAt,
			UpdatedAt:   gr.UpdatedAt,
		}
		if err := db.Create(&role).Error; err != nil {
			log.Printf("Insert vai trò %s thất bại: %v", gr.Name, err)
			continue
		}
		log.Printf("Đã migrate vai trò toàn cục: %s", gr.Name)
	}

	log.Println("Migrate dữ liệu vai trò toàn cục hoàn tất")
	return nil
}

func ensureDefaultVADConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "vad").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data := map[string]interface{}{
		"provider":           "ten_vad",
		"hop_size":           320,
		"threshold":          0.3,
		"pool_size":          10,
		"acquire_timeout_ms": 3000,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "vad",
		Name:      "TEN VAD mặc định",
		ConfigID:  "ten_vad_default",
		Provider:  "ten_vad",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình VAD mặc định TEN VAD")
	return nil
}

func ensureDefaultLLMConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "llm").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data := map[string]interface{}{
		"type":        "openai",
		"model_name":  "cx/gpt-5.5",
		"api_key":     "",
		"base_url":    "https://api.9router.com/v1",
		"max_tokens":  4000,
		"temperature": 0.7,
		"top_p":       0.9,
		"thinking": map[string]interface{}{
			"mode":   "default",
			"effort": "medium",
		},
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "llm",
		Name:      "9Router GPT-5.5",
		ConfigID:  "9router_gpt_5_5",
		Provider:  "9router",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình LLM mặc định 9Router GPT-5.5")
	return nil
}

func ensureDefaultASRConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "asr").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	baseURL := os.Getenv("DEFAULT_ASR_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:1231"
	}
	data := map[string]interface{}{
		"base_url":    baseURL,
		"sample_rate": 16000,
		"timeout_ms":  30000,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "asr",
		Name:      "Vietnamese ASR (Go)",
		ConfigID:  "wyoming_vietnamese_asr_default",
		Provider:  "wyoming_vietnamese_asr",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình ASR mặc định Vietnamese ASR (Go)")
	return nil
}

func ensureDefaultTTSConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "tts").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	serverURL := os.Getenv("DEFAULT_EDGE_OFFLINE_URL")
	if serverURL == "" {
		serverURL = "ws://127.0.0.1:1232/tts"
	}
	data := map[string]interface{}{
		"provider":       "edge_offline",
		"server_url":     serverURL,
		"timeout":        30,
		"sample_rate":    16000,
		"channels":       1,
		"frame_duration": 20,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "tts",
		Name:      "Edge Offline TTS",
		ConfigID:  "edge_offline_default",
		Provider:  "edge_offline",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình TTS mặc định Edge Offline")
	return nil
}

func ensureDefaultPiperTTSConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ? AND provider = ?", "tts", "piper").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	apiURL := os.Getenv("DEFAULT_PIPER_API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:1232/piper/tts"
	}
	modelPath := os.Getenv("DEFAULT_PIPER_MODEL_PATH")
	if modelPath == "" {
		modelPath = "tts-model/ngochuyen.onnx"
	}
	modelConfigPath := os.Getenv("DEFAULT_PIPER_MODEL_CONFIG_PATH")
	if modelConfigPath == "" {
		modelConfigPath = "tts-model/ngochuyen.onnx.json"
	}
	voice := os.Getenv("DEFAULT_PIPER_VOICE")
	if voice == "" {
		voice = "ngochuyen"
	}
	data := map[string]interface{}{
		"provider":          "piper",
		"api_url":           apiURL,
		"voice":             voice,
		"model_path":        modelPath,
		"model_config_path": modelConfigPath,
		"response_format":   "wav",
		"sample_rate":       22050,
		"frame_duration":    20,
		"timeout":           60,
		"length_scale":      1.0,
		"noise_scale":       0.667,
		"noise_w":           0.8,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "tts",
		Name:      "Piper TTS offline",
		ConfigID:  "piper_offline_default",
		Provider:  "piper",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình Piper TTS offline")
	return nil
}

func ensureDefaultMemoryConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "memory").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data := map[string]interface{}{
		"api_key":          "",
		"base_url":         "https://api.memobase.dev",
		"enable_search":    true,
		"search_threshold": 0.5,
		"search_top_k":     3,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "memory",
		Name:      "Memobase mặc định",
		ConfigID:  "memobase_default",
		Provider:  "memobase",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: false,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình Memory mặc định Memobase")
	return nil
}

func ensureDefaultKnowledgeSearchConfig(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Config{}).Where("type = ?", "knowledge_search").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data := map[string]interface{}{
		"api_key":         "",
		"base_url":        "https://api.dify.ai/v1",
		"score_threshold": 0.2,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cfg := models.Config{
		Type:      "knowledge_search",
		Name:      "Dify mặc định",
		ConfigID:  "dify_default",
		Provider:  "dify",
		JsonData:  string(bytes),
		Enabled:   true,
		IsDefault: true,
	}
	if err := db.Create(&cfg).Error; err != nil {
		return err
	}
	log.Println("Đã tạo cấu hình truy xuất kho tri thức mặc định Dify")
	return nil
}

func repairConfigProviders(db *gorm.DB) error {
	var configs []models.Config
	if err := db.Where("type IN ?", []string{"vad", "asr", "llm", "tts", "memory", "vision"}).Find(&configs).Error; err != nil {
		return err
	}

	repaired := 0
	for _, cfg := range configs {
		var data map[string]interface{}
		if cfg.JsonData != "" {
			if err := json.Unmarshal([]byte(cfg.JsonData), &data); err != nil {
				log.Printf("Bỏ qua sửa provider, phân tích json_data thất bại type=%s config_id=%s: %v", cfg.Type, cfg.ConfigID, err)
				continue
			}
		}
		if data == nil {
			data = map[string]interface{}{}
		}

		provider := configprovider.NormalizeExistingProvider(cfg.Type, cfg.Provider, cfg.ConfigID, data)
		if provider == "" || provider == cfg.Provider {
			if jsonProvider, _ := data["provider"].(string); strings.TrimSpace(jsonProvider) == "" || strings.EqualFold(strings.TrimSpace(jsonProvider), provider) {
				continue
			}
		}

		updates := map[string]interface{}{}
		if provider != "" && provider != cfg.Provider {
			updates["provider"] = provider
		}
		if provider != "" {
			if jsonProvider, _ := data["provider"].(string); !strings.EqualFold(strings.TrimSpace(jsonProvider), provider) {
				data["provider"] = provider
				bytes, err := json.Marshal(data)
				if err != nil {
					return err
				}
				updates["json_data"] = string(bytes)
			}
		}
		if len(updates) == 0 {
			continue
		}
		if err := db.Model(&models.Config{}).Where("id = ?", cfg.ID).Updates(updates).Error; err != nil {
			return err
		}
		repaired++
	}

	if repaired > 0 {
		log.Printf("Đã sửa %d cấu hình provider", repaired)
	}
	return nil
}
