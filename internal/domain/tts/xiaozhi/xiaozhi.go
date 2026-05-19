package xiaozhi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

var deviceIdList = []string{
	"ba:8f:17:de:94:94",
	"f2:85:44:27:7b:51",
	"4f:57:fb:d4:69:fa",
	"b3:1e:1c:80:cc:78",
	"32:a5:cc:b7:c0:e4",
	"2b:60:6a:5a:72:10",
	"ca:a6:8b:20:f1:6f",
	"26:1a:d7:27:9f:f8",
	"03:02:26:58:2b:06",
	"5f:f3:85:8b:5d:da",
}

// Ghinoi_dungdeviceIdnoi_dungtắttớinoi_dungthời gian
var (
	deviceIdBlocklist     = make(map[string]time.Time)
	deviceIdBlocklistLock sync.Mutex
	// noi_dungIDtắtthời gian（noi_dungsaunoi_dungdùng）
	deviceIdBlockDuration = 5 * time.Second
)

// XiaozhiProvider noi_dungTTS WebSocket Provider
// hỗ trợstreamingtext-to-speech
type XiaozhiProvider struct {
	ServerAddr  string
	DeviceID    string
	AudioFormat map[string]interface{}
	Header      http.Header
}

// noi_dungdeviceIdtắtlist
func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// noi_dungdeviceIdtắtlist
			deviceIdBlocklistLock.Lock()
			now := time.Now()
			for id, expireTime := range deviceIdBlocklist {
				if now.After(expireTime) {
					delete(deviceIdBlocklist, id)
					log.Debugf("noi_dungIDtắtđãnoi_dung，noi_dungmớibật: %s", id)
				}
			}
			deviceIdBlocklistLock.Unlock()
		}
	}()
}

// noi_dungdeviceIdnoi_dungtớitắtlist
func blockDeviceId(deviceId string) {
	deviceIdBlocklistLock.Lock()
	defer deviceIdBlocklistLock.Unlock()

	deviceIdBlocklist[deviceId] = time.Now().Add(deviceIdBlockDuration)
	log.Warnf("noi_dungID %s đãnoi_dungtớitắtlist，noi_dung %v saunoi_dungmớibật", deviceId, deviceIdBlockDuration)
}

// kiểm tradeviceIdnoi_dungtắtlistđang
func isDeviceIdBlocked(deviceId string) bool {
	deviceIdBlocklistLock.Lock()
	defer deviceIdBlocklistLock.Unlock()

	expireTime, exists := deviceIdBlocklist[deviceId]
	if !exists {
		return false
	}

	// nếunoi_dungthời gianđãnoi_dung，noi_dungtắtlistđangnoi_dung
	if time.Now().After(expireTime) {
		delete(deviceIdBlocklist, deviceId)
		log.Debugf("noi_dungIDtắtđãnoi_dung，noi_dungmớibật: %s", deviceId)
		return false
	}

	return true
}

// NewXiaozhiProvider Tạo mớinoi_dungTTS Provider
func NewXiaozhiProvider(config map[string]interface{}) *XiaozhiProvider {
	serverAddr, _ := config["server_addr"].(string)
	deviceID, _ := config["device_id"].(string)
	clientID, _ := config["client_id"].(string)
	token, _ := config["token"].(string)
	format := map[string]interface{}{
		"sample_rate":    16000,
		"channels":       1,
		"frame_duration": 20,
		"format":         "opus",
	}

	header := http.Header{}
	header.Set("Device-Id", deviceID)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+token)
	header.Set("Protocol-Version", "1")
	header.Set("Client-Id", clientID)

	return &XiaozhiProvider{
		ServerAddr:  serverAddr,
		DeviceID:    deviceID,
		AudioFormat: format,
		Header:      header,
	}
}

// selectDeviceId noi_dungmộtkhả dụngnoi_dungID
func (p *XiaozhiProvider) selectDeviceId() string {
	// noi_dungdeviceIdListđangnoi_dungchưanoi_dungtắtnoi_dungdeviceId
	for _, deviceId := range deviceIdList {
		if !isDeviceIdBlocked(deviceId) {
			log.Debugf("noi_dungchưanoi_dungtắtnoi_dungID: %s", deviceId)
			return deviceId
		}
	}

	// nếunoi_dungdeviceIdnoi_dungtắt，noi_dungdeviceIdđangnoi_dung
	if len(deviceIdList) > 0 {
		// dùngnoi_dung（noi_dungthời gian）
		selectedIndex := int(time.Now().Unix()) % len(deviceIdList)
		selectedDeviceId := deviceIdList[selectedIndex]
		log.Warnf("noi_dungdeviceIdnoi_dungtắt，noi_dungID: %s (noi_dung: %d)", selectedDeviceId, selectedIndex)
		return selectedDeviceId
	}

	// nếudeviceIdListlàrỗng，dùngnoi_dungdeviceId
	if p.DeviceID != "" {
		log.Warnf("deviceIdListlàrỗng，dùnghiện tạinoi_dungID: %s", p.DeviceID)
		return p.DeviceID
	}

	// nếunoi_dung，trả vềnoi_dungmộtnoi_dungID（nếutồn tại）
	if len(deviceIdList) > 0 {
		return deviceIdList[0]
	}

	return ""
}

// createWSConnection Tạo mớiWebSocketkết nối
func (p *XiaozhiProvider) createWSConnection(ctx context.Context) (*websocket.Conn, string, error) {
	// noi_dungmộtkhả dụngnoi_dungID
	selectedDeviceId := p.selectDeviceId()
	if selectedDeviceId == "" {
		return nil, "", fmt.Errorf("noi_dungID")
	}

	// noi_dungmớihiện tạip.DeviceIDvàHeader
	p.DeviceID = selectedDeviceId
	p.Header.Set("Device-Id", selectedDeviceId)

	// Tạo kết nối mới
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, p.ServerAddr, p.Header)
	if err != nil {
		log.Errorf("tạoKết nối WebSocket thất bại: %v, noi_dungID: %s", err, selectedDeviceId)
		blockDeviceId(selectedDeviceId) // noi_dungthất bạinoi_dungdeviceIdnoi_dungtắtlist
		return nil, "", err
	}

	// noi_dungkết nối
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})

	// mớinoi_dungkết nốinoi_dunggửihellomessage
	helloMsg := map[string]interface{}{
		"type":         "hello",
		"device_id":    selectedDeviceId,
		"transport":    "websocket",
		"version":      1,
		"audio_params": p.AudioFormat,
	}
	log.Debugf("Tạo kết nối mớinoi_dunggửihellomessage，noi_dungID: %s", selectedDeviceId)
	if err := conn.WriteJSON(helloMsg); err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("gửihellomessagethất bại: %v", err)
	}

	return conn, selectedDeviceId, nil
}

type RecvMsg struct {
	Type    string `json:"type"`
	State   string `json:"state"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

// sendStopMessage gửistopmessagenoi_dungđóngkết nối
func sendStopMessage(conn *websocket.Conn, deviceId string) {
	stopMsg := map[string]interface{}{
		"type":      "listen",
		"device_id": deviceId,
		"state":     "stop",
	}
	if err := conn.WriteJSON(stopMsg); err != nil {
		log.Warnf("gửistopmessagethất bại: %v, noi_dungID: %s", err, deviceId)
	} else {
		log.Debugf("gửistopmessagethành công，noi_dungID: %s", deviceId)
	}
}

// handleTTSConnection noi_dungLấy kết nối、gửimessagevànhậnmessagenoi_dung
func (p *XiaozhiProvider) handleTTSConnection(ctx context.Context, text string, outputChan chan []byte) error {
	// Tạo kết nối mới
	conn, deviceId, err := p.createWSConnection(ctx)
	if err != nil {
		return fmt.Errorf("tạonoi_dungTTSkết nốithất bại: %v", err)
	}
	defer func() {
		// gửistopmessagenoi_dungđóngkết nối
		sendStopMessage(conn, deviceId)
		conn.Close()
	}()

	// gửilisten detectmessage
	sendText := fmt.Sprintf("`%s`", text)
	listenMsg := map[string]interface{}{
		"type":      "listen",
		"device_id": deviceId,
		"state":     "detect",
		"text":      sendText,
	}
	log.Debugf("gửixiaozhiservicenoi_dungmessage: %v", listenMsg)

	if err := conn.WriteJSON(listenMsg); err != nil {
		log.Errorf("gửilistenmessagethất bại: %v，noi_dungID: %s", err, deviceId)
		blockDeviceId(deviceId) // noi_dungdeviceIdnoi_dungtắtlist
		return fmt.Errorf("gửimessagethất bại: %v", err)
	}

	// đọcnoi_dungxử lýmessage
	startTs := time.Now().UnixMilli()
	var firstFrameTs bool
	i := 0
	receivedFrames := false

	for {
		select {
		case <-ctx.Done():
			log.Debugf("xiaozhiservicenoi_dungmessagectx.Done(), noi_dungID: %s", deviceId)
			return nil
		default:
		}
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			// kết nốinoi_dung
			log.Errorf("đọcmessagelỗi: %v，noi_dungID: %s", err, deviceId)

			// nếunoi_dungNhậnbất kỳaudioframe，noi_dungkết nốinoi_dung，noi_dungdeviceIdnoi_dungtắtlist
			if !receivedFrames {
				blockDeviceId(deviceId)
			}

			return fmt.Errorf("đọcmessagelỗi: %v", err)
		}
		if msgType == websocket.TextMessage {
			log.Debugf("Nhậnxiaozhiservicenoi_dungmessage: %s", string(msg))
			var recvMsg RecvMsg
			err := json.Unmarshal(msg, &recvMsg)
			if err != nil {
				continue
			}
			if recvMsg.Type == "tts" {
				if recvMsg.State == "stop" {
					log.Debugf("xiaozhiservicenoi_dungmessagetts stopmessage")
					return nil
				}
			}
		} else if msgType == websocket.BinaryMessage {
			receivedFrames = true
			if !firstFrameTs {
				firstFrameTs = true
				log.Debugf("ttsnoi_dung: xiaozhiservicetts noi_dungmộtaudioframethời gian: %d", time.Now().UnixMilli()-startTs)
			}
			outputChan <- msg
			if i%20 == 0 {
				log.Debugf("xiaozhiservicenoi_dungaudiomessage, đãNhận%dnoi_dungaudioframe", i)
			}
			i++
		}
	}
}

// TextToSpeechStream triển khaistreamingTTS，trả vềopusaudioframechan
func (p *XiaozhiProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	outputChan := make(chan []byte, 1000)

	// noi_dungxử lýTTSkết nối，hỗ trợnoi_dung
	go func() {
		defer close(outputChan)

		retryCount := 0
		maxRetries := 2
		var lastError error

		// noi_dungmaxRetriesnoi_dung
		for retryCount <= maxRetries {
			if retryCount > 0 {
				log.Infof("noi_dungmớiLấy kết nối，noi_dung %d/%d noi_dung", retryCount, maxRetries)

				// noi_dungtrướckiểm tranoi_dungđãhủy
				select {
				case <-ctx.Done():
					log.Debugf("Context đã hủy，noi_dung")
					return
				default:
					// noi_dung
				}
			}

			// xử lýTTSkết nối
			err := p.handleTTSConnection(ctx, text, outputChan)

			if err == nil {
				// kết nốixử lýthành công，noi_dung
				return
			}

			lastError = err
			log.Errorf("TTSkết nốixử lýthất bại: %v (noi_dung: %d/%d)", err, retryCount, maxRetries)

			retryCount++
		}

		if retryCount > maxRetries {
			log.Warnf("noi_dungtớinoi_dung %d，noi_dung，noi_dungsaulỗi: %v", maxRetries, lastError)
		}
	}()

	return outputChan, nil
}

// GetVoiceInfo lấyTTSconfignoi_dung
func (p *XiaozhiProvider) GetVoiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":         "xiaozhi_ws",
		"server_addr":  p.ServerAddr,
		"device_id":    p.DeviceID,
		"audio_format": p.AudioFormat,
	}
}

// SetVoice Thiết lập tham số voice（Xiaozhi Provider noi_dunghỗ trợđộngnoi_dungvoice）
func (p *XiaozhiProvider) SetVoice(voiceConfig map[string]interface{}) error {
	return fmt.Errorf("Xiaozhi TTS Provider noi_dunghỗ trợđộngnoi_dungvoice")
}

// Close đóngtài nguyên（noi_dungtrạng thái Provider，noi_dungđóng）
func (p *XiaozhiProvider) Close() error {
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *XiaozhiProvider) IsValid() bool {
	return p != nil
}

// TextToSpeech triển khai BaseTTSProvider interface，trực tiếpnoi_dungstreamingframe
func (p *XiaozhiProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	ch, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	var frames [][]byte
	for frame := range ch {
		frames = append(frames, frame)
	}
	return frames, nil
}
