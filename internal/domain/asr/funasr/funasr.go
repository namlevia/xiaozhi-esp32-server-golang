package funasr

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"xiaozhi-esp32-server-golang/constants"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
)

// FunasrConfig là cấu trúc config.
type FunasrConfig struct {
	Host          string // Địa chỉ host service FunASR
	Port          string // Port service FunASR
	Mode          string // Mode nhận diện, ví dụ "online"
	SampleRate    int    // Sample rate
	ChunkSize     []int  // Kích thước chunk
	ChunkInterval int    // Khoảng cách chunk
	Timeout       int    // Timeout kết nối (giây)
	AutoEnd       bool   // Có tự kết thúc sau timeout xx ms hay không, không phụ thuộc isSpeaking=false
}

// DefaultConfig là config mặc định.
var DefaultConfig = FunasrConfig{
	Host:          "localhost",
	Port:          "10095",
	Mode:          "online",
	SampleRate:    audio.SampleRate,
	ChunkInterval: 10,
	ChunkSize:     []int{5, 10, 5},
	Timeout:       30,
}

// Funasr triển khai interface ASR.
type Funasr struct {
	config FunasrConfig

	// Quản lý kết nối
	conn      *websocket.Conn
	connMutex sync.RWMutex
	// Khóa gửi, đảm bảo cùng lúc chỉ có một request dùng kết nối
	sendMutex sync.Mutex
}

var funasrStreamSeq atomic.Uint64
var funasrStreamPrefix = uuid.NewString()

type streamDebugState struct {
	audioChunkCount  atomic.Uint64
	audioSampleCount atomic.Uint64
}

// FunasrRequest là cấu trúc request WebSocket FunASR.
type FunasrRequest struct {
	Mode          string `json:"mode,omitempty"`           // Mode nhận diện, ví dụ "online"
	ChunkSize     []int  `json:"chunk_size,omitempty"`     // Kích thước chunk
	ChunkInterval int    `json:"chunk_interval,omitempty"` // Khoảng cách chunk
	AudioFs       int    `json:"audio_fs,omitempty"`       // Sample rate
	WavName       string `json:"wav_name,omitempty"`       // Tên audio
	WavFormat     string `json:"wav_format,omitempty"`     // Định dạng audio
	IsSpeaking    bool   `json:"is_speaking"`              // Có đang nói hay không
	Hotwords      string `json:"hotwords,omitempty"`       // Hotword
	Itn           bool   `json:"itn,omitempty"`            // Có chuẩn hóa text hay không
}

// FunasrResponse là cấu trúc response WebSocket FunASR.
type FunasrResponse struct {
	Text       string  `json:"text"`       // Text nhận diện được
	IsFinal    bool    `json:"is_final"`   // Có phải kết quả cuối hay không
	WavName    string  `json:"wav_name"`   // Tên audio
	TimeStamp  string  `json:"timestamp"`  // Timestamp
	Mode       string  `json:"mode"`       // Mode
	Confidence float64 `json:"confidence"` // Độ tin cậy
}

// NewFunasr tạo instance Funasr mới.
func NewFunasr(config FunasrConfig) (*Funasr, error) {
	if config.Host == "" {
		config = DefaultConfig
	}

	return &Funasr{
		config: config,
	}, nil
}

// getConnection lấy kết nối, nếu chưa có thì tạo mới.
func (f *Funasr) getConnection(ctx context.Context) (*websocket.Conn, error) {
	// Thử đọc kết nối hiện có trước
	f.connMutex.RLock()
	conn := f.conn
	f.connMutex.RUnlock()

	if conn != nil {
		log.Debugf("FunASR WebSocket tái sử dụng kết nối: conn=%p", conn)
		return conn, nil
	}

	// Cần tạo kết nối mới
	f.connMutex.Lock()
	defer f.connMutex.Unlock()

	// Kiểm tra hai lần vì goroutine khác có thể đã tạo kết nối
	if f.conn != nil {
		return f.conn, nil
	}

	// Tạo kết nối mới
	url := fmt.Sprintf("ws://%s:%s/", f.config.Host, f.config.Port)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Kết nối service FunASR thất bại: %v", err)
	}

	f.conn = conn
	log.Infof("Kết nối FunASR WebSocket đã tạo: conn=%p", conn)
	return conn, nil
}

// clearConnection xóa kết nối để reconnect sau khi mất kết nối.
func (f *Funasr) clearConnection() {
	f.connMutex.Lock()
	defer f.connMutex.Unlock()

	if f.conn != nil {
		log.Infof("Kết nối FunASR WebSocket đã được xóa: conn=%p", f.conn)
		f.conn.Close()
		f.conn = nil
	}
}

// StreamingResult là kết quả nhận diện streaming.
type StreamingResult struct {
	Text    string // Text nhận diện được
	IsFinal bool   // Có phải kết quả cuối hay không
}

// isTimeoutError kiểm tra có phải lỗi timeout hay không.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Kiểm tra có phải lỗi timeout mạng hay không
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Kiểm tra error message có chứa từ khóa timeout hay không
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "i/o timeout")
}

// isConnectionClosedError kiểm tra có phải lỗi đóng kết nối hay không.
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}

	// Kiểm tra có phải lỗi đóng WebSocket hay không
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
		return true
	}

	// Kiểm tra error message có chứa từ khóa đóng kết nối hay không
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection closed") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "use of closed network connection")
}

// writeMessage ghi message vào kết nối WebSocket một cách an toàn.
func (f *Funasr) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	// Dùng read lock bảo vệ thao tác ghi kết nối, tránh ghi concurrent làm rối dữ liệu.
	f.connMutex.RLock()
	defer f.connMutex.RUnlock()

	// Kiểm tra kết nối có hợp lệ hay không
	if conn == nil {
		return fmt.Errorf("Kết nối đã đóng")
	}

	return conn.WriteMessage(messageType, data)
}

// StreamingRecognize triển khai nhận diện streaming.
// Nhận dữ liệu audio từ audioStream và trả kết quả qua resultChan.
// Có thể dùng ctx để điều khiển hủy và timeout quá trình nhận diện.
func (f *Funasr) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	// Dùng khóa gửi để đảm bảo cùng lúc chỉ một request dùng kết nối.
	f.sendMutex.Lock()
	// Lưu ý: không giải phóng khóa khi function return, mà giải phóng khi goroutine hoàn tất.

	// Lấy kết nối, tái sử dụng hoặc tạo mới
	conn, err := f.getConnection(ctx)
	if err != nil {
		f.sendMutex.Unlock() // Giải phóng khóa ngay khi lấy kết nối thất bại
		return nil, err
	}

	subCtx, cancelFunc := context.WithCancel(ctx)
	streamID := fmt.Sprintf("funasr-stream-%s-%d", funasrStreamPrefix, funasrStreamSeq.Add(1))
	wavName := streamID
	debugState := &streamDebugState{}

	// Gửi message khởi tạo
	firstMessage := FunasrRequest{
		Mode:          f.config.Mode,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: f.config.ChunkInterval,
		AudioFs:       f.config.SampleRate,
		WavName:       wavName,
		WavFormat:     "pcm",
		IsSpeaking:    true,
		Hotwords:      "{\"Alibaba\":20,\"hello world\":40}",
		Itn:           true,
	}

	log.Debugf(
		"funasr StreamingRecognize bắt đầu: stream_id=%s, conn=%p, mode=%s, chunk_interval=%d, chunk_size=%v, wav_name=%s",
		streamID,
		conn,
		f.config.Mode,
		f.config.ChunkInterval,
		firstMessage.ChunkSize,
		firstMessage.WavName,
	)

	messageBytes, err := json.Marshal(firstMessage)
	if err != nil {
		cancelFunc()
		f.sendMutex.Unlock() // Giải phóng khóa ngay khi serialize thất bại
		return nil, fmt.Errorf("Serialize message khởi tạo thất bại: %v", err)
	}

	err = f.writeMessage(conn, websocket.TextMessage, messageBytes)
	if err != nil {
		// Gửi thất bại, xóa kết nối để lần sau tự reconnect
		log.Errorf("Gửi message khởi tạo thất bại: %v, xóa kết nối", err)
		f.clearConnection()
		cancelFunc()
		f.sendMutex.Unlock() // Giải phóng khóa ngay khi gửi thất bại
		return nil, fmt.Errorf("Gửi message khởi tạo thất bại: %v", err)
	}

	// Tạo channel kết quả có buffer để tránh block
	resultChan := make(chan types.StreamingResult, 20)

	// Dùng WaitGroup chờ hai goroutine hoàn tất
	var wg sync.WaitGroup
	wg.Add(2)

	// Khởi động goroutine nhận và gửi dữ liệu
	// Giải phóng khóa khi goroutine hoàn tất
	go func() {
		defer wg.Done()
		f.recvResult(subCtx, conn, streamID, wavName, debugState, resultChan)
	}()

	go func() {
		defer wg.Done()
		f.forwardStreamAudio(subCtx, cancelFunc, conn, streamID, wavName, debugState, audioStream)
	}()

	// Chờ goroutine hoàn tất ở background rồi giải phóng khóa
	go func() {
		wg.Wait()
		f.clearConnection()
		f.sendMutex.Unlock()
		log.Debugf(
			"funasr StreamingRecognize goroutine hoàn tất, đã giải phóng sendMutex: stream_id=%s, wav_name=%s, chunks=%d, samples=%d",
			streamID,
			wavName,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
		)
	}()

	return resultChan, nil
}

func (f *Funasr) recvResult(ctx context.Context, conn *websocket.Conn, streamID string, wavName string, debugState *streamDebugState, resultChan chan types.StreamingResult) {
	defer func() {
		close(resultChan)
	}()

	for {
		select {
		case <-ctx.Done():
			// Context bị cancel, thoát goroutine
			log.Debugf("funasr recvResult đã bị cancel: %v", ctx.Err())
			return
		default:
			// Tiếp tục xử lý bình thường
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Debugf("funasr recvResult đọc kết quả nhận diện thất bại: stream_id=%s, conn=%p, err=%v, xóa kết nối", streamID, conn, err)
			// Đọc thất bại, xóa kết nối để lần sau tự reconnect
			f.clearConnection()
			return
		}
		log.Debugf(
			"funasr recvResult đọc kết quả nhận diện: stream_id=%s, conn=%p, chunks=%d, samples=%d, payload=%v",
			streamID,
			conn,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
			string(message),
		)

		var response FunasrResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			log.Debugf("funasr recvResult Parse kết quả nhận diện thất bại: %v", err)
			continue
		}

		if response.WavName != "" && response.WavName != wavName {
			log.Warnf(
				"funasr recvResult bỏ qua kết quả không thuộc stream hiện tại: stream_id=%s, expected_wav=%s, actual_wav=%s, conn=%p, chunks=%d, samples=%d",
				streamID,
				wavName,
				response.WavName,
				conn,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
			)
			continue
		}

		// Chỉ gửi kết quả khi có text
		/*if response.Text == "" {
			continue
		}*/

		streamingResult := f.toStreamingResult(response)

		// Gửi kết quả nhận diện
		select {
		case <-ctx.Done():
			// Context bị cancel, thoát goroutine
			log.Debugf("funasr recvResult đã bị cancel: %v", ctx.Err())
			return
		case resultChan <- streamingResult:
		}
		/*if f.config.AutoEnd {
			log.Debugf("funasr recvResult autoend")
			return
		}*/
		// Gửi kết quả thành công
		// Nếu là kết quả cuối và input đã kết thúc thì thoát vòng lặp
		if streamingResult.IsFinal {
			log.Debugf(
				"funasr recvResult isfinal: stream_id=%s, conn=%p, response_mode=%s, raw_is_final=%v, text_len=%d, wav_name=%s, chunks=%d, samples=%d",
				streamID,
				conn,
				response.Mode,
				response.IsFinal,
				len([]rune(response.Text)),
				response.WavName,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
			)
			return
		}
	}
}

func (f *Funasr) toStreamingResult(response FunasrResponse) types.StreamingResult {
	result := types.StreamingResult{
		Text:    response.Text,
		IsFinal: response.IsFinal,
		AsrType: constants.AsrTypeFunAsr,
		Mode:    response.Mode,
	}

	if strings.EqualFold(strings.TrimSpace(f.config.Mode), "2pass") {
		switch strings.ToLower(strings.TrimSpace(response.Mode)) {
		case "2pass-online":
			result.IsFinal = false
		case "2pass-offline":
			result.IsFinal = true
		}
	}

	if result.IsFinal && strings.TrimSpace(result.Text) == "" {
		result.EmptyReason = types.EmptyReasonProviderEmptyFinal
	}

	return result
}

func (f *Funasr) forwardStreamAudio(ctx context.Context, cancelFunc context.CancelFunc, conn *websocket.Conn, streamID string, wavName string, debugState *streamDebugState, audioStream <-chan []float32) {
	sendEndMsg := func() {
		// Gửi message kết thúc
		endMessage := FunasrRequest{
			Mode:          f.config.Mode,
			ChunkInterval: f.config.ChunkInterval,
			ChunkSize:     []int{5, 10, 5},
			WavName:       wavName,
			IsSpeaking:    false,
		}
		endMessageBytes, _ := json.Marshal(endMessage)
		log.Debugf(
			"funasr forwardStreamAudio gửi message kết thúc: stream_id=%s, conn=%p, chunks=%d, samples=%d, payload=%v",
			streamID,
			conn,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
			string(endMessageBytes),
		)
		err := f.writeMessage(conn, websocket.TextMessage, endMessageBytes)
		if err != nil {
			log.Debugf("funasr forwardStreamAudio gửi message kết thúc thất bại: stream_id=%s, conn=%p, err=%v, xóa kết nối", streamID, conn, err)
			f.clearConnection()
		}
	}
	// Xử lý audio stream input
	for {
		select {
		case <-ctx.Done():
			// Context bị cancel, gửi message kết thúc rồi thoát
			log.Debugf(
				"funasr forwardStreamAudio context đã bị cancel: stream_id=%s, conn=%p, chunks=%d, samples=%d, err=%v",
				streamID,
				conn,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
				ctx.Err(),
			)
			// Lưu ý: không cần gọi cancelFunc ở đây vì ctx.Done đã kích hoạt, nghĩa là context đã bị cancel.
			sendEndMsg()
			return
		case pcmChunk, ok := <-audioStream:
			if !ok {
				// Channel đã đóng, input kết thúc, cần thông báo goroutine nhận dừng
				log.Debugf(
					"funasr forwardStreamAudio channel audio đã đóng: stream_id=%s, conn=%p, chunks=%d, samples=%d",
					streamID,
					conn,
					debugState.audioChunkCount.Load(),
					debugState.audioSampleCount.Load(),
				)
				sendEndMsg()
				return
			}

			// Chuyển dữ liệu PCM thành byte
			audioBytes := Float32SliceToBytes(pcmChunk)

			//log.Debugf("funasr forwardStreamAudio gửi dữ liệu audio, pcmChunk len: %v, audioBytes len: %v", len(pcmChunk), len(audioBytes))

			// Gửi dữ liệu audio
			err := f.writeMessage(conn, websocket.BinaryMessage, audioBytes)
			if err != nil {
				log.Debugf("funasr forwardStreamAudio gửi dữ liệu audio thất bại: stream_id=%s, conn=%p, err=%v, xóa kết nối", streamID, conn, err)
				f.clearConnection()
				cancelFunc() // Khi gửi thất bại, cancel context để thông báo goroutine recvResult dừng
				return
			}
			chunkCount := debugState.audioChunkCount.Add(1)
			sampleCount := debugState.audioSampleCount.Add(uint64(len(pcmChunk)))
			if chunkCount <= 3 || chunkCount%10 == 0 {
				log.Debugf(
					"funasr forwardStreamAudio đã gửi audio chunk: stream_id=%s, conn=%p, chunk=%d, chunk_samples=%d, total_samples=%d, bytes=%d",
					streamID,
					conn,
					chunkCount,
					len(pcmChunk),
					sampleCount,
					len(audioBytes),
				)
			}
		}
	}
}

// Process xử lý dữ liệu audio và trả về kết quả nhận diện.
func (f *Funasr) Process(pcmData []float32) (string, error) {
	ctx := context.Background()

	// Dùng khóa gửi để đảm bảo cùng lúc chỉ một request dùng kết nối.
	f.sendMutex.Lock()
	defer f.sendMutex.Unlock()

	// Lấy kết nối, tái sử dụng hoặc tạo mới
	conn, err := f.getConnection(ctx)
	if err != nil {
		return "", err
	}

	audioBytes := Float32SliceToBytes(pcmData)

	// Gửi message khởi tạo
	firstMessage := FunasrRequest{
		Mode:          f.config.Mode,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: f.config.ChunkInterval,
		AudioFs:       f.config.SampleRate,
		WavName:       "stream",
		WavFormat:     "pcm",
		IsSpeaking:    true,
		Hotwords:      "",
		Itn:           true,
	}

	messageBytes, err := json.Marshal(firstMessage)
	if err != nil {
		return "", fmt.Errorf("Serialize message khởi tạo thất bại: %v", err)
	}

	err = f.writeMessage(conn, websocket.TextMessage, messageBytes)
	if err != nil {
		// Gửi thất bại, xóa kết nối để lần sau tự reconnect
		log.Errorf("Gửi message khởi tạo thất bại: %v, xóa kết nối", err)
		f.clearConnection()
		return "", fmt.Errorf("Gửi message khởi tạo thất bại: %v", err)
	}

	// Gửi dữ liệu audio theo từng chunk
	chunkSize := int(audio.SampleRate * 0.1) // Mỗi chunk khoảng 100ms audio (16000 * 0.1)
	for i := 0; i < len(audioBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(audioBytes) {
			end = len(audioBytes)
		}
		chunk := audioBytes[i:end]

		err = f.writeMessage(conn, websocket.BinaryMessage, chunk)
		if err != nil {
			// Gửi thất bại, xóa kết nối để lần sau tự reconnect
			log.Errorf("Gửi dữ liệu audio thất bại: %v, xóa kết nối", err)
			f.clearConnection()
			return "", fmt.Errorf("Gửi dữ liệu audio thất bại: %v", err)
		}
	}

	// Gửi message kết thúc
	endMessage := FunasrRequest{
		IsSpeaking: false,
	}
	endMessageBytes, _ := json.Marshal(endMessage)
	err = f.writeMessage(conn, websocket.TextMessage, endMessageBytes)
	if err != nil {
		// Gửi thất bại, xóa kết nối để lần sau tự reconnect
		log.Errorf("Gửi message kết thúc thất bại: %v, xóa kết nối", err)
		f.clearConnection()
		return "", fmt.Errorf("Gửi message kết thúc thất bại: %v", err)
	}

	// Thiết lập timeout đọc
	conn.SetReadDeadline(time.Now().Add(time.Duration(f.config.Timeout) * time.Second))

	// Đọc kết quả
	var result string
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if isTimeoutError(err) {
				log.Debugf("funasr Process đọc kết quả timeout: %v", err)
				f.clearConnection() // Đọc timeout, xóa kết nối
				return "", fmt.Errorf("Đọc kết quả timeout: %v", err)
			}
			if isConnectionClosedError(err) {
				log.Debugf("funasr Process đọc kết quả gặp kết nối đã đóng: %v", err)
				f.clearConnection() // Kết nối đã đóng, xóa kết nối
				return "", fmt.Errorf("Kết nối đã đóng: %v", err)
			}
			// Đọc thất bại, xóa kết nối để lần sau tự reconnect
			log.Errorf("funasr Process đọc kết quả thất bại: %v, xóa kết nối", err)
			f.clearConnection()
			return "", fmt.Errorf("Đọc kết quả thất bại: %v", err)
		}

		var response FunasrResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			continue
		}

		// Kiểm tra có phải kết quả cuối hay không
		if response.IsFinal {
			result = response.Text
			break
		}
	}

	return result, nil
}

func Float32ToInt16(sample float32) int16 {
	// Giới hạn trong [-1, 1] để tránh overflow
	if sample > 1.0 {
		sample = 1.0
	} else if sample < -1.0 {
		sample = -1.0
	}
	return int16(sample * 32767)
}

func Float32SliceToBytes(samples []float32) []byte {
	data := make([]byte, len(samples)*2)
	for i, s := range samples {
		i16 := Float32ToInt16(s)
		data[2*i] = byte(i16)
		data[2*i+1] = byte(i16 >> 8)
	}
	return data
}

// Close đóng tài nguyên và giải phóng kết nối.
func (f *Funasr) Close() error {
	f.clearConnection()
	return nil
}

// IsValid kiểm tra tài nguyên có hợp lệ hay không.
func (f *Funasr) IsValid() bool {
	f.connMutex.RLock()
	conn := f.conn
	f.connMutex.RUnlock()
	return conn != nil
}

/*
Ví dụ kiểm tra loại lỗi:

1. Kiểm tra lỗi timeout:
   if isTimeoutError(err) {
       // Xử lý timeout, có thể cần retry hoặc điều chỉnh timeout
       log.Warnf("Thao tác timeout: %v", err)
   }

2. Kiểm tra lỗi đóng kết nối:
   if isConnectionClosedError(err) {
       // Xử lý trường hợp kết nối đóng, có thể cần tạo lại kết nối
       log.Warnf("Kết nối đã đóng: %v", err)
   }

3. Xử lý lỗi tổng hợp:
   _, message, err := conn.ReadMessage()
   if err != nil {
       if isTimeoutError(err) {
           // Timeout: có thể do mạng chậm hoặc server phản hồi chậm
           // Gợi ý: điều chỉnh timeout hoặc retry
       } else if isConnectionClosedError(err) {
           // Kết nối đóng: có thể server chủ động ngắt hoặc mạng gián đoạn
           // Gợi ý: tạo lại kết nối
       } else {
           // Lỗi khác: có thể là lỗi protocol hoặc sai định dạng dữ liệu
           // Gợi ý: kiểm tra định dạng dữ liệu hoặc implementation protocol
       }
   }

Các loại lỗi thường gặp:
- Lỗi timeout: i/o timeout, context deadline exceeded
- Kết nối đóng: connection closed, broken pipe, connection reset
- WebSocket đóng: close 1000 (normal), close 1001 (going away)
*/
