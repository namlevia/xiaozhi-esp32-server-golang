package server

import (
	"context"
	"hash/fnv"
	"sync"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	log "xiaozhi-esp32-server-golang/logger"
)

// EventWrapper bọc event để xử lý thống nhất nhiều loại event.
type EventWrapper struct {
	Topic string      // Tên topic
	Data  interface{} // Dữ liệu event
}

// TopicHandler là interface xử lý topic dùng chung.
type TopicHandler interface {
	// Process xử lý event.
	Process(ctx context.Context, data interface{}) error
	// GetRoutingKey lấy key dùng để hash route, thường là DeviceID hoặc SessionID.
	GetRoutingKey(data interface{}) string
}

// UnifiedWorkerPool là worker pool thống nhất, có thể xử lý nhiều topic.
type UnifiedWorkerPool struct {
	workers   []chan *EventWrapper
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	handlers  map[string]TopicHandler // Map topic -> handler
	workerNum int
	mu        sync.RWMutex // Bảo vệ handlers map
}

// NewUnifiedWorkerPool tạo worker pool thống nhất.
func NewUnifiedWorkerPool(workerNum int) *UnifiedWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &UnifiedWorkerPool{
		workers:   make([]chan *EventWrapper, workerNum),
		ctx:       ctx,
		cancel:    cancel,
		handlers:  make(map[string]TopicHandler),
		workerNum: workerNum,
	}

	// Khởi tạo channel của từng worker và start goroutine.
	for i := 0; i < workerNum; i++ {
		pool.workers[i] = make(chan *EventWrapper, 100) // Buffer 100 message
		pool.wg.Add(1)
		go pool.workerLoop(i)
	}

	log.Infof("UnifiedWorkerPool khởi tạo xong, đã start %d worker goroutine (có thể xử lý nhiều topic)", workerNum)
	return pool
}

// RegisterHandler đăng ký handler cho topic.
func (p *UnifiedWorkerPool) RegisterHandler(topic string, handler TopicHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[topic] = handler
	log.Infof("UnifiedWorkerPool: đã đăng ký topic handler [%s]", topic)
}

// workerLoop là vòng lặp xử lý của từng worker, đảm bảo xử lý tuần tự.
func (p *UnifiedWorkerPool) workerLoop(index int) {
	defer p.wg.Done()
	defer log.Infof("UnifiedWorkerPool worker %d đã thoát", index)

	ch := p.workers[index]
	for {
		select {
		case <-p.ctx.Done():
			// Dọn message còn lại trong channel.
			for {
				select {
				case event := <-ch:
					if event != nil {
						p.processEvent(event)
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
				p.processEvent(event)
			}
		}
	}
}

// processEvent xử lý event, dispatch tới handler tương ứng theo topic.
func (p *UnifiedWorkerPool) processEvent(event *EventWrapper) {
	p.mu.RLock()
	handler, exists := p.handlers[event.Topic]
	p.mu.RUnlock()

	if !exists {
		log.Warnf("UnifiedWorkerPool: topic [%s] chưa đăng ký handler, bỏ qua", event.Topic)
		return
	}

	if err := handler.Process(context.Background(), event.Data); err != nil {
		log.Errorf("UnifiedWorkerPool: xử lý topic [%s] thất bại: %v", event.Topic, err)
	}
}

// Route route event tới worker tương ứng bằng hash distribution.
func (p *UnifiedWorkerPool) Route(topic string, data interface{}) bool {
	p.mu.RLock()
	handler, exists := p.handlers[topic]
	p.mu.RUnlock()

	if !exists {
		log.Warnf("UnifiedWorkerPool: topic [%s] chưa đăng ký handler, không thể route", topic)
		return false
	}

	// Lấy routing key.
	key := handler.GetRoutingKey(data)
	if key == "" {
		log.Warnf("UnifiedWorkerPool: routing key của topic [%s] rỗng, không thể route message", topic)
		return false
	}

	// Tính hash và route tới worker tương ứng.
	workerIndex := p.hashKey(key)

	// Tạo event wrapper.
	event := &EventWrapper{
		Topic: topic,
		Data:  data,
	}

	// Gửi non-blocking vào worker channel tương ứng.
	select {
	case p.workers[workerIndex] <- event:
		return true
	default:
		log.Warnf("UnifiedWorkerPool: channel của worker %d cho topic [%s] đã đầy, bỏ message, key: %s",
			topic, workerIndex, key)
		return false
	}
}

// hashKey tính hash của key và trả về worker index.
func (p *UnifiedWorkerPool) hashKey(key string) int {
	if key == "" {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	hash := h.Sum32()
	return int(hash) % p.workerNum
}

// Close đóng worker pool.
func (p *UnifiedWorkerPool) Close() {
	p.cancel()
	p.wg.Wait()

	// Đóng toàn bộ worker channel.
	for i := 0; i < p.workerNum; i++ {
		close(p.workers[i])
	}

	log.Info("UnifiedWorkerPool đã đóng")
}

type EventHandle struct {
	// Worker pool thống nhất, có thể xử lý nhiều topic.
	workerPool *UnifiedWorkerPool
	// Tham chiếu App để lấy ChatManager.
	app *App
}

// SessionEndHandler xử lý event SessionEnd.
type SessionEndHandler struct{}

func (h *SessionEndHandler) Process(ctx context.Context, data interface{}) error {
	clientState, ok := data.(*ClientState)
	if !ok || clientState == nil {
		return nil
	}

	if clientState.MemoryProvider == nil {
		return nil
	}
	if clientState.GetMemoryMode() != MemoryModeLong {
		return nil
	}

	log.Debugf("HandleSessionEnd: deviceId: %s", clientState.DeviceID)

	// Flush message vào bộ nhớ dài hạn.
	err := clientState.MemoryProvider.Flush(
		clientState.Ctx,
		clientState.GetDeviceIDOrAgentID())
	if err != nil {
		log.Errorf("flush message to memory provider failed: %v", err)
		return err
	}
	return nil
}

func (h *SessionEndHandler) GetRoutingKey(data interface{}) string {
	clientState, ok := data.(*ClientState)
	if !ok || clientState == nil {
		return ""
	}
	return clientState.DeviceID
}

// ExitChatHandler xử lý event ExitChat.
type ExitChatHandler struct {
	eventHandle *EventHandle // Giữ tham chiếu EventHandle để truy cập App.
}

func (h *ExitChatHandler) Process(ctx context.Context, data interface{}) error {
	event, ok := data.(*eventbus.ExitChatEvent)
	if !ok || event == nil {
		return nil
	}

	clientState := event.ClientState
	if clientState == nil {
		return nil
	}

	log.Debugf("Xử lý event thoát chat: device_id: %s, reason: %s, trigger: %s, user_text: %s",
		clientState.DeviceID, event.Reason, event.TriggerType, event.UserText)

	// Lấy ChatManager theo deviceId.
	if h.eventHandle == nil || h.eventHandle.app == nil {
		log.Warnf("EventHandle hoặc App chưa khởi tạo, không thể lấy ChatManager")
		return nil
	}

	chatManager, exists := h.eventHandle.app.GetChatManager(clientState.DeviceID)
	if !exists {
		log.Warnf("Không tìm thấy ChatManager của thiết bị %s, có thể đã đóng", clientState.DeviceID)
		return nil
	}

	return chatManager.ExitChat()
}

func (h *ExitChatHandler) GetRoutingKey(data interface{}) string {
	event, ok := data.(*eventbus.ExitChatEvent)
	if !ok || event == nil || event.ClientState == nil {
		return ""
	}
	return event.ClientState.DeviceID
}

func NewEventHandle(app *App) (*EventHandle, error) {
	// Tạo worker pool thống nhất.
	workerPool := NewUnifiedWorkerPool(MessageWorkerNum)

	// Đăng ký handler SessionEnd.
	sessionEndHandler := &SessionEndHandler{}
	workerPool.RegisterHandler(eventbus.TopicSessionEnd, sessionEndHandler)

	handle := &EventHandle{
		workerPool: workerPool,
		app:        app,
	}

	// Đăng ký handler ExitChat.
	exitChatHandler := &ExitChatHandler{
		eventHandle: handle,
	}
	workerPool.RegisterHandler(eventbus.TopicExitChat, exitChatHandler)

	log.Infof("EventHandle khởi tạo xong (dùng worker pool thống nhất để xử lý nhiều topic, xử lý Redis đã chuyển sang MessageWorker)")
	return handle, nil
}

func (s *EventHandle) Start() error {
	// Subscribe event SessionEnd.
	go s.HandleSessionEnd()

	// Subscribe event ExitChat.
	go s.HandleExitChat()

	// Có thể thêm subscription topic khác tại đây.
	// go s.HandleDeviceOnline()

	return nil
}

// HandleSessionEnd subscribe và xử lý event SessionEnd.
func (s *EventHandle) HandleSessionEnd() error {
	eventbus.Get().Subscribe(eventbus.TopicSessionEnd, func(clientState *ClientState) {
		if clientState == nil {
			log.Warnf("HandleSessionEnd: clientState is nil, skipping")
			return
		}

		// Route tới worker pool thống nhất.
		s.workerPool.Route(eventbus.TopicSessionEnd, clientState)
	})
	return nil
}

// HandleExitChat subscribe và xử lý event ExitChat.
func (s *EventHandle) HandleExitChat() error {
	eventbus.Get().Subscribe(eventbus.TopicExitChat, func(event *eventbus.ExitChatEvent) {
		if event == nil {
			log.Warnf("HandleExitChat: event is nil, skipping")
			return
		}

		// Route tới worker pool thống nhất.
		s.workerPool.Route(eventbus.TopicExitChat, event)
	})
	return nil
}

// RegisterTopic đăng ký handler cho topic mới, là helper tiện dụng.
func (s *EventHandle) RegisterTopic(topic string, handler TopicHandler) {
	s.workerPool.RegisterHandler(topic, handler)
}

// Close đóng EventHandle và graceful shutdown worker pool.
func (s *EventHandle) Close() {
	if s.workerPool != nil {
		s.workerPool.Close()
	}
	log.Info("EventHandle đã đóng")
}
