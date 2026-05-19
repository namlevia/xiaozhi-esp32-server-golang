package speaker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sync"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// StreamingClient client nhận diện streaming WebSocket
type StreamingClient struct {
	wsURL      string
	conn       *websocket.Conn
	sampleRate int
	mutex      sync.Mutex
	writeMu    sync.Mutex
	peekMu     sync.Mutex
	finishWait chan finishResponse
	peekWaits  map[string]chan peekResponse
	lastPeekAt time.Time
}

type finishResponse struct {
	result *IdentifyResult
	err    error
}

type peekResponse struct {
	result    *IdentifyResult
	throttled bool
	err       error
}

// NewStreamingClient Tạo client nhận diện streaming
func NewStreamingClient(baseURL string) *StreamingClient {
	wsURL := deriveWebSocketURL(baseURL)
	return &StreamingClient{
		wsURL: wsURL,
	}
}

// deriveWebSocketURL Suy ra WebSocket URL từ HTTP base_url
func deriveWebSocketURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Errorf("Parse base_url thất bại: %v, dùng giá trị mặc định", err)
		return "ws://localhost:8080/api/v1/speaker/identify_ws"
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	return fmt.Sprintf("%s://%s/api/v1/speaker/identify_ws", scheme, u.Host)
}

// Connect Kết nối tới WebSocket của dịch vụ speaker recognition
func (sc *StreamingClient) Connect(sampleRate int, agentId string, threshold float32) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.sampleRate = sampleRate

	// Nếu đã có kết nối, dùng Ping để kiểm tra kết nối còn hợp lệ không
	if sc.conn != nil {
		if sc.pingConnectionLocked() {
			// Kết nối hợp lệ, tái sử dụng kết nối hiện có
			return nil
		}
		// Kết nối đã ngắt, đóng kết nối cũ để chuẩn bị reconnect
		log.Debugf("Phát hiện kết nối cũ đã ngắt, sẽ thiết lập lại kết nối")
		sc.closeConnectionLocked()
	}

	// Dựng WebSocket URL, gồm tham số sample rate, agent_id và threshold
	wsURL := fmt.Sprintf("%s?sample_rate=%d", sc.wsURL, sampleRate)
	if agentId != "" {
		wsURL += fmt.Sprintf("&agent_id=%s", url.QueryEscape(agentId))
	}
	// Nếu ngưỡng lớn hơn 0 thì truyền tham số threshold
	if threshold > 0 {
		wsURL += fmt.Sprintf("&threshold=%.6f", threshold)
	}

	// Kết nối WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("Kết nối WebSocket thất bại: %v", err)
	}

	sc.conn = conn
	sc.finishWait = nil
	sc.peekWaits = make(map[string]chan peekResponse)

	// Thiết lập read timeout
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Nhận message xác nhận kết nối
	var connectionMsg map[string]interface{}
	if err := conn.ReadJSON(&connectionMsg); err != nil {
		conn.Close()
		sc.conn = nil
		return fmt.Errorf("Đọc message xác nhận kết nối thất bại: %v", err)
	}

	if msgType, ok := connectionMsg["type"].(string); !ok || msgType != "connection" {
		conn.Close()
		sc.conn = nil
		return fmt.Errorf("Message kết nối ngoài dự kiến: %v", connectionMsg)
	}
	conn.SetReadDeadline(time.Time{})

	log.Debugf("Kết nối WebSocket speaker recognition thành công, sample rate: %d Hz, agent_id: %s, ngưỡng: %.4f", sampleRate, agentId, threshold)
	go sc.readLoop(conn)
	return nil
}

// SendAudioChunk Gửi audio data chunk
func (sc *StreamingClient) SendAudioChunk(audioData []float32) error {
	conn := sc.getConn()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	// Chuyển mảng float32 thành byte nhị phân
	chunkBytes := float32ToBytes(audioData)

	// Gửi binary message
	sc.writeMu.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, chunkBytes)
	sc.writeMu.Unlock()
	if err != nil {
		// Đóng kết nối khi gửi thất bại
		sc.failConnection(conn, fmt.Errorf("Gửi audio data thất bại: %v", err))
		return fmt.Errorf("Gửi audio data thất bại: %v", err)
	}

	return nil
}

// FinishAndIdentify Hoàn tất input và lấy kết quả nhận diện
func (sc *StreamingClient) FinishAndIdentify(ctx context.Context) (*IdentifyResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resultCh := make(chan finishResponse, 1)

	sc.mutex.Lock()
	if sc.conn == nil {
		sc.mutex.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	if sc.finishWait != nil {
		sc.mutex.Unlock()
		return nil, fmt.Errorf("finish already in progress")
	}
	sc.finishWait = resultCh
	conn := sc.conn
	sc.mutex.Unlock()

	// Gửi lệnh hoàn tất
	finishCmd := map[string]interface{}{
		"action": "finish",
	}
	sc.writeMu.Lock()
	err := conn.WriteJSON(finishCmd)
	sc.writeMu.Unlock()
	if err != nil {
		sc.clearFinishWait(resultCh)
		sc.failConnection(conn, fmt.Errorf("Gửi lệnh hoàn tất thất bại: %v", err))
		return nil, fmt.Errorf("Gửi lệnh hoàn tất thất bại: %v", err)
	}

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	select {
	case resp := <-resultCh:
		return resp.result, resp.err
	case <-ctx.Done():
		sc.clearFinishWait(resultCh)
		return nil, ctx.Err()
	case <-timer.C:
		sc.clearFinishWait(resultCh)
		return nil, fmt.Errorf("Timeout khi chờ kết quả nhận diện cuối")
	}
}

// PeekAndIdentify Lấy kết quả nhận diện trung gian (không kết thúc lượt hiện tại)
// Trả về: kết quả nhận diện, có bị server debounce không, lỗi
func (sc *StreamingClient) PeekAndIdentify(ctx context.Context, requestID string) (*IdentifyResult, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if requestID == "" {
		requestID = fmt.Sprintf("peek_%d", time.Now().UnixNano())
	}

	sc.peekMu.Lock()
	peekStarted := false
	defer func() {
		if peekStarted {
			sc.lastPeekAt = time.Now()
		}
		sc.peekMu.Unlock()
	}()
	if !sc.lastPeekAt.IsZero() && time.Since(sc.lastPeekAt) < 200*time.Millisecond {
		return nil, true, nil
	}
	peekStarted = true

	respCh := make(chan peekResponse, 1)

	sc.mutex.Lock()
	if sc.conn == nil {
		sc.mutex.Unlock()
		return nil, false, fmt.Errorf("not connected")
	}
	if sc.peekWaits == nil {
		sc.peekWaits = make(map[string]chan peekResponse)
	}
	sc.peekWaits[requestID] = respCh
	conn := sc.conn
	sc.mutex.Unlock()

	peekCmd := map[string]interface{}{
		"action": "peek",
	}
	peekCmd["request_id"] = requestID
	sc.writeMu.Lock()
	err := conn.WriteJSON(peekCmd)
	sc.writeMu.Unlock()
	if err != nil {
		sc.removePeekWait(requestID, respCh)
		sc.failConnection(conn, fmt.Errorf("Gửi lệnh peek thất bại: %v", err))
		return nil, false, fmt.Errorf("Gửi lệnh peek thất bại: %v", err)
	}

	timer := time.NewTimer(1500 * time.Millisecond)
	defer timer.Stop()

	select {
	case resp := <-respCh:
		return resp.result, resp.throttled, resp.err
	case <-ctx.Done():
		sc.removePeekWait(requestID, respCh)
		return nil, false, ctx.Err()
	case <-timer.C:
		sc.removePeekWait(requestID, respCh)
		return nil, false, fmt.Errorf("Timeout khi chờ kết quả peek")
	}
}

// Close Đóng kết nối
func (sc *StreamingClient) Close() error {
	sc.mutex.Lock()
	conn := sc.conn
	sc.conn = nil
	finishWait, peekWaits := sc.takePendingLocked()
	sc.mutex.Unlock()

	if conn != nil {
		if err := conn.Close(); err != nil {
			sc.signalPending(finishWait, peekWaits, fmt.Errorf("Kết nối đã đóng: %v", err))
			return err
		}
	}
	sc.signalPending(finishWait, peekWaits, fmt.Errorf("Kết nối đã đóng"))
	return nil
}

// closeConnectionLocked đóng kết nối (phải gọi khi đã giữ mutex)
func (sc *StreamingClient) closeConnectionLocked() error {
	if sc.conn != nil {
		err := sc.conn.Close()
		sc.conn = nil
		return err
	}
	return nil
}

// IsConnected Kiểm tra đã kết nối chưa
func (sc *StreamingClient) IsConnected() bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.conn != nil
}

// pingConnectionLocked Dùng Ping để kiểm tra kết nối có hợp lệ không (phải gọi khi đã giữ mutex)
func (sc *StreamingClient) pingConnectionLocked() bool {
	if sc.conn == nil {
		return false
	}

	// Dùng Ping message để kiểm tra kết nối còn sống
	sc.writeMu.Lock()
	sc.conn.SetWriteDeadline(time.Now().Add(1000 * time.Millisecond))
	err := sc.conn.WriteMessage(websocket.PingMessage, nil)
	sc.conn.SetWriteDeadline(time.Time{})
	sc.writeMu.Unlock()

	return err == nil
}

func (sc *StreamingClient) getConn() *websocket.Conn {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.conn
}

func (sc *StreamingClient) clearFinishWait(waitCh chan finishResponse) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if sc.finishWait == waitCh {
		sc.finishWait = nil
	}
}

func (sc *StreamingClient) removePeekWait(requestID string, waitCh chan peekResponse) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if existing, ok := sc.peekWaits[requestID]; ok && existing == waitCh {
		delete(sc.peekWaits, requestID)
	}
}

func (sc *StreamingClient) takePendingLocked() (chan finishResponse, []chan peekResponse) {
	finishWait := sc.finishWait
	sc.finishWait = nil

	peekWaits := make([]chan peekResponse, 0, len(sc.peekWaits))
	for requestID, waitCh := range sc.peekWaits {
		peekWaits = append(peekWaits, waitCh)
		delete(sc.peekWaits, requestID)
	}
	return finishWait, peekWaits
}

func (sc *StreamingClient) signalPending(finishWait chan finishResponse, peekWaits []chan peekResponse, err error) {
	if finishWait != nil {
		select {
		case finishWait <- finishResponse{err: err}:
		default:
		}
	}
	for _, waitCh := range peekWaits {
		if waitCh == nil {
			continue
		}
		select {
		case waitCh <- peekResponse{err: err}:
		default:
		}
	}
}

func (sc *StreamingClient) failConnection(conn *websocket.Conn, err error) {
	sc.mutex.Lock()
	if sc.conn != conn {
		sc.mutex.Unlock()
		return
	}
	_ = sc.closeConnectionLocked()
	finishWait, peekWaits := sc.takePendingLocked()
	sc.mutex.Unlock()
	sc.signalPending(finishWait, peekWaits, err)
}

func (sc *StreamingClient) readLoop(conn *websocket.Conn) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			sc.failConnection(conn, fmt.Errorf("Đọc message thất bại: %v", err))
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Warnf("Parse message speaker recognition thất bại: %v", err)
			continue
		}

		if !sc.dispatchMessage(msg) {
			sc.failConnection(conn, parseServerError(msg))
			return
		}
	}
}

func (sc *StreamingClient) dispatchMessage(msg map[string]interface{}) bool {
	msgType, _ := msg["type"].(string)
	switch msgType {
	case "partial_result":
		requestID := getString(msg, "request_id")
		throttled := getBool(msg, "throttled")

		sc.mutex.Lock()
		waitCh := sc.peekWaits[requestID]
		if waitCh != nil {
			delete(sc.peekWaits, requestID)
		}
		sc.mutex.Unlock()
		if waitCh == nil {
			return true
		}

		var result *IdentifyResult
		if resultData, ok := msg["result"].(map[string]interface{}); ok && resultData != nil {
			result = identifyResultFromMap(resultData)
		}
		select {
		case waitCh <- peekResponse{result: result, throttled: throttled}:
		default:
		}
		return true
	case "result":
		sc.mutex.Lock()
		waitCh := sc.finishWait
		sc.finishWait = nil
		sc.mutex.Unlock()
		if waitCh == nil {
			return true
		}

		var result *IdentifyResult
		if resultData, ok := msg["result"].(map[string]interface{}); ok && resultData != nil {
			result = identifyResultFromMap(resultData)
		}
		select {
		case waitCh <- finishResponse{result: result}:
		default:
		}
		return true
	case "error":
		return false
	default:
		// Các message audio_received/connection/ready/cancelled/closing chỉ dùng để báo trạng thái, bỏ qua trực tiếp tại đây
		return true
	}
}

func parseServerError(msg map[string]interface{}) error {
	if errMsg, ok := msg["message"].(string); ok && errMsg != "" {
		return fmt.Errorf("Lỗi server: %s", errMsg)
	}
	return fmt.Errorf("Lỗi server: %v", msg)
}

// float32ToBytes chuyển mảng float32 thành byte nhị phân (little-endian)
func float32ToBytes(samples []float32) []byte {
	buf := make([]byte, len(samples)*4)
	for i, sample := range samples {
		bits := math.Float32bits(sample)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}

// Hàm phụ trợ: lấy value an toàn từ map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getFloat32(m map[string]interface{}, key string) float32 {
	if v, ok := m[key].(float64); ok {
		return float32(v)
	}
	return 0.0
}

func identifyResultFromMap(resultData map[string]interface{}) *IdentifyResult {
	return &IdentifyResult{
		Identified:  getBool(resultData, "identified"),
		SpeakerID:   getString(resultData, "speaker_id"),
		SpeakerName: getString(resultData, "speaker_name"),
		Confidence:  getFloat32(resultData, "confidence"),
		Threshold:   getFloat32(resultData, "threshold"),
	}
}
