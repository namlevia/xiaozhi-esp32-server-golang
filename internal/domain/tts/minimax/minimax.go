package minimax

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// noi_dung
const (
	wsURL = "wss://api.minimaxi.com/ws/v1/t2a_v2"
)

// noi_dungWebSocket Dialer
var wsDialer = websocket.Dialer{
	ReadBufferSize:   16384, // 16KB đọcnoi_dung
	WriteBufferSize:  16384, // 16KB ghinoi_dung
	HandshakeTimeout: 45 * time.Second,
}

// MinimaxTTSProvider Minimax provider TTS
type MinimaxTTSProvider struct {
	APIKey     string
	Model      string
	Voice      string
	Speed      float64
	Volume     float64
	Pitch      int
	SampleRate int
	Bitrate    int
	Format     string
	Channel    int

	// Quản lý kết nối
	conn      *websocket.Conn
	connMutex sync.RWMutex
	// Lock gửi, đảm bảo mỗi thời điểm chỉ một request dùng kết nối
	sendMutex sync.Mutex
}

// WebSocket messagenoi_dung
type minimaxMessage struct {
	Event           string        `json:"event,omitempty"`
	Model           string        `json:"model,omitempty"`
	VoiceSetting    *voiceSetting `json:"voice_setting,omitempty"`
	AudioSetting    *audioSetting `json:"audio_setting,omitempty"`
	ContinuousSound bool          `json:"continuous_sound,omitempty"`
	Text            string        `json:"text,omitempty"`
}

type minimaxResp struct {
	SessionId string            `json:"session_id,omitempty"`
	Event     string            `json:"event,omitempty"`
	TraceId   string            `json:"trace_id,omitempty"`
	Data      *minimaxData      `json:"data,omitempty"`
	IsFinal   bool              `json:"is_final,omitempty"`
	BaseResp  *minimaxBaseResp  `json:"base_resp,omitempty"`
	ExtraInfo *minimaxExtraInfo `json:"extra_info,omitempty"`
}

type minimaxExtraInfo struct {
	AudioLength     int    `json:"audio_length"`
	AudioSampleRate int    `json:"audio_sample_rate"`
	AudioDuration   int    `json:"audio_duration"`
	AudioSize       int    `json:"audio_size"`
	Bitrate         int    `json:"bitrate"`
	AudioFormat     string `json:"audio_format"`
	AudioChannel    int    `json:"audio_channel"`

	UsageCharacters int `json:"usage_characters"`
	WordCount       int `json:"word_count"`
}

type minimaxBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type voiceSetting struct {
	VoiceID              string  `json:"voice_id"`
	Speed                float64 `json:"speed"`
	Vol                  float64 `json:"vol"`
	Pitch                int     `json:"pitch"`
	EnglishNormalization bool    `json:"english_normalization"`
}

type audioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type minimaxData struct {
	Audio string `json:"audio"`
}

// NewMinimaxTTSProvider Tạo mớiMinimax provider TTS
func NewMinimaxTTSProvider(config map[string]interface{}) *MinimaxTTSProvider {
	apiKey, _ := config["api_key"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	speed, _ := config["speed"].(float64)
	volume, _ := config["vol"].(float64)
	if volume == 0 {
		volume, _ = config["volume"].(float64)
	}
	pitch, _ := config["pitch"].(float64)
	sampleRate, _ := config["sample_rate"].(float64)
	bitrate, _ := config["bitrate"].(float64)
	format, _ := config["format"].(string)
	channel, _ := config["channel"].(float64)

	// Thiết lập giá trị mặc định
	if model == "" {
		model = "speech-2.8-hd"
	}
	if voice == "" {
		voice = "male-qn-qingse"
	}
	if speed == 0 {
		speed = 1.0
	}
	if volume == 0 {
		volume = 1.0
	}
	if sampleRate == 0 {
		sampleRate = 32000
	}
	if bitrate == 0 {
		bitrate = 128000
	}
	if format == "" {
		format = "mp3"
	}
	if channel == 0 {
		channel = 1
	}

	return &MinimaxTTSProvider{
		APIKey:     apiKey,
		Model:      model,
		Voice:      voice,
		Speed:      speed,
		Volume:     volume,
		Pitch:      int(pitch),
		SampleRate: int(sampleRate),
		Bitrate:    int(bitrate),
		Format:     format,
		Channel:    int(channel),
	}
}

// TextToSpeech noi_dungtổng hợp（noi_dunghỗ trợ，dùngstreamingtriển khai）
func (p *MinimaxTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	// Minimax noi_dunghỗ trợstreaming，noi_dungstreamingdatasautrả về
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	var frames [][]byte
	for frame := range outputChan {
		frames = append(frames, frame)
	}

	return frames, nil
}

// TextToSpeechStream Tổng hợp giọng nói streamingtriển khai
func (p *MinimaxTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	// dùnggửilocknoi_dung，noi_dungthời giannoi_dungmộtrequestnoi_dungdùngkết nối
	p.sendMutex.Lock()
	// Lưu ý：noi_dungfunctiontrả vềnoi_dunggiải phónglock，noi_dung goroutine hoàn tấtnoi_dunggiải phóng

	// Lấy kết nối（noi_dunghoặctạo）
	conn, err := p.getConnection(ctx)
	if err != nil {
		p.sendMutex.Unlock()
		return nil, fmt.Errorf("lấyKết nối WebSocket thất bại: %v", err)
	}

	// tạooutputchannel
	outputChan = make(chan []byte, 100)

	// tạopipedùng đểaudionoi_dung
	pipeReader, pipeWriter := io.Pipe()

	// Khởi độngaudiodecoder goroutine
	go func() {
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, p.Format, sampleRate)
		if err != nil {
			log.Errorf("Tạo audio decoder thất bại: %v", err)
			pipeReader.Close()
			close(outputChan)
			return
		}

		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Decode audio thất bại: %v", err)
		}
	}()

	// dùng WaitGroup noi_dungđọc goroutine hoàn tất
	var wg sync.WaitGroup
	wg.Add(1)

	// Khởi độngđọcvàxử lý goroutine；locknoi_dung goroutine noi_dung defer giải phóng，noi_dung、lỗihoặc panic noi_dunggiải phóng
	go func() {
		defer wg.Done()
		defer p.sendMutex.Unlock()
		defer func() {
			pipeWriter.Close()
			pipeReader.Close()
		}()

		p.processStreamTTS(ctx, conn, text, pipeWriter)
	}()

	// noi_dungsaunoi_dung goroutine hoàn tấtnoi_dunggiải phónglock
	go func() {
		wg.Wait()
		log.Debugf("Minimax TTSstreamingtổng hợphoàn tất，noi_dung: %d ms", time.Now().UnixMilli()-startTs)
	}()

	return outputChan, nil
}

// processStreamTTS xử lýstreamingTTStổng hợpnoi_dung
func (p *MinimaxTTSProvider) processStreamTTS(ctx context.Context, conn *websocket.Conn, text string, pipeWriter *io.PipeWriter) {
	// gửitasknoi_dungmessage
	startMsg := minimaxMessage{
		Event: "task_start",
		Model: p.Model,
		VoiceSetting: &voiceSetting{
			VoiceID:              p.Voice,
			Speed:                p.Speed,
			Vol:                  p.Volume,
			Pitch:                p.Pitch,
			EnglishNormalization: false,
		},
		AudioSetting: &audioSetting{
			SampleRate: p.SampleRate,
			Bitrate:    p.Bitrate,
			Format:     p.Format,
			Channel:    p.Channel,
		},
		ContinuousSound: false,
	}

	log.Debugf("minimax gửitasknoi_dungmessage: model=%s, voice=%s, format=%s", p.Model, p.Voice, p.Format)
	if err := p.sendMessage(conn, startMsg); err != nil {
		log.Errorf("gửitasknoi_dungmessagethất bại: %v", err)
		p.clearConnection()
		return
	}

	// noi_dungtasknoi_dung
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := p.readMessage(conn)
	if err != nil {
		// kiểm tranoi_dungtimeoutlỗi
		if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			log.Errorf("đọctasknoi_dungtimeout（10noi_dungchưaNhậnresponse）")
		} else {
			log.Errorf("đọctasknoi_dungthất bại: %v", err)
		}
		p.clearConnection()
		return
	}

	log.Debugf("Nhậntasknoi_dungmessage: %+v", msg)

	if msg.Event != "task_started" {
		log.Errorf("tasknoi_dungthất bại，noi_dung 'task_started'，Nhận: event=%s, đầy đủmessage=%+v", msg.Event, msg)
		if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
			log.Errorf("lỗinoi_dung: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
		}
		p.clearConnection()
		return
	}
	// noi_dungđọctimeout
	conn.SetReadDeadline(time.Time{})

	log.Debugf("tasknoi_dungthành công")

	// gửitextmessage
	continueMsg := minimaxMessage{
		Event: "task_continue",
		Text:  text,
	}

	if err := p.sendMessage(conn, continueMsg); err != nil {
		log.Errorf("gửitextmessagethất bại: %v", err)
		p.clearConnection()
		return
	}

	// đọcaudiodata
	chunkCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Minimax TTSstreamingtổng hợphủy, text: %s", text)
			// gửitasknoi_dungmessage
			finishMsg := minimaxMessage{Event: "task_finish"}
			p.sendMessage(conn, finishMsg)

			// noi_dung，servicenoi_dungNhận task_finish saunoi_dungđóng WebSocket kết nối
			// noi_dungđọc task_finished response（nếuservicenoi_dunggửinoi_dung）
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if finishResp, err := p.readMessage(conn); err == nil {
				log.Debugf("Nhậntasknoi_dung: event=%s, đầy đủmessage=%+v", finishResp.Event, finishResp)
			} else {
				// kết nốinoi_dungđãnoi_dungđóng，noi_dunglà
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debugf("servicenoi_dungđãđóngkết nối（noi_dunglà）")
					if closeErr, ok := err.(*websocket.CloseError); ok {
						log.Debugf("đóngframenoi_dung: code=%d, text=%s", closeErr.Code, closeErr.Text)
					}
				} else {
					log.Debugf("đọctasknoi_dungthất bại: %v", err)
					if closeErr, ok := err.(*websocket.CloseError); ok {
						log.Debugf("đóngframenoi_dung: code=%d, text=%s", closeErr.Code, closeErr.Text)
					}
				}
			}

			// noi_dungrỗngkết nốitrạng thái，noi_dunglàservicenoi_dungđãnoi_dungđóngnoi_dungkết nối
			p.clearConnection()
			return
		default:
		}

		// noi_dungđọctimeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		msg, err := p.readMessage(conn)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("đọcWebSocketmessagethất bại: %v", err)
				// noi_dunglấyđóngframenoi_dung
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Errorf("WebSocketđóngframenoi_dung: code=%d, text=%s", closeErr.Code, closeErr.Text)
				}
				p.clearConnection()
				return
			}
			// noi_dungđónghoặcđọclỗi
			log.Debugf("WebSocketkết nốiđónghoặcđọclỗi: %v", err)
			if closeErr, ok := err.(*websocket.CloseError); ok {
				log.Debugf("WebSocketđóngframenoi_dung: code=%d, text=%s", closeErr.Code, closeErr.Text)
			}
			return
		}

		if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
			log.Errorf("BaseResp: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
		}

		// kiểm tranoi_dunglỗimessage
		if msg.Event == "error" || msg.Event == "task_error" {
			log.Errorf("Nhậnlỗimessage: %+v", msg)
			if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
				log.Errorf("lỗinoi_dung: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
			}
			p.clearConnection()
			return
		}

		// xử lýaudiodata
		if msg.Data != nil && msg.Data.Audio != "" {
			chunkCount++

			// noi_dung hex noi_dungaudiodatachuyểnlànoi_dung
			audioBytes, err := hex.DecodeString(msg.Data.Audio)
			if err != nil {
				log.Errorf("noi_dungaudiodatathất bại: %v", err)
				continue
			}

			// ghipipenoi_dungdecoderxử lý
			if _, err := pipeWriter.Write(audioBytes); err != nil {
				log.Errorf("ghiaudiodatatớipipethất bại: %v", err)
				p.clearConnection()
				return
			}
		}

		// kiểm tranoi_dunghoàn tất
		if msg.IsFinal {
			log.Debugf("Nhậnnoi_dungsaumộtaudionoi_dung，noi_dung%dnoi_dung", chunkCount)
			// gửitasknoi_dungmessage
			finishMsg := minimaxMessage{Event: "task_finish"}
			p.sendMessage(conn, finishMsg)

			// noi_dungrỗngkết nốitrạng thái，noi_dunglàservicenoi_dungđãnoi_dungđóngnoi_dungkết nối
			// noi_dungdùngnoi_dungTạo kết nối mới
			p.clearConnection()
			return
		}
	}
}

// getConnection Lấy kết nối，nếunoi_dungtồn tạinoi_dungtạo
func (p *MinimaxTTSProvider) getConnection(ctx context.Context) (*websocket.Conn, error) {
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

	// tạoHTTPnoi_dung
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	// Tạo kết nối mới
	conn, resp, err := wsDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			log.Errorf("WebSocketkết nốithất bại，trạng tháinoi_dung: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Kết nối WebSocket thất bại: %v", err)
	}

	// noi_dungmessageđọcnoi_dung
	conn.SetReadLimit(1024 * 1024) // 1MB noi_dungmessagenoi_dung

	// noi_dungkết nối
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(1*time.Second))
	})

	// noi_dungkết nốithành côngmessage
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("đọckết nốinoi_dungmessagethất bại: %v", err)
	}

	log.Debugf("Nhậnkết nốinoi_dungmessage（gốc）: %s", string(message))

	var connectMsg minimaxResp
	if err := json.Unmarshal(message, &connectMsg); err != nil {
		conn.Close()
		log.Errorf("parsekết nốinoi_dungmessagethất bại，gốcmessage: %s, lỗi: %v", string(message), err)
		return nil, fmt.Errorf("parsekết nốinoi_dungmessagethất bại: %v", err)
	}

	log.Debugf("Nhậnkết nốinoi_dungmessage（parsesau）: %+v", connectMsg)

	if connectMsg.Event != "connected_success" {
		conn.Close()
		log.Errorf("kết nốithất bại，noi_dung 'connected_success'，Nhận: %+v", connectMsg)
		return nil, fmt.Errorf("kết nốithất bại，Nhận: %+v", connectMsg)
	}

	p.conn = conn
	log.Infof("Minimax Kết nối WebSocket đã thiết lập")
	return conn, nil
}

// clearConnection noi_dungrỗngkết nối（dùng đểnoi_dung）
func (p *MinimaxTTSProvider) clearConnection() {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		log.Infof("Minimax Kết nối WebSocket đã được xóa, chờ reconnect lần sau")
	}
}

// sendMessage gửiJSONmessage
func (p *MinimaxTTSProvider) sendMessage(conn *websocket.Conn, msg minimaxMessage) error {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()

	if conn == nil {
		return fmt.Errorf("Kết nối đã đóng")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sequencenoi_dungmessagethất bại: %v", err)
	}

	log.Debugf("minimax gửimessage: %s", string(data))

	return conn.WriteMessage(websocket.TextMessage, data)
}

// readMessage đọcJSONmessage
func (p *MinimaxTTSProvider) readMessage(conn *websocket.Conn) (*minimaxResp, error) {
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	_ = messageType
	//log.Debugf("minimax đọctớiWebSocketmessage: type=%d, gốcnội dungđộ dài=%d, nội dung=%s", messageType, len(message), string(message))

	var msg minimaxResp
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Errorf("parsemessagethất bại，gốcmessage: %s, lỗi: %v", string(message), err)
		return nil, fmt.Errorf("parsemessagethất bại: %v", err)
	}

	return &msg, nil
}

// SetVoice Thiết lập tham số voice
func (p *MinimaxTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	return nil
}

// Close Đóng tài nguyên, giải phóng kết nối
func (p *MinimaxTTSProvider) Close() error {
	p.clearConnection()
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *MinimaxTTSProvider) IsValid() bool {
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	return conn != nil
}
