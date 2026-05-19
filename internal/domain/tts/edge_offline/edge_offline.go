package edge_offline

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// EdgeOfflineTTSProvider WebSocket TTS provider
type EdgeOfflineTTSProvider struct {
	ServerURL        string
	Timeout          time.Duration
	HandshakeTimeout time.Duration

	// Quản lý kết nối
	conn      *websocket.Conn
	connMutex sync.RWMutex
	// Lock gửi, đảm bảo mỗi thời điểm chỉ một request dùng kết nối
	sendMutex sync.Mutex
}

// NewEdgeOfflineTTSProvider Tạo mới Edge Offline TTS provider
func NewEdgeOfflineTTSProvider(config map[string]interface{}) *EdgeOfflineTTSProvider {
	serverURL, _ := config["server_url"].(string)
	timeout, _ := config["timeout"].(float64)
	handshakeTimeout, _ := config["handshake_timeout"].(float64)

	// Thiết lập giá trị mặc định
	if serverURL == "" {
		serverURL = "ws://localhost:9001/tts"
	}
	if timeout == 0 {
		timeout = 30 // mặc định30noi_dungtimeout
	}
	if handshakeTimeout == 0 {
		handshakeTimeout = 10 // mặc định10noi_dunghandshaketimeout
	}

	return &EdgeOfflineTTSProvider{
		ServerURL:        serverURL,
		Timeout:          time.Duration(timeout) * time.Second,
		HandshakeTimeout: time.Duration(handshakeTimeout) * time.Second,
	}
}

// getConnection Lấy kết nối，nếunoi_dungtồn tạinoi_dungtạo
func (p *EdgeOfflineTTSProvider) getConnection(ctx context.Context) (*websocket.Conn, error) {
	// noi_dungđọcnoi_dungkết nối
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	if conn != nil {
		return conn, nil
	}

	// noi_dungTạo kết nối mới
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	// dualnoi_dungkiểm tra，noi_dung goroutine đãnoi_dungtạonoi_dungkết nối
	if p.conn != nil {
		return p.conn, nil
	}

	// Tạo kết nối mới
	dialer := &websocket.Dialer{
		HandshakeTimeout: p.HandshakeTimeout,
	}
	conn, _, err := dialer.DialContext(ctx, p.ServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Kết nối WebSocket thất bại: %v", err)
	}

	p.conn = conn
	log.Infof("Kết nối WebSocket đã thiết lập")
	return conn, nil
}

// clearConnection noi_dungrỗngkết nối（dùng đểnoi_dung）
func (p *EdgeOfflineTTSProvider) clearConnection() {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		log.Infof("Kết nối WebSocket đã được xóa, chờ reconnect lần sau")
	}
}

// writeMessage safenoi_dung WebSocket kết nốighimessage
func (p *EdgeOfflineTTSProvider) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	// dùngnoi_dunglocknoi_dungkết nốighinoi_dung，noi_dungghinoi_dungdatanoi_dung
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()

	// kiểm trakết nốinoi_dunghợp lệ
	if conn == nil {
		return fmt.Errorf("Kết nối đã đóng")
	}

	return conn.WriteMessage(messageType, data)
}

// TextToSpeech Chuyển text thành giọng nói, trả về dữ liệu frame audio
func (p *EdgeOfflineTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	var frames [][]byte

	// dùnggửilocknoi_dung，noi_dungthời giannoi_dungmộtrequestnoi_dungdùngkết nối
	p.sendMutex.Lock()
	// Lưu ý：noi_dungfunctiontrả vềnoi_dunggiải phónglock，noi_dung goroutine hoàn tấtnoi_dunggiải phóng

	// Lấy kết nối（noi_dunghoặctạo）
	conn, err := p.getConnection(ctx)
	if err != nil {
		p.sendMutex.Unlock() // Lấy kết nốithất bạinoi_dunggiải phónglock
		return nil, err
	}

	// gửitext（dùngnoi_dungghimethod）
	err = p.writeMessage(conn, websocket.TextMessage, []byte(text))
	if err != nil {
		// gửithất bại，noi_dungrỗngkết nối，noi_dungdùngnoi_dung
		log.Errorf("Gửi text thất bại: %v, xóa kết nối", err)
		p.clearConnection()
		p.sendMutex.Unlock() // gửithất bạinoi_dunggiải phónglock
		return nil, fmt.Errorf("Gửi text thất bại: %v", err)
	}

	// tạopipedùng đểaudiodatanoi_dung
	pipeReader, pipeWriter := io.Pipe()
	outputChan := make(chan []byte, 1000)
	startTs := time.Now().UnixMilli()

	// tạoaudiodecoder
	audioDecoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
	if err != nil {
		pipeReader.Close()
		p.sendMutex.Unlock() // tạodecoderthất bạinoi_dunggiải phónglock
		return nil, fmt.Errorf("Tạo audio decoder thất bại: %v", err)
	}

	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if err := audioDecoder.Run(startTs); err != nil {
			log.Errorf("Decode audio thất bại: %v", err)
		}
	}()

	// dùng WaitGroup noi_dungđọc goroutine hoàn tất
	var wg sync.WaitGroup
	wg.Add(1)

	// nhậnWebSocketdatanoi_dungghipipe；locknoi_dung goroutine noi_dung defer giải phóng，noi_dung、lỗihoặc panic noi_dunggiải phóng
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		defer p.sendMutex.Unlock()
		defer close(done)
		defer pipeWriter.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return
				}
				log.Errorf("Đọc message WebSocket thất bại: %v, xóa kết nối", err)
				// kết nốinoi_dung，noi_dungrỗngkết nối，noi_dungdùngnoi_dung
				p.clearConnection()
				return
			}

			if messageType == websocket.BinaryMessage {
				if _, err := pipeWriter.Write(data); err != nil {
					log.Errorf("Ghi audio data thất bại: %v", err)
				}
			}
		}
	}()

	// Thu thập toàn bộOpusframe
	collectorDone := make(chan struct{})
	go func() {
		for frame := range outputChan {
			frames = append(frames, frame)
		}
		close(collectorDone)
	}()

	// noi_dunghoàn tấthoặctimeout
	select {
	case <-ctx.Done():
		_ = pipeWriter.CloseWithError(ctx.Err())
		p.clearConnection()
		<-decoderDone
		<-collectorDone
		return nil, fmt.Errorf("Tổng hợp TTS timeout hoặc bị hủy")
	case <-done:
		<-decoderDone
		<-collectorDone
		return frames, nil
	}
}

// TextToSpeechStream Tổng hợp giọng nói streaming
func (p *EdgeOfflineTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	outputChan := make(chan []byte, 100)

	go func() {
		// dùnggửilocknoi_dung，noi_dungthời giannoi_dungmộtrequestnoi_dungdùngkết nối
		p.sendMutex.Lock()

		// Lấy kết nối（noi_dunghoặctạo）
		conn, err := p.getConnection(ctx)
		if err != nil {
			p.sendMutex.Unlock()
			close(outputChan)
			log.Errorf("lấyKết nối WebSocket thất bại: %v", err)
			return
		}

		// gửitext（dùngnoi_dungghimethod）
		err = p.writeMessage(conn, websocket.TextMessage, []byte(text))
		if err != nil {
			p.sendMutex.Unlock()
			close(outputChan)
			log.Errorf("Gửi text thất bại: %v, xóa kết nối", err)
			// gửithất bại，noi_dungrỗngkết nối，noi_dungdùngnoi_dung
			p.clearConnection()
			return
		}

		// tạopipedùng đểaudiodatanoi_dung
		pipeReader, pipeWriter := io.Pipe()
		startTs := time.Now().UnixMilli()
		audioDecoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
		if err != nil {
			p.sendMutex.Unlock()
			_ = pipeReader.Close()
			_ = pipeWriter.Close()
			close(outputChan)
			log.Errorf("Tạo audio decoder thất bại: %v", err)
			return
		}
		decoderDone := make(chan struct{})
		go func() {
			defer close(decoderDone)
			if err := audioDecoder.Run(startTs); err != nil {
				log.Errorf("Decode audio thất bại: %v", err)
			}
		}()

		defer func() {
			_ = pipeWriter.Close()
			<-decoderDone
			// Sau khi đọc xong thì nhả lock
			log.Debugf("TextToSpeechStream read completed, release sendMutex")
			p.sendMutex.Unlock()
		}()

		// nhậnWebSocketdatanoi_dungghipipe（đọcnoi_dungđangnoi_dunglock，noi_dung）
		for {
			select {
			case <-ctx.Done():
				log.Debugf("TextToSpeechStream context done, exit")
				// Đóng pipeWriter để decoder kết thúc tự nhiên và đóng channel
				return
			default:
				messageType, data, err := conn.ReadMessage()
				if err != nil {
					// Đóng pipeWriter để decoder kết thúc tự nhiên và đóng channel
					pipeWriter.Close()
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						return
					}
					log.Errorf("Đọc message WebSocket thất bại: %v, xóa kết nối", err)
					// kết nốinoi_dung，noi_dungrỗngkết nối，noi_dungdùngnoi_dung
					p.clearConnection()
					return
				}

				if messageType == websocket.BinaryMessage {
					if _, err := pipeWriter.Write(data); err != nil {
						log.Errorf("Ghi audio data thất bại: %v", err)
						return
					}
				}
			}
		}
	}()

	return outputChan, nil
}

// SetVoice Thiết lập tham số voice（EdgeOffline không hỗ trợ thiết lập voice động nhưng không báo lỗi）
func (p *EdgeOfflineTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	// EdgeOffline kết nối qua WebSocket, voice do server kiểm soát, client không hỗ trợ thiết lập động
	// Trả nil để biểu thị thao tác thành công (dù thực tế không làm gì)
	return nil
}

// Close Đóng tài nguyên, giải phóng kết nối
func (p *EdgeOfflineTTSProvider) Close() error {
	p.clearConnection()
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *EdgeOfflineTTSProvider) IsValid() bool {
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	// Kiểm tra kết nối có tồn tại không
	return conn != nil
}
