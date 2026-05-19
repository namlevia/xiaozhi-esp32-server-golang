package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
	"gorm.io/gorm"

	"xiaozhi/manager/backend/models"
)

type WebSocketController struct {
	DB                *gorm.DB
	endpointAuthToken string
	upgrader          websocket.Upgrader
	clientsMap        cmap.ConcurrentMap[string, *WebSocketClient]
}

type WSClientClaims struct {
	Purpose string `json:"purpose"`
	UUID    string `json:"uuid"`
	jwt.RegisteredClaims
}

// WebSocketClient là client kết nối tới Manager Backend.
type WebSocketClient struct {
	ID           string
	conn         *websocket.Conn
	controller   *WebSocketController
	requestChans map[string]chan *WebSocketResponse
	callbacks    map[string]func(*WebSocketResponse)
	mu           sync.RWMutex
	isConnected  bool
	stopChan     chan struct{} // Kênh tín hiệu dừng
}

type WebSocketRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
}

type WebSocketResponse struct {
	ID      string                 `json:"id"`
	Status  int                    `json:"status"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      bool                   `json:"schema"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

const (
	defaultBroadcastRequestTimeout = 30 * time.Second
	defaultMcpStatusRequestTimeout = 3 * time.Second
	openClawChatDefaultTimeoutMs   = 10 * 60 * 1000
	openClawChatMinTimeoutMs       = 1000
	openClawChatMaxTimeoutMs       = 10 * 60 * 1000
)

// NewWebSocketController tạo WebSocket controller.
func NewWebSocketController(db *gorm.DB, endpointAuthToken string) *WebSocketController {
	return &WebSocketController{
		DB:                db,
		endpointAuthToken: strings.TrimSpace(endpointAuthToken),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Cho phép mọi origin; môi trường production nên giới hạn.
			},
		},
		clientsMap: cmap.New[*WebSocketClient](),
	}
}

// HandleWebSocket xử lý nâng cấp kết nối WebSocket.
func (ctrl *WebSocketController) HandleWebSocket(c *gin.Context) {
	tokenString := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = strings.TrimSpace(tokenString[7:])
	}
	if tokenString == "" {
		tokenString = strings.TrimSpace(c.Query("token"))
	}
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực WebSocket"})
		return
	}

	claims, err := ctrl.parseWSClientToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token xác thực WebSocket không hợp lệ"})
		return
	}
	if claims.Purpose != "manager-ws-client" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mục đích sử dụng WebSocket token không hợp lệ"})
		return
	}

	// Lấy header UUID.
	clientUUID := c.GetHeader("UUID")
	if clientUUID == "" {
		log.Printf("Kết nối WebSocket thiếu header UUID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu header UUID"})
		return
	}
	if strings.TrimSpace(claims.UUID) != "" && strings.TrimSpace(claims.UUID) != strings.TrimSpace(clientUUID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "UUID không khớp với token"})
		return
	}

	// Nâng cấp kết nối HTTP thành WebSocket.
	conn, err := ctrl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Nâng cấp WebSocket thất bại: %v", err)
		return
	}

	// Kiểm tra đã có kết nối cùng UUID hay chưa.
	if existingClient, exists := ctrl.clientsMap.Get(clientUUID); exists {
		log.Printf("Ngắt kết nối hiện có: %s", clientUUID)
		existingClient.conn.Close()
		existingClient.isConnected = false
	}

	// Tạo client mới.
	client := &WebSocketClient{
		ID:           clientUUID,
		conn:         conn,
		controller:   ctrl,
		requestChans: make(map[string]chan *WebSocketResponse),
		callbacks:    make(map[string]func(*WebSocketResponse)),
		isConnected:  true,
		stopChan:     make(chan struct{}),
	}

	// Lưu vào clientsMap.
	ctrl.clientsMap.Set(clientUUID, client)

	log.Printf("WebSocketClient mới đã kết nối: %s", clientUUID)

	// Khởi động xử lý message client.
	go client.handleMessages()

	// Khởi động kiểm tra heartbeat.
	go client.heartbeat()
}

func (ctrl *WebSocketController) parseWSClientToken(tokenString string) (*WSClientClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &WSClientClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(ctrl.endpointAuthToken), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*WSClientClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}
	return claims, nil
}

// Xóa client.
func (ctrl *WebSocketController) removeClient(clientID string) {
	if client, exists := ctrl.clientsMap.Get(clientID); exists {
		// Gửi tín hiệu dừng cho heartbeat.
		select {
		case client.stopChan <- struct{}{}:
			log.Printf("Đã gửi tín hiệu dừng cho client: %s", clientID)
		default:
			// Channel có thể đầy hoặc đã đóng, bỏ qua.
		}

		// Đảm bảo trạng thái client được đặt đúng.
		client.isConnected = false
		// Xóa khỏi map.
		ctrl.clientsMap.Remove(clientID)
		log.Printf("WebSocket client đã ngắt kết nối: %s", clientID)
	}
}

// Lấy client theo UUID.
func (ctrl *WebSocketController) GetClient(uuid string) *WebSocketClient {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return client
	}
	return nil
}

// Kiểm tra client UUID chỉ định có đang kết nối hay không.
func (ctrl *WebSocketController) IsClientConnected(uuid string) bool {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return client.isConnected
	}
	return false
}

// GetFirstConnectedClientUUID trả về UUID của client đang kết nối đầu tiên để kiểm tra cấu hình.
func (ctrl *WebSocketController) GetFirstConnectedClientUUID() string {
	for item := range ctrl.clientsMap.IterBuffered() {
		if client := item.Val; client.isConnected {
			return client.ID
		}
	}
	return ""
}

// Gửi message tới client có UUID chỉ định.
func (ctrl *WebSocketController) SendToClient(uuid string, message interface{}) error {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.conn.WriteJSON(message)
	}
	return fmt.Errorf("Client %s chưa kết nối", uuid)
}

// Broadcast message tới mọi client đang kết nối.
func (ctrl *WebSocketController) Broadcast(message interface{}) {
	for item := range ctrl.clientsMap.IterBuffered() {
		if client := item.Val; client.isConnected {
			if err := client.conn.WriteJSON(message); err != nil {
				log.Printf("Broadcast message tới client %s thất bại: %v", client.ID, err)
			}
		}
	}
}

// BroadcastSystemConfig push thay đổi cấu hình hệ thống tới mọi client đang kết nối.
func (ctrl *WebSocketController) BroadcastSystemConfig(data gin.H) {
	ctrl.Broadcast(gin.H{"type": "system_config", "data": data})
}

// Xử lý message client.
func (client *WebSocketClient) handleMessages() {
	defer func() {
		client.conn.Close()
		client.isConnected = false
		client.controller.removeClient(client.ID)
	}()

	for {
		if !client.isConnected {
			return
		}

		// Đọc loại message.
		messageType, reader, err := client.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Đọc WebSocket lỗi: %v", err)
			}
			return
		}

		// Xử lý các loại message khác nhau.
		switch messageType {
		case websocket.TextMessage:
			// Xử lý message JSON.
			var rawMessage map[string]interface{}
			if err := json.NewDecoder(reader).Decode(&rawMessage); err != nil {
				log.Printf("Phân tích message JSON thất bại: %v", err)
				continue
			}
			// Xử lý message.
			client.handleMessage(rawMessage)

		case websocket.PingMessage:
			// Xử lý ping và tự động trả pong.
			log.Printf("Nhận ping, tự động trả pong")
			if err := client.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Printf("Gửi pong thất bại: %v", err)
			}

		case websocket.PongMessage:
			// Xử lý pong.
			log.Printf("Nhận pong")

		case websocket.CloseMessage:
			// Xử lý close message.
			log.Printf("Nhận close message")
			return

		default:
			log.Printf("Nhận message WebSocket không rõ loại: %d", messageType)
		}
	}
}

// Xử lý message nhận được.
func (client *WebSocketClient) handleMessage(rawMessage map[string]interface{}) {
	// Kiểm tra có phải message yêu cầu hay không.
	if method, exists := rawMessage["method"]; exists && method != nil {
		client.handleRequest(rawMessage)
		return
	}

	// Kiểm tra có phải message phản hồi hay không.
	if status, exists := rawMessage["status"]; exists && status != nil {
		client.handleResponse(rawMessage)
		return
	}

	log.Printf("Nhận message không nhận diện được: %+v", rawMessage)
}

// Xử lý message yêu cầu.
func (client *WebSocketClient) handleRequest(rawMessage map[string]interface{}) {
	var request WebSocketRequest
	if err := mapToStruct(rawMessage, &request); err != nil {
		log.Printf("Phân tích yêu cầu thất bại: %v", err)
		return
	}

	log.Printf("Nhận yêu cầu: ID=%s, Method=%s, Path=%s", request.ID, request.Method, request.Path)

	// Xử lý yêu cầu và gửi phản hồi.
	client.processRequest(&request)
}

// Xử lý message phản hồi.
func (client *WebSocketClient) handleResponse(rawMessage map[string]interface{}) {
	var response WebSocketResponse
	if err := mapToStruct(rawMessage, &response); err != nil {
		log.Printf("Phân tích phản hồi thất bại: %v", err)
		return
	}

	log.Printf("Nhận phản hồi: ID=%s, Status=%d", response.ID, response.Status)

	// Tìm channel phản hồi tương ứng.
	client.mu.RLock()
	responseChan, exists := client.requestChans[response.ID]
	callback, callbackExists := client.callbacks[response.ID]
	client.mu.RUnlock()

	if exists {
		select {
		case responseChan <- &response:
		default:
			log.Printf("Channel phản hồi đã đầy, bỏ phản hồi: %s", response.ID)
		}
	}

	if callbackExists {
		go callback(&response)
	}

	if !exists && !callbackExists {
		log.Printf("Nhận ID phản hồi không xác định: %s", response.ID)
	}
}

// Xử lý yêu cầu.
func (client *WebSocketClient) processRequest(request *WebSocketRequest) {
	switch request.Path {
	case "/api/server/info":
		client.handleServerInfoRequest(request)

	case "/api/server/ping":
		client.handlePingRequest(request)

	case "/api/device/active":
		client.handleDeviceActiveRequest(request)

	case "/api/device/inactive":
		client.handleDeviceInactiveRequest(request)

	default:
		log.Printf("Path yêu cầu không xác định: %s", request.Path)
		client.sendResponse(request.ID, 404, nil, "Unknown endpoint")
	}
}

// Xử lý yêu cầu thông tin server.
func (client *WebSocketClient) handleServerInfoRequest(request *WebSocketRequest) {
	response := map[string]interface{}{
		"server_name": "xiaozhi-manager-backend",
		"version":     "1.0.0",
		"uptime":      time.Now().Format(time.RFC3339),
		"request_id":  request.ID,
		"client_id":   client.ID,
	}

	client.sendResponse(request.ID, 200, response, "")
}

// Xử lý yêu cầu ping.
func (client *WebSocketClient) handlePingRequest(request *WebSocketRequest) {
	response := map[string]interface{}{
		"message":   "pong from manager backend",
		"time":      time.Now().Format(time.RFC3339),
		"client_id": client.ID,
	}

	client.sendResponse(request.ID, 200, response, "")
}

// Xử lý yêu cầu cập nhật thời gian hoạt động thiết bị.
func (client *WebSocketClient) handleDeviceActiveRequest(request *WebSocketRequest) {
	// Lấy device_id từ body yêu cầu.
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if deviceID == "" {
		log.Printf("Nhận yêu cầu thiết bị active nhưng thiếu device_id")
		client.sendResponse(request.ID, 400, nil, "Thiếu tham số device_id")
		return
	}

	log.Printf("Xử lý yêu cầu cập nhật thời gian hoạt động thiết bị, device_id: %s", deviceID)

	// Cập nhật thời gian hoạt động cuối của thiết bị.
	now := time.Now()
	result := client.controller.DB.Model(&models.Device{}).
		Where("device_name = ?", deviceID).
		Update("last_active_at", now)

	if result.Error != nil {
		log.Printf("Cập nhật thời gian hoạt động thiết bị thất bại: %v", result.Error)
		client.sendResponse(request.ID, 500, nil, fmt.Sprintf("Cập nhật thời gian hoạt động thiết bị thất bại: %v", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("Thiết bị không tồn tại: %s", deviceID)
		client.sendResponse(request.ID, 404, nil, "Thiết bị không tồn tại")
		return
	}

	// Dựng phản hồi thành công.
	response := map[string]interface{}{
		"device_id":      deviceID,
		"last_active_at": now.Format(time.RFC3339),
		"message":        "Cập nhật thời gian hoạt động của thiết bị thành công",
	}

	client.sendResponse(request.ID, 200, response, "")
	log.Printf("Thời gian hoạt động của thiết bị %s đã cập nhật thành: %s", deviceID, now.Format(time.RFC3339))
}

// Xử lý yêu cầu thiết bị offline.
func (client *WebSocketClient) handleDeviceInactiveRequest(request *WebSocketRequest) {
	// Lấy device_id từ body yêu cầu.
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if deviceID == "" {
		log.Printf("Nhận yêu cầu thiết bị offline nhưng thiếu device_id")
		client.sendResponse(request.ID, 400, nil, "Thiếu tham số device_id")
		return
	}

	log.Printf("Xử lý yêu cầu thiết bị offline, device_id: %s", deviceID)

	// Đặt thời gian hoạt động cuối của thiết bị về nil để biểu thị offline.
	result := client.controller.DB.Model(&models.Device{}).
		Where("device_name = ?", deviceID).
		Update("last_active_at", nil) // Đặt NULL để biểu thị offline

	if result.Error != nil {
		log.Printf("Cập nhật trạng thái offline của thiết bị thất bại: %v", result.Error)
		client.sendResponse(request.ID, 500, nil, fmt.Sprintf("Cập nhật trạng thái offline của thiết bị thất bại: %v", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("Thiết bị không tồn tại: %s", deviceID)
		client.sendResponse(request.ID, 404, nil, "Thiết bị không tồn tại")
		return
	}

	// Dựng phản hồi thành công.
	response := map[string]interface{}{
		"device_id":      deviceID,
		"last_active_at": nil, // Trạng thái offline
		"message":        "Cập nhật trạng thái offline của thiết bị thành công",
	}

	client.sendResponse(request.ID, 200, response, "")
	log.Printf("Thiết bị %s đã được đặt offline", deviceID)
}

// Gửi phản hồi.
func (client *WebSocketClient) sendResponse(requestID string, status int, body map[string]interface{}, errorMsg string) {
	response := WebSocketResponse{
		ID:     requestID,
		Status: status,
		Body:   body,
		Error:  errorMsg,
	}

	if err := client.conn.WriteJSON(response); err != nil {
		log.Printf("Gửi phản hồi thất bại: %v", err)
	} else {
		log.Printf("Đã gửi phản hồi: ID=%s, Status=%d", requestID, status)
	}
}

// Kiểm tra heartbeat bằng ping/pong WebSocket native.
func (client *WebSocketClient) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Đếm số lần ping thất bại liên tiếp.
	pingFailCount := 0
	maxPingFailCount := 3 // Cho phép thất bại liên tiếp 3 lần

	for {
		select {
		case <-client.stopChan:
			log.Printf("Nhận tín hiệu dừng, dừng kiểm tra heartbeat")
			return
		case <-ticker.C:
			if !client.isConnected {
				return
			}

			// Kiểm tra kết nối còn hợp lệ hay không.
			if client.conn == nil {
				log.Printf("Kết nối WebSocket nil, dừng kiểm tra heartbeat")
				return
			}

			// Gửi ping WebSocket native.
			if err := client.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				pingFailCount++
				log.Printf("Gửi ping thất bại (lần %d): %v", pingFailCount, err)

				// Chỉ ngắt kết nối khi số lần thất bại liên tiếp vượt ngưỡng.
				if pingFailCount >= maxPingFailCount {
					log.Printf("Ping thất bại liên tiếp %d lần, ngắt kết nối WebSocket", maxPingFailCount)
					client.conn.Close()
					return
				}
			} else {
				// Ping thành công, reset bộ đếm thất bại.
				if pingFailCount > 0 {
					log.Printf("Ping khôi phục thành công, reset bộ đếm thất bại")
					pingFailCount = 0
				}
			}
		}
	}
}

// Gửi yêu cầu tới client để chủ động push.
func (client *WebSocketClient) SendRequest(method, path string, body map[string]interface{}) error {
	request := WebSocketRequest{
		ID:     uuid.New().String(),
		Method: method,
		Path:   path,
		Body:   body,
	}

	return client.conn.WriteJSON(request)
}

// Gửi yêu cầu và chờ phản hồi.
func (client *WebSocketClient) SendRequestWithResponse(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	// Tạo channel phản hồi.
	responseChan := make(chan *WebSocketResponse, 1)
	client.mu.Lock()
	client.requestChans[requestID] = responseChan
	client.mu.Unlock()

	// Dọn channel phản hồi.
	defer func() {
		client.mu.Lock()
		delete(client.requestChans, requestID)
		client.mu.Unlock()
		close(responseChan)
	}()

	// Gửi yêu cầu.
	if err := client.conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("Gửi yêu cầu thất bại: %v", err)
	}

	// Chờ phản hồi.
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("Yêu cầu timeout")
	case <-ctx.Done():
		return nil, fmt.Errorf("Context đã hủy")
	}
}

// mapToStruct là hàm hỗ trợ chuyển map thành struct.
func mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

// Gửi yêu cầu tới client UUID chỉ định và chờ phản hồi.
func (ctrl *WebSocketController) SendRequestToClient(ctx context.Context, uuid string, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.SendRequestWithResponse(ctx, method, path, body)
	}
	return nil, fmt.Errorf("Client %s chưa kết nối", uuid)
}

// Yêu cầu danh sách công cụ MCP từ client bằng broadcast, chờ phản hồi không rỗng đầu tiên.
func (ctrl *WebSocketController) RequestMcpToolsFromClient(ctx context.Context, agentID string) ([]string, error) {
	toolDetails, err := ctrl.RequestMcpToolDetailsFromClient(ctx, agentID)
	if err != nil {
		return nil, err
	}

	toolNames := make([]string, 0, len(toolDetails))
	for _, detail := range toolDetails {
		toolNames = append(toolNames, detail.Name)
	}

	return toolNames, nil
}

func (ctrl *WebSocketController) RequestMcpToolDetailsFromClient(ctx context.Context, agentID string) ([]MCPTool, error) {
	log.Printf("Bắt đầu yêu cầu danh sách công cụ MCP từ client, agentID: %s", agentID)
	return ctrl.requestMcpToolsByBody(ctx, map[string]interface{}{"agent_id": agentID})
}

func (ctrl *WebSocketController) RequestMcpEndpointStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}

	return ctrl.broadcastMcpStatusRequest(ctx, body)
}

// RequestDeviceMcpToolsFromClient yêu cầu danh sách công cụ MCP theo thiết bị bằng broadcast.
func (ctrl *WebSocketController) RequestDeviceMcpToolsFromClient(ctx context.Context, deviceID string) ([]string, error) {
	toolDetails, err := ctrl.RequestDeviceMcpToolDetailsFromClient(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	toolNames := make([]string, 0, len(toolDetails))
	for _, detail := range toolDetails {
		toolNames = append(toolNames, detail.Name)
	}

	return toolNames, nil
}

func (ctrl *WebSocketController) RequestDeviceMcpToolDetailsFromClient(ctx context.Context, deviceID string) ([]MCPTool, error) {
	log.Printf("Bắt đầu yêu cầu danh sách công cụ MCP theo thiết bị, deviceID: %s", deviceID)
	return ctrl.requestMcpToolsByBody(ctx, map[string]interface{}{"device_id": deviceID})
}

func (ctrl *WebSocketController) requestMcpToolsByBody(ctx context.Context, body map[string]interface{}) ([]MCPTool, error) {
	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "GET", "/api/mcp/tools", body)
	if err != nil {
		return nil, err
	}

	toolsData, ok := response.Body["tools"]
	if !ok {
		return []MCPTool{}, nil
	}

	tools := make([]MCPTool, 0)
	switch v := toolsData.(type) {
	case []interface{}:
		for _, item := range v {
			if toolStr, ok := item.(string); ok {
				tools = append(tools, MCPTool{Name: toolStr, Description: fmt.Sprintf("Công cụ MCP: %s", toolStr), Schema: true})
				continue
			}

			toolMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := toolMap["name"].(string)
			if name == "" {
				continue
			}

			description, _ := toolMap["description"].(string)
			if description == "" {
				description = fmt.Sprintf("Công cụ MCP: %s", name)
			}

			parsed := MCPTool{Name: name, Description: description, Schema: true}
			if inputSchema, ok := toolMap["input_schema"].(map[string]interface{}); ok {
				parsed.InputSchema = inputSchema
			} else if inputSchema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
				// Tương thích một số client trả về tên trường camelCase.
				parsed.InputSchema = inputSchema
			}
			tools = append(tools, parsed)
		}
	case []string:
		for _, name := range v {
			tools = append(tools, MCPTool{Name: name, Description: fmt.Sprintf("Công cụ MCP: %s", name), Schema: true})
		}
	}

	return tools, nil
}

// CallMcpToolFromClient yêu cầu client thực thi công cụ MCP.
func (ctrl *WebSocketController) CallMcpToolFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "POST", "/api/mcp/call", body)
	if err != nil {
		return nil, err
	}

	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

// RequestOpenClawStatusFromClient yêu cầu client trả về trạng thái kết nối OpenClaw.
func (ctrl *WebSocketController) RequestOpenClawStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}

	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "GET", "/api/openclaw/status", body)
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

// CallOpenClawChatFromClient yêu cầu client thực thi kiểm tra hội thoại OpenClaw.
func (ctrl *WebSocketController) CallOpenClawChatFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	if body == nil {
		body = map[string]interface{}{}
	}
	timeoutMs := normalizeOpenClawChatTimeoutMs(body["timeout_ms"])
	body["timeout_ms"] = timeoutMs
	waitTimeout := time.Duration(timeoutMs)*time.Millisecond + 5*time.Second

	response, err := ctrl.broadcastRequestAndWaitFirstSuccessWithTimeout(ctx, "POST", "/api/openclaw/chat", body, waitTimeout)
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

type wsClientResponse struct {
	clientID string
	response *WebSocketResponse
}

// CallOpenClawChatStreamFromClient yêu cầu client thực thi kiểm tra hội thoại OpenClaw dạng stream.
func (ctrl *WebSocketController) CallOpenClawChatStreamFromClient(
	ctx context.Context,
	body map[string]interface{},
	onResponse func(*WebSocketResponse) error,
) (map[string]interface{}, error) {
	if body == nil {
		body = map[string]interface{}{}
	}
	timeoutMs := normalizeOpenClawChatTimeoutMs(body["timeout_ms"])
	body["timeout_ms"] = timeoutMs
	body["stream_events"] = true
	waitTimeout := time.Duration(timeoutMs)*time.Millisecond + 5*time.Second

	responseChan := make(chan wsClientResponse, 64)
	requestID := uuid.New().String()
	callbacksRegistered := 0

	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if !client.isConnected {
			continue
		}

		clientID := client.ID
		responseHandler := func(response *WebSocketResponse) {
			select {
			case responseChan <- wsClientResponse{clientID: clientID, response: response}:
			default:
				log.Printf("Channel phản hồi stream OpenClaw đã đầy, bỏ phản hồi: %s", requestID)
			}
		}

		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
		callbacksRegistered++

		request := WebSocketRequest{
			ID:     requestID,
			Method: "POST",
			Path:   "/api/openclaw/chat",
			Body:   body,
		}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("Gửi yêu cầu stream OpenClaw tới client %s thất bại: %v", client.ID, err)
		}
	}

	if callbacksRegistered == 0 {
		return nil, fmt.Errorf("Không có client đang kết nối")
	}

	defer func() {
		for item := range ctrl.clientsMap.IterBuffered() {
			client := item.Val
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	selectedClientID := ""
	failedClients := map[string]bool{}
	firstError := ""
	timeout := time.After(waitTimeout)

	for {
		select {
		case event := <-responseChan:
			resp := event.response
			if resp == nil {
				continue
			}

			if selectedClientID == "" {
				if resp.Status >= http.StatusBadRequest {
					failedClients[event.clientID] = true
					if firstError == "" {
						msg := strings.TrimSpace(resp.Error)
						if msg != "" {
							firstError = msg
						}
					}
					if len(failedClients) >= callbacksRegistered {
						if firstError != "" {
							return nil, fmt.Errorf("%s", firstError)
						}
						return nil, fmt.Errorf("Tất cả client đều trả về thất bại")
					}
					continue
				}
				selectedClientID = event.clientID
			}

			if event.clientID != selectedClientID {
				continue
			}

			if onResponse != nil {
				if err := onResponse(resp); err != nil {
					return nil, err
				}
			}

			if resp.Status == http.StatusOK {
				if resp.Body == nil {
					return map[string]interface{}{}, nil
				}
				return resp.Body, nil
			}

			if resp.Status >= http.StatusBadRequest {
				msg := strings.TrimSpace(resp.Error)
				if msg == "" {
					msg = fmt.Sprintf("Yêu cầu stream OpenClaw thất bại: status=%d", resp.Status)
				}
				return nil, fmt.Errorf("%s", msg)
			}
		case <-timeout:
			return nil, fmt.Errorf("Yêu cầu timeout")
		case <-ctx.Done():
			return nil, fmt.Errorf("Context đã hủy")
		}
	}
}

func (ctrl *WebSocketController) broadcastRequestAndWaitFirstSuccess(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return ctrl.broadcastRequestAndWaitFirstSuccessWithTimeout(ctx, method, path, body, defaultBroadcastRequestTimeout)
}

func isMcpStatusOnline(body map[string]interface{}) bool {
	if body == nil {
		return false
	}
	if connected, ok := body["connected"].(bool); ok && connected {
		return true
	}
	status, _ := body["status"].(string)
	return strings.EqualFold(strings.TrimSpace(status), "online")
}

func mcpStatusClientCount(body map[string]interface{}) int {
	if body == nil {
		return 0
	}
	switch v := body["client_count"].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (ctrl *WebSocketController) broadcastMcpStatusRequest(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	requestID := uuid.New().String()

	clients := make([]*WebSocketClient, 0)
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if client != nil && client.isConnected {
			clients = append(clients, client)
		}
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("Không có client đang kết nối")
	}

	responseChan := make(chan *WebSocketResponse, len(clients))
	responseHandler := func(response *WebSocketResponse) {
		select {
		case responseChan <- response:
		default:
			log.Printf("Channel phản hồi trạng thái MCP đã đầy, bỏ phản hồi: %s", response.ID)
		}
	}

	for _, client := range clients {
		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
	}
	defer func() {
		for _, client := range clients {
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	sentCount := 0
	for _, client := range clients {
		request := WebSocketRequest{ID: requestID, Method: "GET", Path: "/api/mcp/status", Body: body}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("Gửi yêu cầu trạng thái MCP tới client %s thất bại: %v", client.ID, err)
			continue
		}
		sentCount++
	}
	if sentCount == 0 {
		return nil, fmt.Errorf("Không có client khả dụng")
	}

	offline := map[string]interface{}{
		"connected":    false,
		"status":       "offline",
		"client_count": 0,
	}
	responsesReceived := 0
	successResponses := 0
	firstError := ""
	timeout := time.After(defaultMcpStatusRequestTimeout)
	for {
		select {
		case response := <-responseChan:
			responsesReceived++
			if response != nil && response.Status == http.StatusOK {
				successResponses++
				if isMcpStatusOnline(response.Body) {
					return response.Body, nil
				}
				offline["client_count"] = mcpStatusClientCount(offline) + mcpStatusClientCount(response.Body)
			} else if response != nil && firstError == "" {
				firstError = strings.TrimSpace(response.Error)
			}

			if responsesReceived >= sentCount {
				if successResponses > 0 {
					return offline, nil
				}
				if firstError != "" {
					return nil, fmt.Errorf("%s", firstError)
				}
				return nil, fmt.Errorf("Tất cả client đều trả về thất bại")
			}
		case <-timeout:
			if successResponses > 0 {
				return offline, nil
			}
			if firstError != "" {
				return nil, fmt.Errorf("%s", firstError)
			}
			return nil, fmt.Errorf("Yêu cầu timeout")
		case <-ctx.Done():
			return nil, fmt.Errorf("Context đã hủy")
		}
	}
}

func normalizeOpenClawChatTimeoutMs(v interface{}) int {
	timeout := openClawChatDefaultTimeoutMs
	switch x := v.(type) {
	case int:
		timeout = x
	case int32:
		timeout = int(x)
	case int64:
		timeout = int(x)
	case float32:
		timeout = int(x)
	case float64:
		timeout = int(x)
	}

	if timeout < openClawChatMinTimeoutMs {
		timeout = openClawChatMinTimeoutMs
	}
	if timeout > openClawChatMaxTimeoutMs {
		timeout = openClawChatMaxTimeoutMs
	}
	return timeout
}

func (ctrl *WebSocketController) broadcastRequestAndWaitFirstSuccessWithTimeout(
	ctx context.Context,
	method, path string,
	body map[string]interface{},
	waitTimeout time.Duration,
) (*WebSocketResponse, error) {
	if waitTimeout <= 0 {
		waitTimeout = defaultBroadcastRequestTimeout
	}

	responseChan := make(chan *WebSocketResponse, 10)
	requestID := uuid.New().String()

	responseHandler := func(response *WebSocketResponse) {
		select {
		case responseChan <- response:
		default:
			log.Printf("Channel phản hồi đã đầy, bỏ phản hồi: %s", response.ID)
		}
	}

	callbacksRegistered := 0
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if !client.isConnected {
			continue
		}

		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
		callbacksRegistered++

		request := WebSocketRequest{ID: requestID, Method: method, Path: path, Body: body}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("Gửi yêu cầu tới client %s thất bại: %v", client.ID, err)
		}
	}

	if callbacksRegistered == 0 {
		return nil, fmt.Errorf("Không có client đang kết nối")
	}

	defer func() {
		for item := range ctrl.clientsMap.IterBuffered() {
			client := item.Val
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	responsesReceived := 0
	firstError := ""
	timeout := time.After(waitTimeout)
	for {
		select {
		case response := <-responseChan:
			responsesReceived++
			if response != nil && response.Status == http.StatusOK {
				return response, nil
			}
			if response != nil && firstError == "" {
				msg := strings.TrimSpace(response.Error)
				if msg != "" {
					firstError = msg
				}
			}
			if responsesReceived >= callbacksRegistered {
				if firstError != "" {
					return nil, fmt.Errorf("%s", firstError)
				}
				return nil, fmt.Errorf("Tất cả client đều trả về thất bại")
			}
		case <-timeout:
			return nil, fmt.Errorf("Yêu cầu timeout")
		case <-ctx.Done():
			return nil, fmt.Errorf("Context đã hủy")
		}
	}
}

// Yêu cầu thông tin server từ client.
func (ctrl *WebSocketController) RequestServerInfoFromClient(ctx context.Context, uuid string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/server/info", nil)
}

func (ctrl *WebSocketController) RequestDeviceActivation(ctx context.Context, uuid, deviceID string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/device/activation", map[string]interface{}{
		"device_id": deviceID,
	})
}

// Yêu cầu ping client.
func (ctrl *WebSocketController) RequestPingFromClient(ctx context.Context, uuid string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/server/ping", nil)
}

// InjectMessageToDevice gửi message vào thiết bị bằng broadcast.
func (ctrl *WebSocketController) InjectMessageToDevice(ctx context.Context, deviceID, message string, skipLlm bool, autoListen bool) error {
	body := map[string]interface{}{
		"device_id":   deviceID,
		"message":     message,
		"skip_llm":    skipLlm,
		"auto_listen": autoListen,
	}

	// Tạo yêu cầu.
	request := WebSocketRequest{
		ID:     uuid.New().String(),
		Method: "POST",
		Path:   "/api/device/inject_msg",
		Body:   body,
	}

	// Broadcast tới mọi client đang kết nối.
	var lastError error
	clientCount := 0

	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if client.isConnected {
			clientCount++
			if err := client.conn.WriteJSON(request); err != nil {
				log.Printf("Broadcast message inject tới client %s thất bại: %v", client.ID, err)
				lastError = err
			} else {
				log.Printf("Broadcast message inject tới client %s thành công", client.ID)
			}
		}
	}

	if clientCount == 0 {
		return fmt.Errorf("Không có client đang kết nối")
	}

	return lastError
}

// Gửi yêu cầu bất đồng bộ tới client, không chờ phản hồi.
func (ctrl *WebSocketController) SendRequestToClientAsync(uuid string, method, path string, body map[string]interface{}) error {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.SendRequest(method, path, body)
	}
	return fmt.Errorf("Client %s chưa kết nối", uuid)
}

// Lấy trạng thái kết nối của toàn bộ client.
func (ctrl *WebSocketController) GetClientConnectionStatus() map[string]interface{} {
	clients := make([]map[string]interface{}, 0)
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		clients = append(clients, map[string]interface{}{
			"uuid":      client.ID,
			"connected": client.isConnected,
		})
	}

	return map[string]interface{}{
		"clients": clients,
		"count":   len(clients),
	}
}

// Lấy trạng thái kết nối của client chỉ định.
func (ctrl *WebSocketController) GetClientStatus(uuid string) map[string]interface{} {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return map[string]interface{}{
			"uuid":      client.ID,
			"connected": client.isConnected,
			"message":   "Client đã kết nối",
		}
	}

	return map[string]interface{}{
		"uuid":      uuid,
		"connected": false,
		"message":   "Client chưa kết nối",
	}
}
