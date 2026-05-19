package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	log "xiaozhi-esp32-server-golang/logger"
)

const (
	// DefaultRequestTimeout là timeout request mặc định.
	DefaultRequestTimeout = 30 * time.Second
	// DefaultCloseTimeout là timeout đóng mặc định.
	DefaultCloseTimeout = 5 * time.Second
)

type pendingResponseResult struct {
	response *transport.JSONRPCResponse
	err      error
}

type pendingResponse struct {
	resultCh chan pendingResponseResult
	once     sync.Once
}

func newPendingResponse() *pendingResponse {
	return &pendingResponse{
		resultCh: make(chan pendingResponseResult, 1),
	}
}

func (p *pendingResponse) resolve(response *transport.JSONRPCResponse, err error) {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.resultCh <- pendingResponseResult{
			response: response,
			err:      err,
		}
	})
}

type jsonRPCMessageEnvelope struct {
	Method string           `json:"method"`
	ID     *json.RawMessage `json:"id"`
}

func classifyJSONRPCMessage(message []byte) (method string, hasID bool, err error) {
	var envelope jsonRPCMessageEnvelope
	if err := json.Unmarshal(message, &envelope); err != nil {
		return "", false, err
	}
	return envelope.Method, envelope.ID != nil, nil
}

func requestIDKey(id mcp.RequestId) string {
	raw, err := id.MarshalJSON()
	if err == nil {
		return string(raw)
	}
	return id.String()
}

func isTransportTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "timeout") || strings.Contains(err.Error(), "timeout")
}

/**
// Interface for the transport layer.
type Interface interface {
	// Start the connection. Start should only be called once.
	Start(ctx context.Context) error

	// SendRequest sends a json RPC request and returns the response synchronously.
	SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)

	// SendNotification sends a json RPC Notification to the server.
	SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error

	// SetNotificationHandler sets the handler for notifications.
	// Any notification before the handler is set will be discarded.
	SetNotificationHandler(handler func(notification mcp.JSONRPCNotification))

	// Close the connection.
	Close() error
}
*/

type WebsocketTransport struct {
	url  string
	conn *websocket.Conn

	notifyHandler func(notification mcp.JSONRPCNotification)
	// Thêm callback đóng
	onCloseHandler func(reason string)

	// Quản lý response channel
	respChans    map[string]*pendingResponse
	respChansMux sync.RWMutex

	// Điều khiển listen message
	readDone chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc

	// Trạng thái kết nối
	closed    bool
	closedMux sync.RWMutex

	// Lock ghi WebSocket, tránh ghi concurrent
	writeMux sync.Mutex

	// Config timeout
	requestTimeout time.Duration
	closeTimeout   time.Duration
}

func (t *WebsocketTransport) Send(ctx context.Context, msg []byte) error {
	// Kiểm tra trạng thái kết nối
	t.closedMux.RLock()
	if t.closed {
		t.closedMux.RUnlock()
		return fmt.Errorf("connection is closed")
	}
	t.closedMux.RUnlock()

	// Gửi message, dùng mutex bảo vệ thao tác ghi
	t.writeMux.Lock()
	err := t.conn.WriteMessage(websocket.TextMessage, msg)
	t.writeMux.Unlock()
	return err
}

func NewWebsocketTransport(conn *websocket.Conn) (*WebsocketTransport, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wst := &WebsocketTransport{
		conn:           conn,
		respChans:      make(map[string]*pendingResponse),
		readDone:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		requestTimeout: DefaultRequestTimeout,
		closeTimeout:   DefaultCloseTimeout,
	}
	// Khởi động goroutine listen message
	go wst.readMessages()

	return wst, nil
}

// Triển khai interface
func (t *WebsocketTransport) Start(ctx context.Context) error {
	return nil
}

func (t *WebsocketTransport) popPending(id string) *pendingResponse {
	t.respChansMux.Lock()
	defer t.respChansMux.Unlock()

	pending := t.respChans[id]
	if pending != nil {
		delete(t.respChans, id)
	}
	return pending
}

func (t *WebsocketTransport) failAllPending(err error) {
	t.respChansMux.Lock()
	pending := make([]*pendingResponse, 0, len(t.respChans))
	for id, pendingResp := range t.respChans {
		pending = append(pending, pendingResp)
		delete(t.respChans, id)
	}
	t.respChansMux.Unlock()

	for _, pendingResp := range pending {
		pendingResp.resolve(nil, err)
	}
}

// readMessages liên tục listen message WebSocket.
func (t *WebsocketTransport) readMessages() {
	defer close(t.readDone)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			// Dùng timeout control ở cấp Go
			_, message, err := t.conn.ReadMessage()
			if err != nil {
				t.closedMux.Lock()
				t.closed = true
				t.closedMux.Unlock()
				t.failAllPending(fmt.Errorf("connection is closed"))

				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Errorf("WebSocket read error: %v", err)
				}

				// Khi kết nối đóng, thông báo cho tầng client
				if t.onCloseHandler != nil {
					reason := "connection_closed"
					if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						reason = "normal_closure"
					} else if websocket.IsUnexpectedCloseError(err) {
						reason = "unexpected_closure"
					}
					t.onCloseHandler(reason)
				}

				return
			}

			// Xử lý message nhận được
			t.handleMessage(message)
		}
	}
}

// handleMessage xử lý message nhận được.
func (t *WebsocketTransport) handleMessage(message []byte) {
	method, hasID, err := classifyJSONRPCMessage(message)
	if err != nil {
		log.Warnf("Received unrecognized message: %s", string(message))
		return
	}

	if method != "" {
		if hasID {
			log.Warnf("Received unsupported JSON-RPC request: %s", method)
			return
		}

		var notification mcp.JSONRPCNotification
		if err := json.Unmarshal(message, &notification); err != nil {
			log.Warnf("Received malformed JSON-RPC notification: %s", string(message))
			return
		}
		t.handleNotification(&notification)
		return
	}

	if hasID {
		var response transport.JSONRPCResponse
		if err := json.Unmarshal(message, &response); err != nil {
			log.Warnf("Received malformed JSON-RPC response: %s", string(message))
			return
		}
		t.handleResponse(&response)
		return
	}

	// Format message không nhận diện được
	log.Warnf("Received unrecognized message: %s", string(message))
}

// handleResponse Xử lý response JSON-RPC
func (t *WebsocketTransport) handleResponse(response *transport.JSONRPCResponse) {
	respByte, _ := json.Marshal(response)
	// Chuyển ID thành chuỗi làm key
	idStr := requestIDKey(response.ID)

	pending := t.popPending(idStr)
	if pending == nil {
		log.Warnf("No response channel found for ID: %s, response: %+v", idStr, string(respByte))
		return
	}
	pending.resolve(response, nil)
}

// handleNotification Xử lý notification JSON-RPC
func (t *WebsocketTransport) handleNotification(notification *mcp.JSONRPCNotification) {
	if t.notifyHandler != nil {
		t.notifyHandler(*notification)
	}
}

func (t *WebsocketTransport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	// Kiểm tra trạng thái kết nối
	t.closedMux.RLock()
	if t.closed {
		t.closedMux.RUnlock()
		return nil, fmt.Errorf("connection is closed")
	}
	t.closedMux.RUnlock()

	// Tạo response channel
	idStr := requestIDKey(request.ID)
	pending := newPendingResponse()

	// Đăng ký response channel
	t.respChansMux.Lock()
	t.respChans[idStr] = pending
	t.respChansMux.Unlock()

	// Gửi request (dùng mutex bảo vệ thao tác write)
	t.writeMux.Lock()
	err := t.conn.WriteJSON(request)
	t.writeMux.Unlock()
	if err != nil {
		// Gửi thất bại, dọn channel
		t.popPending(idStr)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Dùng timeout control ở cấp Go để chờ response
	select {
	case result := <-pending.resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return result.response, nil
	case <-ctx.Done():
		// Context bị hủy, dọn channel
		t.popPending(idStr)
		return nil, ctx.Err()
	case <-time.After(t.requestTimeout):
		// Timeout control ở cấp Go
		t.popPending(idStr)
		return nil, fmt.Errorf("request timeout")
	}
}

func (t *WebsocketTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	// Kiểm tra trạng thái kết nối
	t.closedMux.RLock()
	if t.closed {
		t.closedMux.RUnlock()
		return fmt.Errorf("connection is closed")
	}
	t.closedMux.RUnlock()

	// Gửi notification message (dùng mutex bảo vệ thao tác write)
	t.writeMux.Lock()
	err := t.conn.WriteJSON(notification)
	t.writeMux.Unlock()
	return err
}

func (t *WebsocketTransport) SetNotificationHandler(handler func(notification mcp.JSONRPCNotification)) {
	t.notifyHandler = handler
}

// SetOnCloseHandler Thiết lập callback đóng kết nối
func (t *WebsocketTransport) SetOnCloseHandler(handler func(reason string)) {
	t.onCloseHandler = handler
}

func (t *WebsocketTransport) Close() error {
	// Đánh dấu kết nối đã đóng
	t.closedMux.Lock()
	t.closed = true
	t.closedMux.Unlock()
	t.failAllPending(fmt.Errorf("connection is closed"))

	// Thông báo tầng client rằng kết nối sắp đóng
	if t.onCloseHandler != nil {
		t.onCloseHandler("manual_close")
	}

	// Hủy context
	t.cancel()

	// Chờ goroutine đọc kết thúc
	select {
	case <-t.readDone:
	case <-time.After(t.closeTimeout):
		log.Warnf("Timeout waiting for read goroutine to finish")
	}

	// Đóng kết nối WebSocket
	return t.conn.Close()
}

func (t *WebsocketTransport) GetSessionId() string {
	return t.conn.RemoteAddr().String()
}

// IsClosed Kiểm tra kết nối đã đóng chưa
func (t *WebsocketTransport) IsClosed() bool {
	t.closedMux.RLock()
	defer t.closedMux.RUnlock()
	return t.closed
}

// GetActiveRequests Lấy số request đang active hiện tại
func (t *WebsocketTransport) GetActiveRequests() int {
	t.respChansMux.RLock()
	defer t.respChansMux.RUnlock()
	return len(t.respChans)
}

// SetRequestTimeout Thiết lập thời gian timeout request
func (t *WebsocketTransport) SetRequestTimeout(timeout time.Duration) {
	t.requestTimeout = timeout
}

// SetCloseTimeout Thiết lập thời gian timeout đóng
func (t *WebsocketTransport) SetCloseTimeout(timeout time.Duration) {
	t.closeTimeout = timeout
}

// GetRequestTimeout Lấy thời gian timeout request hiện tại
func (t *WebsocketTransport) GetRequestTimeout() time.Duration {
	return t.requestTimeout
}

// GetCloseTimeout Lấy thời gian timeout đóng hiện tại
func (t *WebsocketTransport) GetCloseTimeout() time.Duration {
	return t.closeTimeout
}
