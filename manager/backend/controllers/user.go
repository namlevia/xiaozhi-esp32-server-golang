package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	DB                  *gorm.DB
	InternalAuthToken   string
	EndpointAuthToken   string
	WebSocketController interface {
		RequestMcpToolDetailsFromClient(ctx context.Context, agentID string) ([]MCPTool, error)
		RequestMcpEndpointStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error)
		RequestDeviceMcpToolDetailsFromClient(ctx context.Context, deviceID string) ([]MCPTool, error)
		CallMcpToolFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error)
		RequestOpenClawStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error)
		CallOpenClawChatFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error)
		CallOpenClawChatStreamFromClient(ctx context.Context, body map[string]interface{}, onResponse func(*WebSocketResponse) error) (map[string]interface{}, error)
		InjectMessageToDevice(ctx context.Context, deviceID, message string, skipLlm bool, autoListen bool) error
	}
}

// UserConfigResponse phản hồi cấu hình mà người dùng thường có thể thấy (không gồm json_data và các trường nhạy cảm)
type UserConfigResponse struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	ConfigID  string    `json:"config_id"`
	Provider  string    `json:"provider"`
	Enabled   bool      `json:"enabled"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toUserConfigResponse(cfg *models.Config) *UserConfigResponse {
	if cfg == nil {
		return nil
	}

	return &UserConfigResponse{
		ID:        cfg.ID,
		Type:      cfg.Type,
		Name:      cfg.Name,
		ConfigID:  cfg.ConfigID,
		Provider:  cfg.Provider,
		Enabled:   cfg.Enabled,
		IsDefault: cfg.IsDefault,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	}
}

func toUserConfigResponseList(configs []models.Config) []UserConfigResponse {
	result := make([]UserConfigResponse, 0, len(configs))
	for i := range configs {
		result = append(result, *toUserConfigResponse(&configs[i]))
	}
	return result
}

func normalizeMemoryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "none":
		return "none"
	case "long":
		return "long"
	default:
		return "short"
	}
}

// Đẩy giọng nói đến thiết bị
func (uc *UserController) InjectMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		DeviceID   string `json:"device_id" binding:"required"`
		Message    string `json:"message" binding:"required"`
		SkipLlm    bool   `json:"skip_llm"`
		AutoListen *bool  `json:"auto_listen"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}

	// Xác thực thiết bị có thuộc người dùng hiện tại không
	var device models.Device

	if err := uc.DB.Where("device_name = ? AND user_id = ?", req.DeviceID, userID).First(&device).Error; err != nil {
		log.Printf("[InjectMessage] Truy vấn thiết bị thất bại: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiết bị không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	autoListen := true
	if req.AutoListen != nil {
		autoListen = *req.AutoListen
	}

	// Gửi yêu cầu đẩy giọng nói đến server chính qua WebSocket
	ctx := context.Background()
	err := uc.WebSocketController.InjectMessageToDevice(ctx, device.DeviceName, req.Message, req.SkipLlm, autoListen)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đẩy giọng nói thất bại: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đã gửi yêu cầu đẩy giọng nói",
		"data": gin.H{
			"device_id":   req.DeviceID,
			"message":     req.Message,
			"skip_llm":    req.SkipLlm,
			"auto_listen": autoListen,
		},
	})
}

// Người dùng tạo thiết bị trực tiếp (không cần mã xác minh)
func (uc *UserController) CreateDevice(c *gin.Context) {
	var req DevicePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}
	device, err := NewDeviceService(uc.DB).Create(scopeFromContext(c), req)
	if err != nil {
		writeServiceError(c, err, "Tạo thiết bị thất bại")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Tạo thiết bị thành công",
		"data": gin.H{
			"device_code": device.DeviceCode,
			"device":      device,
		},
	})
}

// Tạo mã số ngẫu nhiên 6 chữ số
func generateRandomCode() string {
	// Tạo số ngẫu nhiên 6 chữ số
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	return code
}

func isSixDigitCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func normalizeDeviceNameCandidate(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "-", ":"))
}

func normalizeDeviceNickName(value string) (string, error) {
	nickName := strings.TrimSpace(value)
	if len([]rune(nickName)) > 50 {
		return "", fmt.Errorf("Biệt danh thiết bị tối đa 50 ký tự")
	}
	return nickName, nil
}

func generateUniqueDeviceCode(db *gorm.DB) string {
	for i := 0; i < 10; i++ { // Thử tối đa 10 lần
		code := generateRandomCode()

		var count int64
		if err := db.Model(&models.Device{}).Where("device_code = ?", code).Count(&count).Error; err == nil && count == 0 {
			return code
		}
	}

	return fmt.Sprintf("%06d", time.Now().Unix()%1000000)
}

// Lấy tổng quan toàn bộ thiết bị của người dùng (chỉ đọc)
func (uc *UserController) GetMyDevices(c *gin.Context) {
	result, err := NewDeviceService(uc.DB).List(scopeFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lấy danh sách thiết bị thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// UpdateDevice cập nhật biệt danh thiết bị của người dùng hiện tại. device_name là định danh phía thiết bị, không sửa ở đây.
func (uc *UserController) UpdateDevice(c *gin.Context) {
	deviceID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req DevicePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}
	device, err := NewDeviceService(uc.DB).Update(scopeFromContext(c), deviceID, req)
	if err != nil {
		writeServiceError(c, err, "Cập nhật thiết bị thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": device})
}

// DeleteDevice xóa thiết bị của người dùng hiện tại khỏi hệ thống. Sau khi xóa, thiết bị cần kích hoạt lại để vào hệ thống.
func (uc *UserController) DeleteDevice(c *gin.Context) {
	deviceID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := NewDeviceService(uc.DB).Delete(scopeFromContext(c), deviceID); err != nil {
		writeServiceError(c, err, "Xóa thiết bị thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Thiết bị đã bị xóa khỏi hệ thống, cần kích hoạt lại trước khi sử dụng tiếp"})
}

// Quản lý trợ lý
func (uc *UserController) GetAgents(c *gin.Context) {
	result, err := NewAgentService(uc.DB).List(scopeFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lấy danh sách trợ lý thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (uc *UserController) CreateAgent(c *gin.Context) {
	var req AgentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu"})
		return
	}
	agent, err := NewAgentService(uc.DB).Create(scopeFromContext(c), req)
	if err != nil {
		writeServiceError(c, err, "Tạo trợ lý thất bại")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"agent": agent, "knowledge_base_ids": agent.KnowledgeBaseIDs}})
}

func (uc *UserController) GetAgent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	result, err := NewAgentService(uc.DB).Get(scopeFromContext(c), id)
	if err != nil {
		writeServiceError(c, err, "Lấy trợ lý thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (uc *UserController) UpdateAgent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req AgentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu"})
		return
	}
	agent, err := NewAgentService(uc.DB).Update(scopeFromContext(c), id, req)
	if err != nil {
		writeServiceError(c, err, "Cập nhật trợ lý thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"agent": agent, "knowledge_base_ids": agent.KnowledgeBaseIDs}})
}

func (uc *UserController) DeleteAgent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := NewAgentService(uc.DB).Delete(scopeFromContext(c), id); err != nil {
		writeServiceError(c, err, "Xóa trợ lý thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Xóa thành công"})
}

// Lấy thiết bị liên kết với trợ lý
func (uc *UserController) GetAgentDevices(c *gin.Context) {
	agentID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	devices, err := NewDeviceService(uc.DB).ListByAgent(scopeFromContext(c), agentID)
	if err != nil {
		writeServiceError(c, err, "Lấy danh sách thiết bị thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": devices})
}

// Thêm thiết bị vào trợ lý
func (uc *UserController) AddDeviceToAgent(c *gin.Context) {
	agentID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req DevicePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu"})
		return
	}
	device, err := NewDeviceService(uc.DB).BindToAgent(scopeFromContext(c), agentID, req)
	if err != nil {
		writeServiceError(c, err, "Liên kết thiết bị thất bại")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": device})
}

// Gỡ thiết bị khỏi trợ lý
func (uc *UserController) RemoveDeviceFromAgent(c *gin.Context) {
	agentID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	deviceID, ok := parseUintParam(c, "device_id")
	if !ok {
		return
	}
	if err := NewDeviceService(uc.DB).UnbindFromAgent(scopeFromContext(c), agentID, deviceID); err != nil {
		writeServiceError(c, err, "Gỡ thiết bị thất bại")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Gỡ thiết bị thành công"})
}

// Lấy mẫu vai trò
func (uc *UserController) GetRoleTemplates(c *gin.Context) {
	var roles []models.GlobalRole
	if err := uc.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lấy mẫu vai trò thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func trimSuffixFoldForURL(s string, suffix string) string {
	if len(s) < len(suffix) {
		return s
	}
	start := len(s) - len(suffix)
	if strings.EqualFold(s[start:], suffix) {
		return s[:start]
	}
	return s
}

func normalizeIndexTTSVoiceOptionsBaseURL(raw string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	baseURL = trimSuffixFoldForURL(baseURL, "/audio/speech")
	baseURL = trimSuffixFoldForURL(baseURL, "/audio/voices")
	return strings.TrimRight(baseURL, "/")
}

func normalizePiperVoiceOptionsURL(raw string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	baseURL = trimSuffixFoldForURL(baseURL, "/piper/tts")
	baseURL = trimSuffixFoldForURL(baseURL, "/piper/voices")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:1232"
	}
	return baseURL + "/piper/voices"
}

func (uc *UserController) fetchPiperVoices(c *gin.Context, configID, overrideURL string) ([]VoiceOption, error) {
	apiURL := "http://127.0.0.1:1232/piper/tts"
	if strings.TrimSpace(configID) != "" {
		var cfg models.Config
		if err := uc.DB.Where("type = ? AND config_id = ?", "tts", configID).First(&cfg).Error; err == nil {
			var cfgMap map[string]any
			if strings.TrimSpace(cfg.JsonData) != "" && json.Unmarshal([]byte(cfg.JsonData), &cfgMap) == nil {
				if v, ok := cfgMap["api_url"].(string); ok && strings.TrimSpace(v) != "" {
					apiURL = strings.TrimSpace(v)
				}
			}
		}
	}
	if strings.TrimSpace(overrideURL) != "" {
		apiURL = strings.TrimSpace(overrideURL)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, normalizePiperVoiceOptionsURL(apiURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Piper lấy giọng thất bại: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Voices []struct {
			ID              string  `json:"id"`
			Name            string  `json:"name"`
			ModelPath       string  `json:"model_path"`
			ModelConfigPath string  `json:"model_config_path"`
			SampleRate      int     `json:"sample_rate"`
			Language        string  `json:"language"`
			LengthScale     float32 `json:"length_scale"`
			NoiseScale      float32 `json:"noise_scale"`
			NoiseW          float32 `json:"noise_w"`
		} `json:"voices"`
	}
	if err = json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	result := make([]VoiceOption, 0, len(payload.Voices))
	for _, voice := range payload.Voices {
		value := strings.TrimSpace(voice.ID)
		if value == "" {
			value = strings.TrimSpace(voice.Name)
		}
		if value == "" {
			continue
		}
		label := strings.TrimSpace(voice.Name)
		if label == "" {
			label = value
		}
		if voice.Language != "" {
			label = fmt.Sprintf("%s (%s)", label, voice.Language)
		}
		result = append(result, VoiceOption{
			Value:           value,
			Label:           label,
			ModelPath:       voice.ModelPath,
			ModelConfigPath: voice.ModelConfigPath,
			SampleRate:      voice.SampleRate,
			Language:        voice.Language,
			LengthScale:     voice.LengthScale,
			NoiseScale:      voice.NoiseScale,
			NoiseW:          voice.NoiseW,
		})
	}
	return result, nil
}

func (uc *UserController) fetchIndexTTSVoices(c *gin.Context, configID, overrideURL, overrideAPIKey string) ([]VoiceOption, error) {
	baseURL := "http://127.0.0.1:7860"
	apiKey := ""
	if strings.TrimSpace(configID) != "" {
		var cfg models.Config
		if err := uc.DB.Where("type = ? AND config_id = ?", "tts", configID).First(&cfg).Error; err == nil {
			var cfgMap map[string]any
			if strings.TrimSpace(cfg.JsonData) != "" && json.Unmarshal([]byte(cfg.JsonData), &cfgMap) == nil {
				if v, ok := cfgMap["api_url"].(string); ok && strings.TrimSpace(v) != "" {
					baseURL = strings.TrimSpace(v)
				}
				if v, ok := cfgMap["api_key"].(string); ok {
					apiKey = strings.TrimSpace(v)
				}
			}
		}
	}
	if strings.TrimSpace(overrideURL) != "" {
		baseURL = strings.TrimSpace(overrideURL)
	}
	if strings.TrimSpace(overrideAPIKey) != "" {
		apiKey = strings.TrimSpace(overrideAPIKey)
	}
	baseURL = normalizeIndexTTSVoiceOptionsBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:7860"
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, baseURL+indexTTSVoicesEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("IndexTTS lấy giọng thất bại: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	voiceMap := map[string]any{}
	if err = json.Unmarshal(body, &voiceMap); err != nil {
		return nil, err
	}
	result := make([]VoiceOption, 0, len(voiceMap))
	normalizedConfigPrefix := strings.ToLower(strings.TrimSpace(configID))
	if normalizedConfigPrefix != "" {
		normalizedConfigPrefix += "_"
	}
	for voice := range voiceMap {
		v := strings.TrimSpace(voice)
		if v == "" {
			continue
		}
		// Lọc giọng có tiền tố nội bộ do instance cấu hình IndexTTS hiện tại tạo ra để tránh trùng với giọng clone.
		if normalizedConfigPrefix != "" && strings.HasPrefix(strings.ToLower(v), normalizedConfigPrefix) {
			continue
		}
		result = append(result, VoiceOption{Value: v, Label: v})
	}
	return result, nil
}

// Lấy tùy chọn giọng
func (uc *UserController) GetVoiceOptions(c *gin.Context) {
	scope := scopeFromContext(c)
	voices, err := getVoiceOptionsForUser(
		uc.DB,
		c,
		scope.ActorUserID,
		c.Query("provider"),
		c.Query("config_id"),
		c.Query("api_url"),
		c.Query("api_key"),
	)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "IndexTTS") {
			status = http.StatusBadGateway
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": voices})
}

// Lấy danh sách cấu hình LLM
func (uc *UserController) GetLLMConfigs(c *gin.Context) {
	var configs []models.Config
	// Lấy toàn bộ cấu hình LLM đã bật từ cấu hình toàn cục, cấu hình mặc định xếp trước
	if err := uc.DB.Where("type = ? AND enabled = ?", "llm", true).Order("is_default DESC, name ASC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lấy cấu hình LLM thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toUserConfigResponseList(configs)})
}

// Lấy danh sách cấu hình TTS
func (uc *UserController) GetTTSConfigs(c *gin.Context) {
	var configs []models.Config
	// Lấy toàn bộ cấu hình TTS đã bật từ cấu hình toàn cục, cấu hình mặc định xếp trước
	if err := uc.DB.Where("type = ? AND enabled = ?", "tts", true).Order("is_default DESC, name ASC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lấy cấu hình TTS thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toUserConfigResponseList(configs)})
}

// GetDeviceMcpTools lấy danh sách công cụ MCP theo thiết bị (bản người dùng)
func (uc *UserController) GetDeviceMcpTools(c *gin.Context) {
	userID, _ := c.Get("user_id")
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số device_id là bắt buộc"})
		return
	}

	var device models.Device
	if err := uc.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thiết bị không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	tools, err := uc.WebSocketController.RequestDeviceMcpToolDetailsFromClient(context.Background(), device.DeviceName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": tools}})
}

// CallAgentMcpTool gọi công cụ MCP theo trợ lý (bản người dùng)
func (uc *UserController) CallAgentMcpTool(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID := c.Param("id")

	var req struct {
		ToolName  string                 `json:"tool_name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}

	var agent models.Agent
	if err := uc.DB.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trợ lý không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	body := map[string]interface{}{
		"agent_id":  agentID,
		"tool_name": req.ToolName,
		"arguments": req.Arguments,
	}
	result, err := uc.WebSocketController.CallMcpToolFromClient(context.Background(), body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gọi công cụ MCP thất bại: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (uc *UserController) GetAgentMCPServiceOptions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var agent models.Agent
	if err := uc.DB.Where("id = ? AND user_id = ?", id, userID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trợ lý không tồn tại"})
		return
	}

	options, err := listEnabledGlobalMCPServiceNames(uc.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Lấy tùy chọn dịch vụ MCP thất bại: %v", err)})
		return
	}

	normalized := normalizeMCPServiceNamesCSV(agent.MCPServiceNames)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"options":           options,
		"selected":          splitMCPServiceNames(normalized),
		"mcp_service_names": normalized,
	}})
}

func (uc *UserController) GetMCPServiceOptions(c *gin.Context) {
	options, err := listEnabledGlobalMCPServiceNames(uc.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Lấy tùy chọn dịch vụ MCP thất bại: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"options": options,
	}})
}

// CallDeviceMcpTool gọi công cụ MCP theo thiết bị (bản người dùng)
func (uc *UserController) CallDeviceMcpTool(c *gin.Context) {
	userID, _ := c.Get("user_id")
	deviceID := c.Param("id")

	var req struct {
		ToolName  string                 `json:"tool_name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}

	var device models.Device
	if err := uc.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thiết bị không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	body := map[string]interface{}{
		"device_id": device.DeviceName,
		"tool_name": req.ToolName,
		"arguments": req.Arguments,
	}
	result, err := uc.WebSocketController.CallMcpToolFromClient(context.Background(), body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gọi công cụ MCP thất bại: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetAgentMCPEndpoint lấy URL endpoint MCP của trợ lý (bản người dùng)
func (uc *UserController) GetAgentMCPEndpoint(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số agent_id là bắt buộc"})
		return
	}

	// Xác thực trợ lý có tồn tại và thuộc người dùng hiện tại không
	var agent models.Agent
	if err := uc.DB.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trợ lý không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	// Dùng hàm chung để tạo endpoint MCP
	endpoint, err := GenerateAgentMCPEndpoint(uc.DB, agentID, userID.(uint), uc.EndpointAuthToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	data := gin.H{
		"endpoint":    endpoint,
		"status":      "unknown",
		"connected":   false,
		"tools_count": 0,
	}
	if uc.WebSocketController == nil {
		data["status_message"] = "Bộ điều khiển websocket không khả dụng"
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	statusResult, statusErr := uc.WebSocketController.RequestMcpEndpointStatusFromClient(context.Background(), agentID)
	if statusErr != nil {
		data["status_message"] = statusErr.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	connected, _ := statusResult["connected"].(bool)
	status, _ := statusResult["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		if connected {
			status = "online"
		} else {
			status = "offline"
		}
	}

	data["connected"] = connected
	data["status"] = status
	if clientCount, ok := statusResult["client_count"]; ok {
		data["client_count"] = clientCount
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetAgentOpenClawEndpoint lấy URL endpoint OpenClaw của trợ lý (bản người dùng)
func (uc *UserController) GetAgentOpenClawEndpoint(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số agent_id là bắt buộc"})
		return
	}

	var agent models.Agent
	if err := uc.DB.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trợ lý không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	data := gin.H{
		"endpoint":  "",
		"status":    "unknown",
		"connected": false,
	}

	endpoint, err := GenerateAgentOpenClawEndpoint(uc.DB, agentID, userID.(uint), uc.EndpointAuthToken)
	if err != nil {
		data["status_message"] = err.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	data["endpoint"] = endpoint

	if uc.WebSocketController == nil {
		data["status_message"] = "Bộ điều khiển websocket không khả dụng"
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	statusResult, statusErr := uc.WebSocketController.RequestOpenClawStatusFromClient(context.Background(), agentID)
	if statusErr != nil {
		data["status_message"] = statusErr.Error()
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	connected, _ := statusResult["connected"].(bool)
	status, _ := statusResult["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		if connected {
			status = "online"
		} else {
			status = "offline"
		}
	}

	data["connected"] = connected
	data["status"] = status
	if msg, ok := statusResult["status_message"].(string); ok && strings.TrimSpace(msg) != "" {
		data["status_message"] = msg
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// CallAgentOpenClawChatTest gọi kiểm tra hội thoại OpenClaw của trợ lý (bản người dùng)
func (uc *UserController) CallAgentOpenClawChatTest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số agent_id là bắt buộc"})
		return
	}
	if uc.WebSocketController == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Bộ điều khiển websocket không khả dụng"})
		return
	}

	var req struct {
		Message   string `json:"message" binding:"required"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số yêu cầu: " + err.Error()})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message không được để trống"})
		return
	}

	var agent models.Agent
	if err := uc.DB.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trợ lý không tồn tại hoặc không thuộc người dùng hiện tại"})
		return
	}

	body := map[string]interface{}{
		"agent_id": agentID,
		"message":  req.Message,
	}
	if req.TimeoutMs > 0 {
		body["timeout_ms"] = req.TimeoutMs
	}

	if wantsOpenClawSSE(c) {
		if !prepareOpenClawSSE(c) {
			return
		}
		_ = writeOpenClawSSE(c, "start", map[string]interface{}{
			"agent_id": agentID,
		})

		terminalErrorSent := false
		result, err := uc.WebSocketController.CallOpenClawChatStreamFromClient(
			c.Request.Context(),
			body,
			func(resp *WebSocketResponse) error {
				if resp == nil {
					return nil
				}
				payload := map[string]interface{}{
					"status": resp.Status,
				}
				if resp.Body != nil {
					payload["data"] = resp.Body
				}
				if msg := strings.TrimSpace(resp.Error); msg != "" {
					payload["error"] = msg
				}

				switch resp.Status {
				case http.StatusPartialContent:
					return writeOpenClawSSE(c, "chunk", payload)
				case http.StatusOK:
					return writeOpenClawSSE(c, "result", payload)
				default:
					terminalErrorSent = true
					return writeOpenClawSSE(c, "error", payload)
				}
			},
		)
		if err != nil {
			if !terminalErrorSent {
				_ = writeOpenClawSSE(c, "error", map[string]interface{}{
					"error": err.Error(),
				})
			}
			_ = writeOpenClawSSE(c, "done", map[string]interface{}{
				"ok": false,
			})
			return
		}

		_ = writeOpenClawSSE(c, "done", map[string]interface{}{
			"ok":   true,
			"data": result,
		})
		return
	}

	result, err := uc.WebSocketController.CallOpenClawChatFromClient(context.Background(), body)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(strings.ToLower(msg), "not connected"), strings.Contains(msg, "chưa kết nối"):
			c.JSON(http.StatusConflict, gin.H{"error": msg})
		case strings.Contains(strings.ToLower(msg), "timeout"), strings.Contains(msg, "quá thời gian"):
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": msg})
		case strings.Contains(strings.ToLower(msg), "missing"), strings.Contains(msg, "tham số"):
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		case strings.Contains(msg, "không có client đã kết nối"):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": msg})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kiểm tra hội thoại OpenClaw thất bại: " + msg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetAgentMcpTools lấy danh sách công cụ MCP của trợ lý (bản người dùng)
func (uc *UserController) GetAgentMcpTools(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID := c.Param("id")

	// Hàm xác thực người dùng: kiểm tra trợ lý có tồn tại và thuộc người dùng hiện tại không
	userAgentValidator := func(agentID string) error {
		var agent models.Agent
		if err := uc.DB.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
			return fmt.Errorf("Trợ lý không tồn tại hoặc không thuộc người dùng hiện tại")
		}
		return nil
	}

	// Dùng hàm chung
	GetAgentMcpToolsCommon(c, agentID, uc.WebSocketController, userAgentValidator)
}

// Lấy dữ liệu thống kê bảng điều khiển
func (uc *UserController) GetDashboardStats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	type DashboardStats struct {
		TotalUsers       int64  `json:"totalUsers"`
		TotalDevices     int64  `json:"totalDevices"`
		TotalAgents      int64  `json:"totalAgents"`
		OnlineDevices    int64  `json:"onlineDevices"`
		ProgramStartedAt string `json:"programStartedAt"`
	}

	stats := DashboardStats{
		ProgramStartedAt: programStartedAt.Format(time.RFC3339),
	}

	if userRole == "admin" {
		// Quản trị viên xem toàn bộ dữ liệu
		uc.DB.Model(&models.User{}).Count(&stats.TotalUsers)
		uc.DB.Model(&models.Device{}).Count(&stats.TotalDevices)
		uc.DB.Model(&models.Agent{}).Count(&stats.TotalAgents)
		// Thiết bị online: thiết bị hoạt động trong 5 phút gần nhất
		fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
		uc.DB.Model(&models.Device{}).Where("last_active_at > ?", fiveMinutesAgo).Count(&stats.OnlineDevices)
	} else {
		// Người dùng thường chỉ xem dữ liệu của mình
		stats.TotalUsers = 0 // Người dùng thường không hiển thị số người dùng
		uc.DB.Model(&models.Device{}).Where("user_id = ?", userID).Count(&stats.TotalDevices)
		uc.DB.Model(&models.Agent{}).Where("user_id = ?", userID).Count(&stats.TotalAgents)
		// Thiết bị online: thiết bị của người dùng hoạt động trong 5 phút gần nhất
		fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
		uc.DB.Model(&models.Device{}).Where("user_id = ? AND last_active_at > ?", userID, fiveMinutesAgo).Count(&stats.OnlineDevices)
	}

	c.JSON(http.StatusOK, stats)
}
