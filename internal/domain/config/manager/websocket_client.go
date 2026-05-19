package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"

	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

type MessageHandleFunc func(*WebSocketRequest) (string, error)

type WebSocketClient struct {
	conn           *websocket.Conn
	baseURL        string
	requestTimeout time.Duration
	responseChans  map[string]chan *WebSocketResponse
	callbacks      map[string]func(*WebSocketResponse)
	requestHandler func(*WebSocketRequest) // Xử lý request nhận được.
	mu             sync.RWMutex
	writeMu        sync.Mutex // Bảo vệ thao tác ghi WebSocket, tránh ghi đồng thời.
	isConnected    bool
	connectMu      sync.Mutex
	messageQueue   chan *WebSocketRequest
	workers        sync.WaitGroup

	messageHandle cmap.ConcurrentMap[string, MessageHandleFunc]
	uuid          string

	// Các trường liên quan đến reconnect.
	retryStopChan  chan struct{}  // Tín hiệu dừng goroutine reconnect.
	retryWg        sync.WaitGroup // WaitGroup cho goroutine reconnect.
	retryMu        sync.Mutex     // Bảo vệ thao tác liên quan đến reconnect.
	isRetrying     bool           // Đang reconnect hay không.
	isShuttingDown bool           // Đang chủ động đóng kết nối nên không reconnect.
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

type managerWSClientClaims struct {
	Purpose string `json:"purpose"`
	UUID    string `json:"uuid"`
	jwt.RegisteredClaims
}

var (
	defaultClient           *WebSocketClient
	clientOnce              sync.Once
	systemConfigPushHandler func(map[string]interface{})
)

// SetSystemConfigPushHandler đặt callback khi nhận push system_config; user_config inject trong Init.
func SetSystemConfigPushHandler(fn func(map[string]interface{})) {
	systemConfigPushHandler = fn
}

func GetDefaultClient() *WebSocketClient {
	clientOnce.Do(func() {
		defaultClient = NewWebSocketClient()
	})
	return defaultClient
}

func NewWebSocketClient() *WebSocketClient {
	// Ưu tiên lấy từ biến môi trường, nếu không có thì lấy từ cấu hình.
	baseURL := util.GetBackendURL()
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &WebSocketClient{
		baseURL:        baseURL,
		requestTimeout: 30 * time.Second,
		responseChans:  make(map[string]chan *WebSocketResponse),
		callbacks:      make(map[string]func(*WebSocketResponse)),
		messageQueue:   make(chan *WebSocketRequest, 100),
		messageHandle:  cmap.New[MessageHandleFunc](),
		uuid:           uuid.New().String(),
		retryStopChan:  make(chan struct{}),
		isRetrying:     false,
	}
}

func NewWebSocketClientWithHandler(requestHandler func(*WebSocketRequest)) *WebSocketClient {
	client := NewWebSocketClient()
	client.requestHandler = requestHandler
	return client
}

func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.isConnected {
		return nil
	}

	// Chuyển HTTP URL thành WebSocket URL
	wsURL := "ws://" + c.baseURL[7:] + "/ws" // Bỏ "http://" và thêm "/ws"
	wsToken, err := c.generateWSToken()
	if err != nil {
		return fmt.Errorf("Tạo token xác thực WebSocket thất bại: %v", err)
	}

	// Tạo kết nối WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Origin": []string{c.baseURL},
		"UUID":   []string{c.uuid},
		"Authorization": []string{
			"Bearer " + wsToken,
		},
	})
	if err != nil {
		return fmt.Errorf("Kết nối WebSocket thất bại: %v", err)
	}

	c.conn = conn
	c.isConnected = true

	// Thiết lập ping handler
	conn.SetPongHandler(func(appData string) error {
		log.Debugf("Nhận message pong")
		return nil
	})

	// Khởi động vòng lặp xử lý message
	go c.handleMessages()

	// Khởi động worker gửi message
	c.startWorkers()

	// Khởi động kiểm tra heartbeat
	go c.startHeartbeat()

	log.Debugf("WebSocket client đã kết nối tới: %s", wsURL)
	return nil
}

func (c *WebSocketClient) generateWSToken() (string, error) {
	claims := managerWSClientClaims{
		Purpose: "manager-ws-client",
		UUID:    c.uuid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte(util.GetManagerEndpointAuthToken())
	return token.SignedString(secret)
}

func (c *WebSocketClient) Disconnect() error {
	return c.disconnect(false)
}

// disconnect là method ngắt kết nối nội bộ.
// manualDisconnect=true nghĩa là chủ động ngắt, không trigger reconnect; false nghĩa là ngắt do lỗi và trigger reconnect.
func (c *WebSocketClient) disconnect(manualDisconnect bool) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if !c.isConnected {
		return nil
	}

	if manualDisconnect {
		c.isShuttingDown = true
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Debugf("Có lỗi khi đóng kết nối WebSocket: %v", err)
		}
		c.conn = nil
	}

	c.isConnected = false
	c.mu.Lock()
	// Đóng toàn bộ response channel
	for _, ch := range c.responseChans {
		close(ch)
	}
	c.responseChans = make(map[string]chan *WebSocketResponse)
	c.callbacks = make(map[string]func(*WebSocketResponse))
	c.mu.Unlock()

	// Dừng worker
	close(c.messageQueue)
	c.workers.Wait()
	// Tạo lại message queue
	c.messageQueue = make(chan *WebSocketRequest, 100)

	log.Debugf("Kết nối WebSocket đã ngắt")
	return nil
}

func (c *WebSocketClient) IsConnected() bool {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	return c.isConnected
}

func (c *WebSocketClient) SendRequest(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return nil, fmt.Errorf("Kết nối thất bại: %v", err)
		}
	}

	// Tạo UUID làm request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	// Tạo response channel
	responseChan := make(chan *WebSocketResponse, 1)
	c.mu.Lock()
	c.responseChans[requestID] = responseChan
	c.mu.Unlock()

	// Cleanup response channel
	defer func() {
		c.mu.Lock()
		delete(c.responseChans, requestID)
		c.mu.Unlock()
		close(responseChan)
	}()

	// Gửi request, dùng write lock bảo vệ
	c.writeMu.Lock()
	err := c.conn.WriteJSON(request)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("Gửi request thất bại: %v", err)
	}

	// Chờ response
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(c.requestTimeout):
		return nil, fmt.Errorf("Request timeout")
	case <-ctx.Done():
		return nil, fmt.Errorf("Context đã cancel")
	}
}

// Method tiện ích, dùng ping native của WebSocket.
func (c *WebSocketClient) Ping() error {
	if !c.IsConnected() {
		return fmt.Errorf("WebSocket chưa kết nối")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
}

func (c *WebSocketClient) GetStatus(ctx context.Context) (*WebSocketResponse, error) {
	return c.SendRequest(ctx, "GET", "/api/ws/status", nil)
}

func (c *WebSocketClient) Echo(ctx context.Context, message string) (*WebSocketResponse, error) {
	return c.SendRequest(ctx, "POST", "/api/ws/echo", map[string]interface{}{
		"message": message,
	})
}

// Method tiện ích toàn cục.
func ConnectManagerWebSocket(ctx context.Context) error {
	return GetDefaultClient().Connect(ctx)
}

func DisconnectManagerWebSocket() error {
	client := GetDefaultClient()
	client.StopReconnect()
	return client.disconnect(true) // Chủ động ngắt, không trigger reconnect
}

func SendManagerRequest(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return GetDefaultClient().SendRequest(ctx, method, path, body)
}

func ManagerWebSocketPing(ctx context.Context) error {
	return GetDefaultClient().Ping()
}

func ManagerWebSocketStatus(ctx context.Context) (*WebSocketResponse, error) {
	return GetDefaultClient().GetStatus(ctx)
}

func ManagerWebSocketEcho(ctx context.Context, message string) (*WebSocketResponse, error) {
	return GetDefaultClient().Echo(ctx, message)
}

func IsManagerWebSocketConnected() bool {
	return GetDefaultClient().IsConnected()
}

func SendDeviceRequest(ctx context.Context, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return GetDefaultClient().SendRequest(ctx, "POST", path, body)
}

// startWorkers khởi động worker gửi message.
func (c *WebSocketClient) startWorkers() {
	workerCount := 3 // Khởi động 3 worker

	for i := 0; i < workerCount; i++ {
		c.workers.Add(1)
		go func(workerID int) {
			defer c.workers.Done()

			log.Debugf("Worker Manager WebSocket %d đã khởi động", workerID)

			for request := range c.messageQueue {
				if !c.IsConnected() {
					log.Debugf("Worker %d: WebSocket chưa kết nối, bỏ request", workerID)
					continue
				}

				// Gửi request, dùng write lock bảo vệ
				c.writeMu.Lock()
				err := c.conn.WriteJSON(request)
				c.writeMu.Unlock()
				if err != nil {
					log.Debugf("Worker %d: gửi request thất bại: %v", workerID, err)
					// Kết nối có thể đã ngắt, trigger reconnect
					c.handleConnectionError()
					continue
				}

				log.Debugf("Worker %d: đã gửi request %s", workerID, request.ID)
			}

			log.Debugf("Worker Manager WebSocket %d đã dừng", workerID)
		}(i)
	}
}

// handleConnectionError xử lý lỗi kết nối.
func (c *WebSocketClient) handleConnectionError() {
	if c.IsConnected() {
		log.Warn("Phát hiện lỗi kết nối WebSocket, đang ngắt kết nối...")
		c.disconnect(false) // Ngắt do lỗi, sẽ trigger reconnect
		// Trigger reconnect
		c.triggerReconnect()
	}
}

// startHeartbeat khởi động kiểm tra heartbeat.
func (c *WebSocketClient) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second) // Gửi ping mỗi 30 giây
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}

			// Gửi message ping
			c.writeMu.Lock()
			err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			c.writeMu.Unlock()

			if err != nil {
				log.Warnf("Gửi ping thất bại, kết nối có thể đã ngắt: %v", err)
				c.disconnect(false) // Ngắt do lỗi, sẽ trigger reconnect
				// Trigger reconnect
				c.triggerReconnect()
				return
			}
			log.Debugf("Gửi message ping thành công")

		case <-c.retryStopChan:
			return
		}
	}
}

// triggerReconnect trigger reconnect, non-blocking.
func (c *WebSocketClient) triggerReconnect() {
	c.retryMu.Lock()
	defer c.retryMu.Unlock()

	// Nếu đang shutdown thì không trigger reconnect
	if c.isShuttingDown {
		log.Debug("Đang shutdown, không trigger reconnect")
		return
	}

	// Nếu đang reconnect thì không trigger lại
	if c.isRetrying {
		return
	}

	c.isRetrying = true
	// Khởi động goroutine reconnect
	c.retryWg.Add(1)
	go c.startReconnectLoop()
}

// startReconnectLoop khởi động vòng reconnect bằng thuật toán exponential backoff.
func (c *WebSocketClient) startReconnectLoop() {
	defer func() {
		c.retryMu.Lock()
		c.isRetrying = false
		c.retryMu.Unlock()
		c.retryWg.Done()
	}()

	// Tham số backoff hard-code
	initialDelay := 3 * time.Second // Độ trễ ban đầu 3 giây
	maxDelay := 1 * time.Minute     // Độ trễ tối đa 1 phút
	backoffMultiplier := 2.0        // Hệ số backoff

	delay := initialDelay
	retryCount := 0

	log.Infof("Goroutine retry kết nối Manager WebSocket đã khởi động")

	for {
		// Kiểm tra có nên dừng reconnect hay không
		select {
		case <-c.retryStopChan:
			log.Info("Nhận tín hiệu dừng, dừng reconnect")
			return
		default:
		}

		// Nếu đang shutdown thì dừng reconnect
		c.retryMu.Lock()
		shuttingDown := c.isShuttingDown
		c.retryMu.Unlock()
		if shuttingDown {
			log.Info("Đang shutdown, dừng reconnect")
			return
		}

		// Nếu đã kết nối thì dừng reconnect
		if c.IsConnected() {
			log.Info("Kết nối Manager WebSocket đã khôi phục, dừng reconnect")
			return
		}

		retryCount++
		log.Warnf("Kết nối Manager WebSocket thất bại (lần %d), chờ %v rồi retry...", retryCount, delay)

		// Chờ thời gian delay
		select {
		case <-time.After(delay):
			// Tiếp tục reconnect
		case <-c.retryStopChan:
			log.Info("Nhận tín hiệu dừng, dừng reconnect")
			return
		}

		// Thử kết nối
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.Connect(ctx)
		cancel()

		if err != nil {
			log.Warnf("Kết nối Manager WebSocket thất bại (lần %d): %v", retryCount, err)
			// Tính thời gian delay tiếp theo bằng exponential backoff
			delay = time.Duration(float64(delay) * backoffMultiplier)
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		// Kết nối thành công
		log.Info("Kết nối Manager WebSocket thành công")
		return
	}
}

// StopReconnect dừng goroutine reconnect.
func (c *WebSocketClient) StopReconnect() {
	c.retryMu.Lock()
	c.isShuttingDown = true
	shouldClose := c.retryStopChan != nil
	c.retryMu.Unlock()

	if shouldClose {
		// Dùng select để tránh đóng channel lặp lại
		select {
		case <-c.retryStopChan:
			// Channel đã đóng
		default:
			close(c.retryStopChan)
		}
		c.retryWg.Wait()
		log.Info("Goroutine reconnect Manager WebSocket đã đóng graceful")
	}
}

// SendRequestWithCallback gửi request và dùng callback xử lý response.
func (c *WebSocketClient) SendRequestWithCallback(ctx context.Context, method, path string, body map[string]interface{}, callback func(*WebSocketResponse)) error {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return fmt.Errorf("Kết nối thất bại: %v", err)
		}
	}

	// Tạo UUID làm request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	// Đăng ký callback
	c.mu.Lock()
	c.callbacks[requestID] = callback
	c.mu.Unlock()

	// Cleanup callback
	defer func() {
		c.mu.Lock()
		delete(c.callbacks, requestID)
		c.mu.Unlock()
	}()

	// Đưa request vào queue
	select {
	case c.messageQueue <- &request:
		log.Debugf("Request %s đã được đưa vào queue", requestID)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Message queue đã đầy, request timeout")
	case <-ctx.Done():
		return fmt.Errorf("Context đã cancel")
	}
}

// SendRequestAsync gửi request bất đồng bộ.
func (c *WebSocketClient) SendRequestAsync(ctx context.Context, method, path string, body map[string]interface{}) (string, error) {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return "", fmt.Errorf("Kết nối thất bại: %v", err)
		}
	}

	// Tạo UUID làm request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	// Đưa request vào queue
	select {
	case c.messageQueue <- &request:
		log.Debugf("Request async %s đã được đưa vào queue", requestID)
		return requestID, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("Message queue đã đầy, request timeout")
	case <-ctx.Done():
		return "", fmt.Errorf("Context đã cancel")
	}
}

// GetResponse lấy response theo request ID, dùng cho request async.
func (c *WebSocketClient) GetResponse(requestID string, timeout time.Duration) (*WebSocketResponse, error) {
	responseChan := make(chan *WebSocketResponse, 1)

	// Đăng ký callback tạm
	c.mu.Lock()
	c.callbacks[requestID] = func(response *WebSocketResponse) {
		responseChan <- response
	}
	c.mu.Unlock()

	// Cleanup callback
	defer func() {
		c.mu.Lock()
		delete(c.callbacks, requestID)
		c.mu.Unlock()
		close(responseChan)
	}()

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("Chờ response timeout")
	}
}

// handleSystemConfigPush xử lý thay đổi system config do server push, gọi async các callback đã đăng ký.
func (c *WebSocketClient) handleSystemConfigPush(data map[string]interface{}) {
	if systemConfigPushHandler == nil {
		log.Debugf("Nhận push system_config nhưng chưa đăng ký callback xử lý")
		return
	}
	go systemConfigPushHandler(data)
}

// handleMessages xử lý message WebSocket nhận được.
func (c *WebSocketClient) handleMessages() {
	for {
		if !c.isConnected {
			return
		}

		// Đọc loại message
		messageType, reader, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Debugf("Đọc WebSocket lỗi: %v", err)
			}
			c.disconnect(false) // Ngắt do lỗi, sẽ trigger reconnect
			// Trigger reconnect
			c.triggerReconnect()
			return
		}

		// Xử lý các loại message khác nhau
		switch messageType {
		case websocket.TextMessage:
			// Xử lý message JSON
			var rawMessage map[string]interface{}
			if err := json.NewDecoder(reader).Decode(&rawMessage); err != nil {
				log.Errorf("Parse message JSON thất bại: %v", err)
				continue
			}

			// Phân loại theo message type: server push system_config, request, response.
			if msgType, _ := rawMessage["type"].(string); msgType == "system_config" {
				if data, ok := rawMessage["data"].(map[string]interface{}); ok {
					c.handleSystemConfigPush(data)
				} else {
					log.Warnf("Nhận push system_config nhưng format data không hợp lệ")
				}
			} else if method, exists := rawMessage["method"]; exists && method != nil {
				// Đây là request nhận được
				c.handleIncomingRequest(rawMessage)
			} else if status, exists := rawMessage["status"]; exists && status != nil {
				// Đây là response nhận được
				c.handleIncomingResponse(rawMessage)
			} else {
				log.Warnf("Nhận message WebSocket không nhận diện được: %+v", rawMessage)
			}

		case websocket.PingMessage:
			// Xử lý message ping, tự động trả pong bằng write lock.
			log.Debugf("Nhận message ping, tự động trả pong")
			c.writeMu.Lock()
			err := c.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second))
			c.writeMu.Unlock()
			if err != nil {
				log.Errorf("Gửi pong thất bại: %v", err)
			}

		case websocket.PongMessage:
			// Xử lý message pong
			log.Debugf("Nhận message pong")

		case websocket.CloseMessage:
			// Xử lý message đóng
			log.Debugf("Nhận message đóng")
			c.disconnect(false) // Ngắt do lỗi, sẽ trigger reconnect
			// Trigger reconnect
			c.triggerReconnect()
			return

		default:
			log.Warnf("Nhận message WebSocket loại không xác định: %d", messageType)
		}
	}
}

// handleIncomingRequest xử lý request nhận được.
func (c *WebSocketClient) handleIncomingRequest(rawMessage map[string]interface{}) {
	var request WebSocketRequest
	if err := mapToStruct(rawMessage, &request); err != nil {
		log.Errorf("Parse request WebSocket thất bại: %v", err)
		return
	}

	log.Debugf("Nhận request: ID=%s, Method=%s, Path=%s", request.ID, request.Method, request.Path)

	// Nếu có request handler đã đăng ký thì gọi handler đó
	if c.requestHandler != nil {
		go c.requestHandler(&request)
	} else {
		// Nếu chưa đăng ký handler, dùng handler mặc định xử lý path đã biết
		c.handleDefaultRequest(&request)
	}
}

func (c *WebSocketClient) RegisterMessageHandler(ctx context.Context, path string, handler types.EventHandler) {
	f := func(request *WebSocketRequest) (string, error) {
		return handler(ctx, request.Path, request.Body)
	}
	c.messageHandle.Set(path, f)
}

// handleDefaultRequest là handler mặc định cho request.
func (c *WebSocketClient) handleDefaultRequest(request *WebSocketRequest) {
	switch request.Path {
	case "/api/config/test":
		// Kiểm tra cấu hình có thể mất thời gian nên chạy trong goroutine riêng để không chặn vòng đọc.
		go c.handleConfigTestRequest(request)

	case "/api/mcp/tools":
		// Xử lý request danh sách công cụ MCP.
		c.handleMcpToolListRequest(request)

	case "/api/mcp/status":
		c.handleMcpStatusRequest(request)

	case "/api/mcp/call":
		// Xử lý request gọi công cụ MCP.
		c.handleMcpToolCallRequest(request)

	case "/api/openclaw/status":
		c.handleOpenClawStatusRequest(request)

	case "/api/openclaw/chat":
		c.handleOpenClawChatRequest(request)

	case "/api/server/info":
		// Trả về thông tin server.
		response := map[string]interface{}{
			"server_name": "xiaozhi-server",
			"version":     "1.0.0",
			"uptime":      time.Now().Format(time.RFC3339),
			"request_id":  request.ID,
		}

		if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
			log.Errorf("Gửi phản hồi thông tin server thất bại: %v", err)
		}

	case "/api/server/ping":
		// Phản hồi ping đơn giản.
		response := map[string]interface{}{
			"message": "pong from server",
			"time":    time.Now().Format(time.RFC3339),
		}

		if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
			log.Errorf("Gửi phản hồi ping thất bại: %v", err)
		}
	default:
		handler, exists := c.messageHandle.Get(request.Path)
		if exists {
			// Gọi handler và xử lý giá trị trả về.
			result, err := handler(request)
			if err != nil {
				log.Errorf("Xử lý request %s thất bại: %v", request.Path, err)
				// Gửi phản hồi lỗi.
				if err := c.SendResponse(request.ID, 500, nil, err.Error()); err != nil {
					log.Errorf("Gửi phản hồi lỗi thất bại: %v", err)
				}
			} else {
				// Gửi phản hồi thành công.
				response := map[string]interface{}{
					"result": result,
				}
				if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
					log.Errorf("Gửi phản hồi thành công thất bại: %v", err)
				}
			}
		} else {
			log.Warnf("Nhận đường dẫn request WebSocket không xác định: %s, ID: %s", request.Path, request.ID)

			// Gửi response 404
			if err := c.SendResponse(request.ID, 404, nil, "Unknown endpoint"); err != nil {
				log.Errorf("Gửi phản hồi lỗi thất bại: %v", err)
			}
		}
	}
}

// configTestTotalTimeout là timeout tổng cho kiểm tra cấu hình (VAD+ASR+LLM+TTS).
const configTestTotalTimeout = 90 * time.Second

// handleConfigTestRequest xử lý yêu cầu kiểm tra cấu hình VAD/ASR/LLM/TTS bằng cấu hình được gửi xuống.
func (c *WebSocketClient) handleConfigTestRequest(request *WebSocketRequest) {
	data, _ := request.Body["data"].(map[string]interface{})
	if data == nil {
		log.Debugf("[config_test] Yêu cầu ID=%s thiếu trường data", request.ID)
		_ = c.SendResponse(request.ID, 400, nil, "Thiếu trường data")
		return
	}
	testText, _ := request.Body["test_text"].(string)
	// debug: số cấu hình từng loại trong yêu cầu, không tính provider.
	log.Debugf("[config_test] Yêu cầu ID=%s test_text=%q số mục theo loại: vad=%d asr=%d llm=%d tts=%d",
		request.ID, testText,
		countConfigKeys(data["vad"]), countConfigKeys(data["asr"]),
		countConfigKeys(data["llm"]), countConfigKeys(data["tts"]))

	type configTestResult struct {
		vad, asr, llm, tts map[string]interface{}
	}
	done := make(chan configTestResult, 1)
	go func() {
		vadR, asrR, llmR, ttsR := RunConfigTest(data, testText)
		done <- configTestResult{vadR, asrR, llmR, ttsR}
	}()

	var vadR, asrR, llmR, ttsR map[string]interface{}
	select {
	case res := <-done:
		vadR, asrR, llmR, ttsR = res.vad, res.asr, res.llm, res.tts
	case <-time.After(configTestTotalTimeout):
		log.Warnf("[config_test] Yêu cầu ID=%s hết thời gian chờ tổng %v", request.ID, configTestTotalTimeout)
		body := map[string]interface{}{
			"vad": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Kiểm tra cấu hình hết thời gian chờ tổng"}},
			"asr": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Kiểm tra cấu hình hết thời gian chờ tổng"}},
			"llm": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Kiểm tra cấu hình hết thời gian chờ tổng"}},
			"tts": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Kiểm tra cấu hình hết thời gian chờ tổng"}},
		}
		_ = c.SendResponse(request.ID, 200, body, "")
		return
	}

	// Nếu yêu cầu có một loại nhưng không có cấu hình nào để kiểm tra, trả _none để frontend hiển thị lý do.
	fillEmptyConfigTestResult(data, "vad", vadR)
	fillEmptyConfigTestResult(data, "asr", asrR)
	fillEmptyConfigTestResult(data, "llm", llmR)
	fillEmptyConfigTestResult(data, "tts", ttsR)
	body := map[string]interface{}{
		"vad": vadR,
		"asr": asrR,
		"llm": llmR,
		"tts": ttsR,
	}
	log.Debugf("[config_test] Phản hồi ID=%s số kết quả theo loại: vad=%d asr=%d llm=%d tts=%d",
		request.ID, len(vadR), len(asrR), len(llmR), len(ttsR))
	_ = c.SendResponse(request.ID, 200, body, "")
}

// fillEmptyConfigTestResult ghi mục _none khi yêu cầu có loại cấu hình đó nhưng kết quả kiểm tra rỗng.
func fillEmptyConfigTestResult(data map[string]interface{}, typ string, result map[string]interface{}) {
	if _, has := data[typ]; !has || len(result) > 0 {
		return
	}
	msg := "Chưa cấu hình hoặc chưa bật " + strings.ToUpper(typ)
	result["_none"] = map[string]interface{}{"ok": false, "message": msg}
	log.Debugf("[config_test] Loại %s không có kết quả, đã ghi _none: %s", typ, msg)
}

// countConfigKeys đếm số mục cấu hình trong data, trừ provider, để debug.
func countConfigKeys(v interface{}) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return 0
	}
	n := 0
	for k := range m {
		if k != "provider" {
			n++
		}
	}
	return n
}

// handleIncomingResponse xử lý response nhận được.
func (c *WebSocketClient) handleIncomingResponse(rawMessage map[string]interface{}) {
	var response WebSocketResponse
	if err := mapToStruct(rawMessage, &response); err != nil {
		log.Errorf("Parse response WebSocket thất bại: %v", err)
		return
	}

	log.Debugf("Nhận response: ID=%s, Status=%d", response.ID, response.Status)

	// Tìm response channel và callback tương ứng
	c.mu.RLock()
	responseChan, exists := c.responseChans[response.ID]
	callback, callbackExists := c.callbacks[response.ID]
	c.mu.RUnlock()

	if exists {
		select {
		case responseChan <- &response:
		default:
			log.Debugf("Response channel đã đầy, bỏ response: %s", response.ID)
		}
	}

	if callbackExists {
		go callback(&response)
	}

	if !exists && !callbackExists {
		log.Debugf("Nhận response ID không xác định: %s", response.ID)
	}
}

// SendResponse gửi response cho request đã nhận.
func (c *WebSocketClient) SendResponse(requestID string, status int, body map[string]interface{}, errorMsg string) error {
	if !c.IsConnected() {
		return fmt.Errorf("WebSocket chưa kết nối")
	}

	response := WebSocketResponse{
		ID:     requestID,
		Status: status,
		Body:   body,
		Error:  errorMsg,
	}

	// Dùng write lock bảo vệ
	c.writeMu.Lock()
	err := c.conn.WriteJSON(response)
	c.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("Gửi response thất bại: %v", err)
	}

	log.Debugf("Đã gửi response: ID=%s, Status=%d", requestID, status)
	return nil
}

// SetRequestHandler thiết lập request handler.
func (c *WebSocketClient) SetRequestHandler(handler func(*WebSocketRequest)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestHandler = handler
}

// mapToStruct là helper chuyển map thành struct.
func mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

func toolInfoToSchemaMap(paramsOneOf interface{}) map[string]interface{} {
	if paramsOneOf == nil {
		return nil
	}

	// Field nội bộ của ParamsOneOf không export, json.Marshal trực tiếp có thể trả {}.
	// Ưu tiên dùng ToOpenAPIV3 chính thức để đảm bảo lấy được schema tham số thật.
	if p, ok := paramsOneOf.(*einoschema.ParamsOneOf); ok && p != nil {
		if openAPISchema, err := p.ToOpenAPIV3(); err == nil && openAPISchema != nil {
			raw, err := json.Marshal(openAPISchema)
			if err == nil {
				decoded := map[string]interface{}{}
				if err = json.Unmarshal(raw, &decoded); err == nil {
					if len(decoded) > 0 {
						return decoded
					}
				}
			}
		}
	}

	raw, err := json.Marshal(paramsOneOf)
	if err != nil {
		return nil
	}

	decoded := map[string]interface{}{}
	if err = json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	if openAPIV3, ok := decoded["openAPIV3"].(map[string]interface{}); ok {
		return openAPIV3
	}
	if openAPIV3, ok := decoded["open_api_v3"].(map[string]interface{}); ok {
		return openAPIV3
	}
	if len(decoded) == 0 {
		return nil
	}
	return decoded
}

func convertReportedToolsToToolList(reportedTools map[string]tool.InvokableTool) ([]map[string]interface{}, error) {
	toolList := make([]map[string]interface{}, 0)

	names := make([]string, 0, len(reportedTools))
	for name := range reportedTools {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		invokable := reportedTools[name]
		toolInfo := map[string]interface{}{
			"name":        name,
			"description": fmt.Sprintf("Công cụ MCP: %s", name),
			"schema":      true,
		}

		if info, err := invokable.Info(context.Background()); err == nil && info != nil {
			if info.Desc != "" {
				toolInfo["description"] = info.Desc
			}
			inputSchema := toolInfoToSchemaMap(info.ParamsOneOf)
			if inputSchema != nil {
				toolInfo["input_schema"] = inputSchema
			}
		}

		toolList = append(toolList, toolInfo)
	}

	return toolList, nil
}

func getDeviceMcpTools(deviceID string) ([]map[string]interface{}, error) {
	reportedTools, err := mcp.RefreshReportedToolsByDeviceID(deviceID)
	if err != nil {
		log.Errorf("Refresh danh sách công cụ MCP do thiết bị báo cáo thất bại: %v", err)
		return nil, err
	}

	return convertReportedToolsToToolList(reportedTools)
}

func getAgentMcpTools(agentID string) ([]map[string]interface{}, error) {
	reportedTools, err := mcp.RefreshReportedToolsByAgentID(agentID)
	if err != nil {
		log.Errorf("Refresh danh sách công cụ MCP do agent báo cáo thất bại: %v", err)
		return nil, err
	}

	return convertReportedToolsToToolList(reportedTools)
}

// handleMcpToolListRequest xử lý request danh sách công cụ MCP.
func (c *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
	// Lấy agent_id/device_id từ request body
	agentID := ""
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = id
		}
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if agentID == "" && deviceID == "" {
		log.Warnf("Nhận request danh sách công cụ MCP nhưng thiếu agent_id/device_id")
		if err := c.SendResponse(request.ID, 400, nil, "Thiếu tham số agent_id hoặc device_id"); err != nil {
			log.Errorf("Gửi phản hồi lỗi thất bại: %v", err)
		}
		return
	}

	log.Infof("Xử lý request danh sách công cụ MCP, agent_id: %s, device_id: %s", agentID, deviceID)

	if agentID != "" && deviceID != "" {
		if err := c.SendResponse(request.ID, 400, nil, "Không được truyền đồng thời agent_id và device_id"); err != nil {
			log.Errorf("Gửi phản hồi lỗi thất bại: %v", err)
		}
		return
	}

	var (
		toolList []map[string]interface{}
		err      error
	)
	if deviceID != "" {
		toolList, err = getDeviceMcpTools(deviceID)
	} else {
		toolList, err = getAgentMcpTools(agentID)
	}
	if err != nil {
		log.Errorf("Lấy danh sách công cụ MCP thất bại: %v", err)
		if err := c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Lấy danh sách công cụ thất bại: %v", err)); err != nil {
			log.Errorf("Gửi phản hồi lỗi thất bại: %v", err)
		}
		return
	}

	// Tạo response
	response := map[string]interface{}{
		"agent_id":  agentID,
		"device_id": deviceID,
		"tools":     toolList,
		"count":     len(toolList),
	}

	// Gửi response
	if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
		log.Errorf("Gửi response danh sách công cụ MCP thất bại: %v", err)
	}
}

func (c *WebSocketClient) handleMcpStatusRequest(request *WebSocketRequest) {
	agentID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
	}

	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "Thiếu tham số agent_id")
		return
	}

	connected, clientCount := mcp.GetWsEndpointConnectionStatus(agentID)
	status := "offline"
	if connected {
		status = "online"
	}

	response := map[string]interface{}{
		"agent_id":     agentID,
		"connected":    connected,
		"status":       status,
		"client_count": clientCount,
	}
	_ = c.SendResponse(request.ID, 200, response, "")
}

// Method tiện ích toàn cục, bản async.
func SendManagerRequestAsync(ctx context.Context, method, path string, body map[string]interface{}) (string, error) {
	return GetDefaultClient().SendRequestAsync(ctx, method, path, body)
}

func SendManagerRequestWithCallback(ctx context.Context, method, path string, body map[string]interface{}, callback func(*WebSocketResponse)) error {
	return GetDefaultClient().SendRequestWithCallback(ctx, method, path, body, callback)
}

func GetManagerResponse(requestID string, timeout time.Duration) (*WebSocketResponse, error) {
	return GetDefaultClient().GetResponse(requestID, timeout)
}

// Method hỗ trợ giao tiếp hai chiều.
func SetManagerRequestHandler(handler func(*WebSocketRequest)) {
	GetDefaultClient().SetRequestHandler(handler)
}

func SendManagerResponse(requestID string, status int, body map[string]interface{}, errorMsg string) error {
	return GetDefaultClient().SendResponse(requestID, status, body, errorMsg)
}

// Tạo client kèm request handler.
func NewManagerClientWithHandler(handler func(*WebSocketRequest)) *WebSocketClient {
	return NewWebSocketClientWithHandler(handler)
}

// SendMcpToolListRequest gửi request danh sách công cụ MCP.
func SendMcpToolListRequest(ctx context.Context, agentID string) (*WebSocketResponse, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequest(ctx, "GET", "/api/mcp/tools", body)
}

// SendMcpToolListRequestAsync gửi request danh sách công cụ MCP bất đồng bộ.
func SendMcpToolListRequestAsync(ctx context.Context, agentID string) (string, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequestAsync(ctx, "GET", "/api/mcp/tools", body)
}

// SendMcpToolListRequestWithCallback gửi request danh sách công cụ MCP bằng callback.
func SendMcpToolListRequestWithCallback(ctx context.Context, agentID string, callback func(*WebSocketResponse)) error {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequestWithCallback(ctx, "GET", "/api/mcp/tools", body, callback)
}

// Init khởi tạo provider cấu hình Manager.
// Bao gồm khởi tạo kết nối WebSocket và cơ chế reconnect.
func Init(ctx context.Context) error {
	log.Infof("Initializing Manager config provider with WebSocket client")

	// Tạo WebSocket client
	client := GetDefaultClient()

	// Thử kết nối tới server WebSocket
	if err := client.Connect(ctx); err != nil {
		log.Warnf("Kết nối Manager WebSocket ban đầu thất bại: %v, sẽ khởi động cơ chế reconnect", err)
		// Dù kết nối ban đầu thất bại, vẫn khởi động cơ chế reconnect
		client.triggerReconnect()
	} else {
		log.Infof("Manager config provider initialized successfully")
	}

	return nil
}

// Close đóng provider cấu hình Manager và cleanup tài nguyên.
func Close() error {
	log.Infof("Closing Manager config provider")

	// Dừng goroutine reconnect
	client := GetDefaultClient()
	client.StopReconnect()

	// Chủ động ngắt kết nối, không trigger reconnect
	client.disconnect(true)

	return nil
}

// IsConnected kiểm tra provider cấu hình Manager đã kết nối hay chưa.
func IsConnected() bool {
	return IsManagerWebSocketConnected()
}

// handleMcpToolCallRequest xử lý request gọi công cụ MCP.
func (c *WebSocketClient) handleMcpToolCallRequest(request *WebSocketRequest) {
	agentID := ""
	deviceID := ""
	toolName := ""
	arguments := map[string]interface{}{}
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = id
		}
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
		if t, ok := request.Body["tool_name"].(string); ok {
			toolName = t
		}
		if args, ok := request.Body["arguments"].(map[string]interface{}); ok {
			arguments = args
		}
	}

	if toolName == "" || (agentID == "" && deviceID == "") {
		_ = c.SendResponse(request.ID, 400, nil, "Thiếu tham số tool_name hoặc agent_id/device_id")
		return
	}

	if agentID != "" && deviceID != "" {
		_ = c.SendResponse(request.ID, 400, nil, "Không được truyền đồng thời agent_id và device_id")
		return
	}

	var (
		invokable tool.InvokableTool
		ok        bool
	)
	if deviceID != "" {
		invokable, ok = mcp.GetReportedToolByDeviceIDAndName(deviceID, toolName)
	} else {
		invokable, ok = mcp.GetReportedToolByAgentIDAndName(agentID, toolName)
	}
	if !ok {
		var (
			result    string
			rawCalled bool
			err       error
		)
		if deviceID != "" {
			result, rawCalled, err = mcp.RawCallReportedToolByDeviceID(deviceID, toolName, arguments)
		} else {
			result, rawCalled, err = mcp.RawCallReportedToolByAgentID(agentID, toolName, arguments)
		}
		if rawCalled {
			if err != nil {
				_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Gọi công cụ thất bại (raw call): %v", err))
				return
			}
			log.Warnf("Công cụ %s không có trong danh sách công cụ, đã fallback bằng raw call: device=%s agent=%s", toolName, deviceID, agentID)
			_ = c.SendResponse(request.ID, 200, map[string]interface{}{
				"agent_id":  agentID,
				"device_id": deviceID,
				"tool_name": toolName,
				"result":    result,
			}, "")
			return
		}
		_ = c.SendResponse(request.ID, 404, nil, fmt.Sprintf("Công cụ không tồn tại: %s", toolName))
		return
	}

	argBytes, _ := json.Marshal(arguments)
	result, err := invokable.InvokableRun(context.Background(), string(argBytes))
	if err != nil {
		_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Gọi công cụ thất bại: %v", err))
		return
	}

	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":  agentID,
		"device_id": deviceID,
		"tool_name": toolName,
		"result":    result,
	}, "")
}

func (c *WebSocketClient) handleOpenClawStatusRequest(request *WebSocketRequest) {
	agentID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
	}
	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing agent_id")
		return
	}

	manager := openclaw.GetManager()
	connected := manager.GetAgentSession(agentID) != nil
	status := "offline"
	if connected {
		status = "online"
	}

	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":  agentID,
		"connected": connected,
		"status":    status,
	}, "")
}

const (
	defaultOpenClawChatTimeoutMs = 10 * 60 * 1000
	minOpenClawChatTimeoutMs     = 1000
	maxOpenClawChatTimeoutMs     = 10 * 60 * 1000
	openClawChatTestSessionID    = "openclaw-chat-test-global"
)

func buildOpenClawTestDeviceID(agentID string) string {
	trimmed := strings.TrimSpace(agentID)
	if trimmed == "" {
		trimmed = "unknown"
	}
	return "__openclaw_test__:" + trimmed
}

func buildOpenClawTestSessionID() string {
	return openClawChatTestSessionID
}

func parseOpenClawTimeoutMs(v interface{}) int {
	timeout := defaultOpenClawChatTimeoutMs
	switch x := v.(type) {
	case int:
		timeout = x
	case int32:
		timeout = int(x)
	case int64:
		timeout = int(x)
	case float64:
		timeout = int(x)
	case float32:
		timeout = int(x)
	}
	if timeout < minOpenClawChatTimeoutMs {
		timeout = minOpenClawChatTimeoutMs
	}
	if timeout > maxOpenClawChatTimeoutMs {
		timeout = maxOpenClawChatTimeoutMs
	}
	return timeout
}

func parseOpenClawStreamEvents(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true
		}
	case int:
		return x != 0
	case int32:
		return x != 0
	case int64:
		return x != 0
	case float32:
		return x != 0
	case float64:
		return x != 0
	}
	return false
}

func openClawStreamSnippet(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func (c *WebSocketClient) handleOpenClawChatRequest(request *WebSocketRequest) {
	agentID := ""
	message := ""
	sessionID := ""
	timeoutMs := defaultOpenClawChatTimeoutMs
	streamEvents := false

	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
		if msg, ok := request.Body["message"].(string); ok {
			message = strings.TrimSpace(msg)
		}
		if rawSessionID, ok := request.Body["session_id"].(string); ok && strings.TrimSpace(rawSessionID) != "" {
			sessionID = strings.TrimSpace(rawSessionID)
		}
		timeoutMs = parseOpenClawTimeoutMs(request.Body["timeout_ms"])
		streamEvents = parseOpenClawStreamEvents(request.Body["stream_events"])
	}

	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing agent_id")
		return
	}
	if message == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing message")
		return
	}
	if sessionID == "" {
		sessionID = buildOpenClawTestSessionID()
	}

	manager := openclaw.GetManager()
	if manager.GetAgentSession(agentID) == nil {
		_ = c.SendResponse(request.ID, 409, nil, fmt.Sprintf("openclaw session not connected for agent %s", agentID))
		return
	}

	testDeviceID := buildOpenClawTestDeviceID(agentID)
	// Cleanup cache lịch sử thiết bị test để tránh lẫn kết quả vòng test trước.
	manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
		return nil
	})

	start := time.Now()
	messageID, err := manager.SendMessage(agentID, testDeviceID, message, sessionID)
	if err != nil {
		errMsg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(errMsg, "session not found") {
			_ = c.SendResponse(request.ID, 409, nil, fmt.Sprintf("openclaw session not connected for agent %s", agentID))
			return
		}
		_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("openclaw send failed: %v", err))
		return
	}
	if streamEvents {
		log.Infof(
			"openclaw chat stream started: request_id=%s agent=%s message_id=%s session=%s timeout_ms=%d",
			request.ID,
			agentID,
			messageID,
			sessionID,
			timeoutMs,
		)
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	var replyBuilder strings.Builder
	chunks := make([]string, 0, 8)
	done := false
	firstChunkLatencyMs := -1
	for time.Now().Before(deadline) {
		manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
			correlationID := strings.TrimSpace(msg.CorrelationID)
			if correlationID != "" && correlationID != messageID {
				return nil
			}
			chunk := strings.TrimSpace(msg.Text)
			if chunk != "" {
				replyBuilder.WriteString(chunk)
				chunks = append(chunks, chunk)
				if firstChunkLatencyMs < 0 {
					firstChunkLatencyMs = int(time.Since(start).Milliseconds())
				}
				if streamEvents {
					log.Infof(
						"openclaw chat stream chunk received: request_id=%s agent=%s message_id=%s chunk_index=%d chunk_len=%d chunk_snippet=%q",
						request.ID,
						agentID,
						messageID,
						len(chunks),
						len(chunk),
						openClawStreamSnippet(chunk, 64),
					)
				}
				if streamEvents {
					partialBody := map[string]interface{}{
						"agent_id":    agentID,
						"message_id":  messageID,
						"chunk":       chunk,
						"chunk_index": len(chunks),
						"reply":       strings.TrimSpace(replyBuilder.String()),
						"latency_ms":  int(time.Since(start).Milliseconds()),
						"done":        false,
					}
					if firstChunkLatencyMs >= 0 {
						partialBody["first_chunk_latency_ms"] = firstChunkLatencyMs
					}
					if err := c.SendResponse(request.ID, http.StatusPartialContent, partialBody, ""); err != nil {
						log.Warnf("openclaw chat stream partial response send failed: request_id=%s, err=%v", request.ID, err)
					}
				}
			}
			if msg.IsEnd {
				if streamEvents {
					log.Infof(
						"openclaw chat stream end marker received: request_id=%s agent=%s message_id=%s chunk_count=%d partial_reply_len=%d elapsed_ms=%d",
						request.ID,
						agentID,
						messageID,
						len(chunks),
						len(strings.TrimSpace(replyBuilder.String())),
						int(time.Since(start).Milliseconds()),
					)
				}
				done = true
			}
			return nil
		})
		if done {
			break
		}
		time.Sleep(120 * time.Millisecond)
	}
	reply := strings.TrimSpace(replyBuilder.String())

	if !done {
		// Cleanup cache offline của thiết bị test để tránh tích lũy.
		manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
			return nil
		})
		if reply == "" {
			if streamEvents {
				log.Warnf(
					"openclaw chat stream timeout without reply: request_id=%s agent=%s message_id=%s timeout_ms=%d",
					request.ID,
					agentID,
					messageID,
					timeoutMs,
				)
			}
			_ = c.SendResponse(request.ID, 504, nil, "openclaw response timeout")
			return
		}
		if streamEvents {
			log.Warnf(
				"openclaw chat stream timeout with partial reply: request_id=%s agent=%s message_id=%s chunk_count=%d reply_len=%d elapsed_ms=%d",
				request.ID,
				agentID,
				messageID,
				len(chunks),
				len(reply),
				int(time.Since(start).Milliseconds()),
			)
		}
		_ = c.SendResponse(request.ID, 504, map[string]interface{}{
			"agent_id":               agentID,
			"message_id":             messageID,
			"reply":                  reply,
			"chunks":                 chunks,
			"chunk_count":            len(chunks),
			"latency_ms":             int(time.Since(start).Milliseconds()),
			"first_chunk_latency_ms": firstChunkLatencyMs,
			"timeout_ms":             timeoutMs,
			"finished":               false,
		}, "openclaw response timeout (partial reply received)")
		return
	}

	latencyMs := int(time.Since(start).Milliseconds())
	if streamEvents {
		log.Infof(
			"openclaw chat stream completed: request_id=%s agent=%s message_id=%s chunk_count=%d reply_len=%d latency_ms=%d",
			request.ID,
			agentID,
			messageID,
			len(chunks),
			len(reply),
			latencyMs,
		)
	}
	var firstChunkLatency interface{}
	if firstChunkLatencyMs >= 0 {
		firstChunkLatency = firstChunkLatencyMs
	}
	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":               agentID,
		"message_id":             messageID,
		"reply":                  reply,
		"chunks":                 chunks,
		"chunk_count":            len(chunks),
		"latency_ms":             latencyMs,
		"first_chunk_latency_ms": firstChunkLatency,
		"timeout_ms":             timeoutMs,
		"finished":               true,
	}, "")
}
