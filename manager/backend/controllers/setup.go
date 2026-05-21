package controllers

import (
	"log"
	"net/http"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SetupController struct {
	DB *gorm.DB
}

type SetupRequest struct {
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=50"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=100"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
}

// Kiểm tra cơ sở dữ liệu có cần khởi tạo hay không.
func (sc *SetupController) CheckSetupStatus(c *gin.Context) {
	if sc.DB == nil {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Kết nối cơ sở dữ liệu không khả dụng",
		})
		return
	}

	// Kiểm tra bảng người dùng có tồn tại hay không.
	if !sc.DB.Migrator().HasTable(&models.User{}) {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Cấu trúc bảng cơ sở dữ liệu chưa được khởi tạo",
		})
		return
	}

	// Kiểm tra người dùng quản trị có tồn tại hay không.
	var count int64
	sc.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Cần tạo tài khoản quản trị viên",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"needs_setup": false,
		"message":     "Hệ thống đã được khởi tạo",
	})
}

// Khởi tạo cơ sở dữ liệu.
func (sc *SetupController) InitializeDatabase(c *gin.Context) {
	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if sc.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kết nối cơ sở dữ liệu không khả dụng"})
		return
	}

	// Bắt đầu transaction.
	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Khởi tạo transaction cơ sở dữ liệu thất bại"})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Tự động migrate cấu trúc bảng.
	log.Println("Bắt đầu tự động migrate cấu trúc bảng cơ sở dữ liệu...")
	err := tx.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.Agent{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.ChatMessage{},
	)
	if err != nil {
		tx.Rollback()
		log.Printf("Migration cấu trúc bảng cơ sở dữ liệu thất bại: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration cấu trúc bảng cơ sở dữ liệu thất bại: " + err.Error()})
		return
	}
	log.Println("Migration cấu trúc bảng cơ sở dữ liệu thành công")

	// 2. Kiểm tra người dùng quản trị đã tồn tại hay chưa.
	var existingAdmin models.User
	if err := tx.Where("role = ?", "admin").First(&existingAdmin).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người dùng quản trị đã tồn tại, không thể khởi tạo lại"})
		return
	}

	// 3. Kiểm tra tên đăng nhập đã tồn tại hay chưa.
	var existingUser models.User
	if err := tx.Where("username = ?", req.AdminUsername).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên đăng nhập đã tồn tại"})
		return
	}

	// 4. Kiểm tra email đã tồn tại hay chưa.
	if err := tx.Where("email = ?", req.AdminEmail).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã tồn tại"})
		return
	}

	// 5. Mã hóa mật khẩu.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mã hóa mật khẩu thất bại"})
		return
	}

	// 6. Tạo người dùng quản trị.
	admin := models.User{
		Username: req.AdminUsername,
		Password: string(hashedPassword),
		Email:    req.AdminEmail,
		Role:     "admin",
	}

	if err := tx.Create(&admin).Error; err != nil {
		tx.Rollback()
		log.Printf("Tạo người dùng quản trị thất bại: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tạo người dùng quản trị thất bại: " + err.Error()})
		return
	}

	// 7. Tạo một số vai trò toàn cục mặc định
	defaultRoles := []models.Role{
		{
			Name:        "Trợ lý",
			Description: "Một trợ lý AI thân thiện, có thể giúp người dùng giải quyết nhiều vấn đề khác nhau",
			Prompt:      "Bạn là một trợ lý AI thân thiện và chuyên nghiệp. Hãy trả lời câu hỏi của người dùng bằng ngôn ngữ rõ ràng, súc tích và đưa ra gợi ý hữu ích.",
			RoleType:    "global",
			Status:      "active",
			IsDefault:   true,
		},
		{
			Name:        "Giáo viên",
			Description: "Một người thầy kiên nhẫn, có thể giải thích chi tiết các khái niệm phức tạp",
			Prompt:      "Bạn là một giáo viên giàu kinh nghiệm. Hãy giải thích các khái niệm phức tạp theo cách dễ hiểu và đưa ra ví dụ cụ thể để hỗ trợ người dùng nắm bắt.",
			RoleType:    "global",
			Status:      "active",
			IsDefault:   false,
		},
		{
			Name:        "Bạn đồng hành",
			Description: "Một người bạn tinh tế, biết lắng nghe và đồng hành cùng người dùng",
			Prompt:      "Bạn là một người bạn tinh tế. Hãy trò chuyện với thái độ ấm áp, thấu hiểu và mang lại sự động viên cho người dùng.",
			RoleType:    "global",
			Status:      "active",
			IsDefault:   false,
		},
	}

	for _, role := range defaultRoles {
		if err := tx.Create(&role).Error; err != nil {
			log.Printf("Tạo vai trò mặc định thất bại: %v", err)
			// Không ngắt quá trình khởi tạo, tiếp tục xử lý
		}
	}

	// 8. Tạo cấu hình ASR tiếng Việt mặc định.
	defaultASRConfig := models.Config{
		Type:      "asr",
		Name:      "Vietnamese ASR (Go)",
		ConfigID:  "wyoming_vietnamese_asr_default",
		Provider:  "wyoming_vietnamese_asr",
		JsonData:  `{"base_url":"http://127.0.0.1:1231","sample_rate":16000,"timeout_ms":30000}`,
		Enabled:   true,
		IsDefault: true,
	}
	if err := tx.Where("type = ? AND config_id = ?", defaultASRConfig.Type, defaultASRConfig.ConfigID).FirstOrCreate(&defaultASRConfig).Error; err != nil {
		log.Printf("Tạo cấu hình ASR mặc định thất bại: %v", err)
	}

	// Commit transaction.
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit transaction cơ sở dữ liệu thất bại"})
		return
	}

	log.Printf("Khởi tạo cơ sở dữ liệu thành công, người dùng quản trị: %s", req.AdminUsername)
	c.JSON(http.StatusOK, gin.H{
		"message": "Khởi tạo cơ sở dữ liệu thành công",
		"admin": gin.H{
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
		},
	})
}
