package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"runtime"
	"sync"
	"time"

	data_client "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/data/history"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/memory/llm_memory"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

var (
	// MessageWorkerNum là số worker xử lý message, tính theo số CPU core và dùng chung cho Redis+History.
	// Phải là lũy thừa của 2 để phân phối hash.
	MessageWorkerNum = getMessageWorkerNum()
)

// getMessageWorkerNum tính số worker theo số CPU core, làm tròn lên lũy thừa 2 gần nhất.
// Giá trị tối thiểu là 4, tối đa là 64.
func getMessageWorkerNum() int {
	cpuNum := runtime.NumCPU()

	// Giá trị tối thiểu là 4, tối đa là 64.
	if cpuNum < 4 {
		return 4
	}
	if cpuNum > 64 {
		return 64
	}

	// Làm tròn lên lũy thừa 2 gần nhất.
	power := 1
	for power < cpuNum {
		power <<= 1
	}
	return power
}

// MessageWorker xử lý message.
// Dùng pool goroutine cố định, route theo hash của SessionID để đảm bảo message cùng session được xử lý theo thứ tự.
// Xử lý thống nhất message Redis, MemoryProvider và History.
type MessageWorker struct {
	client  *history.HistoryClient
	workers []chan *eventbus.AddMessageEvent // Channel của từng worker
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewMessageWorker tạo message processor.
func NewMessageWorker(cfg history.HistoryClientConfig) *MessageWorker {
	client := history.NewHistoryClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	worker := &MessageWorker{
		client:  client,
		workers: make([]chan *eventbus.AddMessageEvent, MessageWorkerNum),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Khởi tạo channel của từng worker và start goroutine.
	for i := 0; i < MessageWorkerNum; i++ {
		worker.workers[i] = make(chan *eventbus.AddMessageEvent, 100) // Buffer 100 message
		worker.wg.Add(1)
		go worker.workerLoop(i)
	}

	worker.subscribeEvents()
	log.Infof("MessageWorker khởi tạo xong, đã start %d worker goroutine (xử lý thống nhất Redis+MemoryProvider+History)", MessageWorkerNum)
	return worker
}

// workerLoop là vòng lặp xử lý của từng worker, đảm bảo xử lý tuần tự.
func (w *MessageWorker) workerLoop(index int) {
	defer w.wg.Done()
	defer log.Infof("MessageWorker worker %d đã thoát", index)

	ch := w.workers[index]
	for {
		select {
		case <-w.ctx.Done():
			// Dọn message còn lại trong channel.
			for {
				select {
				case event := <-ch:
					if event != nil {
						w.processMessage(event)
					}
				default:
					return
				}
			}
		case event, ok := <-ch:
			if !ok {
				// Channel đã đóng.
				return
			}
			if event != nil {
				w.processMessage(event)
			}
		}
	}
}

// processMessage xử lý message tuần tự trong worker goroutine.
// Xử lý thống nhất Redis, MemoryProvider và History, đảm bảo message cùng thiết bị/session được xử lý theo thứ tự.
func (w *MessageWorker) processMessage(event *eventbus.AddMessageEvent) {
	// 1. Xử lý History cho toàn bộ message.
	// Dùng context độc lập, không phụ thuộc event.ClientState.Ctx, để lưu history không bị ảnh hưởng khi hội thoại bị cancel.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Xác định là thêm mới hay cập nhật.
	if event.IsUpdate {
		// Giai đoạn 2: cập nhật audio.
		w.updateMessageAudio(ctx, event)
	} else {
		// Giai đoạn 1: lưu text message, gồm xử lý Redis.
		w.saveMessageText(ctx, event)
	}

	// 2. Xử lý MemoryProvider khi !IsUpdate, độc lập với redis và manager.
	// Xử lý bộ nhớ dài hạn (memobase/mem0), cần cho cả kịch bản redis và manager.
	if !event.IsUpdate {
		w.processMemoryProvider(event)
	}
}

// processMemoryProvider xử lý bộ nhớ dài hạn (memobase/mem0).
// Độc lập với redis và manager, cần xử lý trong cả hai kịch bản.
func (w *MessageWorker) processMemoryProvider(event *eventbus.AddMessageEvent) {
	clientState := event.ClientState
	if clientState.MemoryProvider == nil {
		return
	}
	if clientState.GetMemoryMode() != data_client.MemoryModeLong {
		return
	}

	err := clientState.MemoryProvider.AddMessage(
		clientState.Ctx,
		clientState.GetDeviceIDOrAgentID(),
		event.Msg)
	if err != nil {
		log.Errorf("add message to memory provider failed: %v", err)
	}
}

// hashSessionID tính hash của SessionID và trả về worker index.
func (w *MessageWorker) hashSessionID(sessionID string) int {
	if sessionID == "" {
		return 0 // Nếu SessionID rỗng, dùng worker đầu tiên.
	}

	// Dùng hàm hash FNV-1a.
	h := fnv.New32a()
	h.Write([]byte(sessionID))
	hash := h.Sum32()
	return int(hash) % MessageWorkerNum
}

// subscribeEvents subscribe event từ EventBus.
func (w *MessageWorker) subscribeEvents() {
	bus := eventbus.Get()
	// Subscribe event thêm message thống nhất, cùng Topic với EventHandle.
	bus.Subscribe(eventbus.TopicAddMessage, w.handleAddMessage)
}

// handleAddMessage xử lý thống nhất event thêm message và route tới worker tương ứng.
func (w *MessageWorker) handleAddMessage(event *eventbus.AddMessageEvent) {
	if event == nil || event.ClientState == nil {
		return
	}

	// Xác định key để route: ưu tiên SessionID, nếu rỗng thì dùng DeviceID.
	key := event.ClientState.SessionID
	if key == "" {
		key = event.ClientState.DeviceID
	}
	if key == "" {
		log.Warnf("SessionID và DeviceID đều rỗng, không thể route message")
		return
	}

	// Tính hash và route tới worker tương ứng.
	workerIndex := w.hashSessionID(key)

	// Gửi non-blocking vào worker channel tương ứng.
	select {
	case w.workers[workerIndex] <- event:
		// Gửi thành công.
	default:
		// Channel đã đầy, ghi cảnh báo; thường không xảy ra vì channel có buffer.
		log.Warnf("Channel của worker %d đã đầy, bỏ message, session_id: %s, device_id: %s",
			workerIndex, event.ClientState.SessionID, event.ClientState.DeviceID)
	}
}

// saveMessageText lưu text message trong giai đoạn 1, hoặc lưu một lần cả text+audio.
// Gồm xử lý Redis khi config_provider.type là redis.
func (w *MessageWorker) saveMessageText(ctx context.Context, event *eventbus.AddMessageEvent) {
	// Xử lý Redis chỉ khi config_provider.type là redis.
	// Thêm vào danh sách message Redis dùng cho LLM context.
	providerType := viper.GetString("config_provider.type")
	if providerType == "redis" {
		clientState := event.ClientState
		llm_memory.Get().AddMessage(
			clientState.Ctx,
			clientState.DeviceID,
			clientState.AgentID,
			event.Msg)
		return
	}

	// Xác định role của message.
	var role history.MessageType
	switch event.Msg.Role {
	case schema.User:
		role = history.MessageTypeUser
	case schema.Assistant:
		role = history.MessageTypeAssistant
	case schema.Tool:
		role = history.MessageTypeTool
	case schema.System:
		role = history.MessageTypeSystem
	default:
		log.Warnf("Role message không được hỗ trợ: %s", event.Msg.Role)
		return
	}

	// Chuyển đổi định dạng audio nếu có.
	var audioBase64 string
	var audioFormat string
	var audioSize int

	if len(event.AudioData) > 0 {
		// Message ASR: lấy text và audio cùng lúc, lưu một lần.
		var wavData []byte
		var err error

		// Chọn cách chuyển đổi audio theo role của message.
		if event.Msg.Role == schema.User {
			// Message User (ASR): định dạng PCM float32.
			if len(event.AudioData) > 0 {
				wavData, err = util.PCMFloat32BytesToWav(
					event.AudioData[0], // Message User chỉ có một phần tử.
					event.SampleRate,
					event.Channels)
			}
		} else {
			// Message Assistant (TTS): định dạng Opus; về lý thuyết không nên vào đây vì Assistant lưu hai giai đoạn.
			wavData, err = util.OpusFramesToWav(
				event.AudioData,
				event.SampleRate,
				event.Channels)
		}

		if err != nil {
			log.Errorf("Chuyển đổi audio thất bại, device_id: %s, message_id: %s, role: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, event.Msg.Role, err)
			// Fallback: nối trực tiếp toàn bộ frame.
			var fallbackData []byte
			for _, frame := range event.AudioData {
				fallbackData = append(fallbackData, frame...)
			}
			audioBase64 = base64.StdEncoding.EncodeToString(fallbackData)
			audioSize = event.AudioSize
			audioFormat = "raw" // Fallback dùng định dạng raw.
		} else {
			audioBase64 = base64.StdEncoding.EncodeToString(wavData)
			audioSize = len(wavData)
			audioFormat = "wav"
		}
	}

	// Tạo Metadata, chỉ lưu timestamp.
	metadata := map[string]interface{}{
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}

	// Chuẩn bị các field liên quan đến tool call.
	var toolCallID string
	var toolCallsJSON *string

	// Role Tool: lưu tool_call_id.
	if event.Msg.Role == schema.Tool && event.Msg.ToolCallID != "" {
		toolCallID = event.Msg.ToolCallID
	}

	// Role Assistant: lưu ToolCalls nếu có.
	if event.Msg.Role == schema.Assistant && len(event.Msg.ToolCalls) > 0 {
		// Serialize ToolCalls thành chuỗi JSON.
		toolCallsBytes, err := json.Marshal(event.Msg.ToolCalls)
		if err != nil {
			log.Warnf("Serialize ToolCalls thất bại, device_id: %s, message_id: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, err)
		} else {
			jsonStr := string(toolCallsBytes)
			toolCallsJSON = &jsonStr
		}
	}

	req := &history.SaveMessageRequest{
		MessageID:     event.MessageID,
		DeviceID:      event.ClientState.DeviceID,
		AgentID:       event.ClientState.AgentID,
		SessionID:     event.ClientState.SessionID,
		Role:          role,
		Content:       event.Msg.Content,
		ToolCallID:    toolCallID,
		ToolCallsJSON: toolCallsJSON,
		AudioData:     audioBase64,
		AudioFormat:   audioFormat,
		AudioSize:     audioSize,
		Metadata:      metadata,
	}

	if err := w.client.SaveMessage(ctx, req); err != nil {
		log.Errorf("Lưu message thất bại, device_id: %s, message_id: %s, error: %v",
			event.ClientState.DeviceID, event.MessageID, err)
	}
}

// updateMessageAudio cập nhật audio của message trong giai đoạn 2.
func (w *MessageWorker) updateMessageAudio(ctx context.Context, event *eventbus.AddMessageEvent) {
	// Chuyển đổi định dạng audio.
	var audioBase64 string
	var audioSize int

	if len(event.AudioData) > 0 {
		var wavData []byte
		var err error

		// Chọn cách chuyển đổi audio theo role của message.
		// Message User (ASR): định dạng PCM float32, dùng PCMFloat32BytesToWav.
		// Message Assistant (TTS): định dạng Opus, dùng OpusFramesToWav.
		if event.Msg.Role == schema.User {
			// Message User: định dạng PCM float32.
			// event.AudioData là [][]byte nhưng message User chỉ có một phần tử, là mảng byte PCM float32 đầy đủ.
			if len(event.AudioData) > 0 {
				wavData, err = util.PCMFloat32BytesToWav(
					event.AudioData[0], // Message User chỉ có một phần tử.
					event.SampleRate,
					event.Channels)
			}
		} else {
			// Message Assistant: định dạng Opus.
			wavData, err = util.OpusFramesToWav(
				event.AudioData,
				event.SampleRate,
				event.Channels)
		}

		if err != nil {
			log.Errorf("Chuyển đổi audio thất bại, device_id: %s, message_id: %s, role: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, event.Msg.Role, err)
			// Fallback: nối trực tiếp toàn bộ frame.
			var fallbackData []byte
			for _, frame := range event.AudioData {
				fallbackData = append(fallbackData, frame...)
			}
			audioBase64 = base64.StdEncoding.EncodeToString(fallbackData)
			audioSize = event.AudioSize
		} else {
			audioBase64 = base64.StdEncoding.EncodeToString(wavData)
			audioSize = len(wavData)
		}
	}

	// Tạo request cập nhật.
	req := &history.UpdateMessageAudioRequest{
		MessageID:   event.MessageID,
		AudioData:   audioBase64,
		AudioFormat: "wav",
		AudioSize:   audioSize,
		Metadata: map[string]interface{}{
			"tts_duration": event.TTSDuration,
		},
	}

	// Gọi API cập nhật.
	if err := w.client.UpdateMessageAudio(ctx, req); err != nil {
		log.Errorf("Cập nhật audio message thất bại, device_id: %s, message_id: %s, error: %v",
			event.ClientState.DeviceID, event.MessageID, err)
	}
}
