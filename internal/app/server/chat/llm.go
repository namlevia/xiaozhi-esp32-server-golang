package chat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	. "xiaozhi-esp32-server-golang/internal/data/client"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"
	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/pool"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

const (
	MaxMessageCount = 10

	McpReadResourcePageSize       = 100 * 1024
	McpReadResourceStreamDoneFlag = "[DONE]"
)

// Kiểu context key dùng để tránh xung đột
type contextKey int

const (
	ttsPlaybackCompletionGrace time.Duration = 150 * time.Millisecond
	fullTextKey                contextKey    = iota
	toolRoundMessagesKey
	ttsTurnTrackerKey
	ttsPlaybackStartHookKey
	ttsTurnEndPolicyKey
	ttsTurnEndPolicyHandlerKey
	ttsTurnPlaybackSettledKey
)

const (
	interruptExtraKey      = "interrupt"
	interruptByExtraKey    = "interrupt_by"
	interruptStageExtraKey = "interrupt_stage"
	interruptContentSuffix = " [người dùng ngắt]"
)

// GetLastMessageID lấy MessageID của message lưu gần nhất (dùng cho lưu hai giai đoạn)
func (l *LLMManager) GetLastMessageID(role string) (string, bool) {
	l.lastMessageIDMu.RLock()
	defer l.lastMessageIDMu.RUnlock()
	id, ok := l.lastMessageID[role]
	return id, ok
}

type LLMResponseChannelItem struct {
	ctx          context.Context
	userMessage  *schema.Message
	responseChan chan llm_common.LLMResponseStruct
	onStartFunc  func(args ...any)
	onEndFunc    func(err error, args ...any)
}

type llmHandleResult struct {
	ok                      bool
	suppressProtocolTtsStop bool
}

func llmHandleResultFromArgs(args []any) llmHandleResult {
	if len(args) == 0 {
		return llmHandleResult{}
	}
	result, ok := args[0].(llmHandleResult)
	if !ok {
		return llmHandleResult{}
	}
	return result
}

func (l *LLMManager) finishTTSTurn(ctx context.Context, stopErr error, result llmHandleResult) {
	l.finishTTSTurnWithReason(ctx, stopErr, result, "LLMManager.finishTTSTurn")
}

func (l *LLMManager) finishTTSTurnWithReason(ctx context.Context, stopErr error, result llmHandleResult, reason string) {
	if l == nil || l.ttsManager == nil {
		return
	}

	if result.suppressProtocolTtsStop {
		// Tool media sẽ chờ phát xong rồi quay lại đây để kết thúc; lúc này vẫn cần gửi bù tts_stop cấp protocol,
		// nếu không client sẽ kẹt ở trạng thái “đang nói”.
		log.Debugf("Media output đã hoàn tất, dùng flow kết thúc TTS thông thường để gửi tts stop")
	}

	l.ttsManager.EnqueueTtsStopWithReason(ctx, reason)
	l.ttsManager.RequestTurnEnd(ctx, stopErr)
}

type llmResponseChannelOptions struct {
	disableTTSCommands bool
	onStartFunc        func(args ...any)
	onEndFunc          func(err error, args ...any)
	onTTSPlaybackStart func()
	ttsTurnEndPolicy   ttsTurnEndPolicy
}

type ttsPlaybackStartHook func()

type ttsTurnEndPolicy uint8

const (
	ttsTurnEndPolicyNone ttsTurnEndPolicy = iota
	ttsTurnEndPolicyGoodbyeAndIdle
)

type ttsTurnEndPolicyHandler interface {
	handleTTSTurnEndPolicy(ctx context.Context, policy ttsTurnEndPolicy, stopErr error)
}

func withTTSPlaybackStartHook(ctx context.Context, hook func()) context.Context {
	if ctx == nil || hook == nil {
		return ctx
	}

	var once sync.Once
	return context.WithValue(ctx, ttsPlaybackStartHookKey, ttsPlaybackStartHook(func() {
		once.Do(hook)
	}))
}

func ttsPlaybackStartHookFromContext(ctx context.Context) func() {
	if ctx == nil {
		return nil
	}
	hook, ok := ctx.Value(ttsPlaybackStartHookKey).(ttsPlaybackStartHook)
	if !ok || hook == nil {
		return nil
	}
	return func() {
		hook()
	}
}

func withTTSTurnEndPolicy(ctx context.Context, policy ttsTurnEndPolicy) context.Context {
	if ctx == nil || policy == ttsTurnEndPolicyNone {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnEndPolicyKey, policy)
}

func ttsTurnEndPolicyFromContext(ctx context.Context) ttsTurnEndPolicy {
	if ctx == nil {
		return ttsTurnEndPolicyNone
	}
	policy, ok := ctx.Value(ttsTurnEndPolicyKey).(ttsTurnEndPolicy)
	if !ok {
		return ttsTurnEndPolicyNone
	}
	return policy
}

func withTTSTurnEndPolicyHandler(ctx context.Context, handler ttsTurnEndPolicyHandler) context.Context {
	if ctx == nil || handler == nil {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnEndPolicyHandlerKey, handler)
}

func withTTSTurnPlaybackSettled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ttsTurnPlaybackSettledKey, true)
}

func ttsTurnPlaybackSettledFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	settled, ok := ctx.Value(ttsTurnPlaybackSettledKey).(bool)
	return ok && settled
}

func ttsTurnEndPolicyHandlerFromContext(ctx context.Context) ttsTurnEndPolicyHandler {
	if ctx == nil {
		return nil
	}
	handler, ok := ctx.Value(ttsTurnEndPolicyHandlerKey).(ttsTurnEndPolicyHandler)
	if !ok {
		return nil
	}
	return handler
}

type ttsTurnTracker struct {
	mu      sync.Mutex
	pending int
	doneCh  chan struct{}
}

func newTTSTurnTracker() *ttsTurnTracker {
	doneCh := make(chan struct{})
	close(doneCh)
	return &ttsTurnTracker{doneCh: doneCh}
}

func (t *ttsTurnTracker) Add() func(error) {
	if t == nil {
		return func(error) {}
	}

	t.mu.Lock()
	if t.pending == 0 {
		t.doneCh = make(chan struct{})
	}
	t.pending++
	t.mu.Unlock()

	var once sync.Once
	return func(error) {
		once.Do(func() {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.pending == 0 {
				return
			}
			t.pending--
			if t.pending == 0 {
				close(t.doneCh)
			}
		})
	}
}

func (t *ttsTurnTracker) Wait(ctx context.Context) error {
	if t == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	t.mu.Lock()
	pending := t.pending
	doneCh := t.doneCh
	t.mu.Unlock()

	if pending == 0 {
		return nil
	}

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type LLMManager struct {
	clientState       *ClientState
	session           *ChatSession
	serverTransport   *ServerTransport
	ttsManager        *TTSManager
	transformRegistry *streamtransform.Registry

	einoTools []*schema.ToolInfo

	llmResponseQueue *util.Queue[LLMResponseChannelItem]

	// Lưu MessageID của message lưu gần nhất (dùng cho lưu hai giai đoạn)
	// key: role (user/assistant), value: MessageID
	lastMessageID   map[string]string
	lastMessageIDMu sync.RWMutex // bảo vệ truy cập đồng thời lastMessageID
}

func NewLLMManager(clientState *ClientState, serverTransport *ServerTransport, ttsManager *TTSManager, session *ChatSession, transformRegistry *streamtransform.Registry) *LLMManager {
	return &LLMManager{
		clientState:       clientState,
		session:           session,
		serverTransport:   serverTransport,
		ttsManager:        ttsManager,
		transformRegistry: transformRegistry,
		llmResponseQueue:  util.NewQueue[LLMResponseChannelItem](10),
		lastMessageID:     make(map[string]string),
	}
}

func (l *LLMManager) openOutputPipeline(ctx context.Context) (*streamtransform.Pipeline, error) {
	if l == nil || l.transformRegistry == nil {
		return &streamtransform.Pipeline{}, nil
	}

	sessionID := ""
	deviceID := ""
	if l.clientState != nil {
		sessionID = l.clientState.SessionID
		deviceID = l.clientState.DeviceID
	}

	return l.transformRegistry.Open(streamtransform.Context{
		Ctx:       ctx,
		SessionID: sessionID,
		DeviceID:  deviceID,
		RequestID: fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano()),
	})
}

func (l *LLMManager) emitLLMOutputRaw(ctx context.Context, data chathooks.LLMOutputRawData) (chathooks.LLMOutputRawData, bool, error) {
	if l == nil || l.session == nil || l.session.hookHub == nil {
		return data, false, nil
	}
	return l.session.hookHub.EmitLLMOutputRaw(l.session.hookContext(ctx), data)
}

// handleLLMWithContextAndTools dùng context control để xử lý response LLM (tương thích có tool và không tool)
// Bên trong tự động quản lý lấy và release resource LLM
func (l *LLMManager) handleLLMWithContextAndTools(
	ctx context.Context,
	dialogue []*schema.Message,
	tools []*schema.ToolInfo,
) (chan llm_common.LLMResponseStruct, error) {
	// Lấy resource LLM
	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		l.clientState.DeviceConfig.Llm.Provider,
		l.clientState.DeviceConfig.Llm.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("Lấy resource LLM thất bại: %w", err)
	}

	// Lấy provider
	llmProvider := llmWrapper.GetProvider()

	// Gọi LLM provider
	msgChan := llmProvider.ResponseWithContext(ctx, l.clientState.SessionID, dialogue, tools)

	pipeline, err := l.openOutputPipeline(ctx)
	if err != nil {
		pool.Release(llmWrapper)
		return nil, fmt.Errorf("Tạo pipeline transform output stream LLM thất bại: %w", err)
	}

	// Tạo response channel
	responseChannel := make(chan llm_common.LLMResponseStruct, 2)
	startTs := time.Now().UnixMilli()
	var firstSegment bool
	var rawFullText strings.Builder

	// Khởi động goroutine xử lý response
	go func() {
		defer func() {
			log.Debugf("full Response with %d tools, fullText: %s", len(tools), rawFullText.String())
			close(responseChannel)
			if closeErr := pipeline.Close(); closeErr != nil {
				log.Warnf("Đóng pipeline transform output stream LLM thất bại: %v", closeErr)
			}
			// Release resource
			pool.Release(llmWrapper)
			log.Debugf("Resource LLM đã release")
		}()

		isFirstOutput := true
		llmFirstTokenMarked := false

		emitResponse := func(item streamtransform.Item) bool {
			response := llm_common.LLMResponseStruct{
				IsEnd: item.IsEnd,
			}

			switch item.Kind {
			case streamtransform.ItemKindToolCalls:
				response.ToolCalls = item.ToolCalls
				if len(item.ToolCalls) > 0 {
					response.IsStart = isFirstOutput
				}
			case streamtransform.ItemKindTextDelta, streamtransform.ItemKindTextSegment:
				response.Text = item.Text
				if strings.TrimSpace(item.Text) != "" {
					response.IsStart = isFirstOutput
					if !firstSegment {
						firstSegment = true
						firstSentenceTs := time.Now().UnixMilli()
						if l.clientState.MarkLlmFirstSentenceAt(firstSentenceTs) && l.session != nil {
							l.session.TraceLlmFirstSentence(ctx, firstSentenceTs)
						}
						log.Infof("Thống kê thời gian: câu LLM đầu: %d ms", firstSentenceTs-startTs)
					}
					if isFirstOutput {
						isFirstOutput = false
					}
				}
			default:
				return true
			}

			if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 && !response.IsEnd {
				return true
			}

			select {
			case <-ctx.Done():
				log.Infof("Context đã cancel, dừng xử lý response LLM: %v, context done, exit", ctx.Err())
				return false
			case responseChannel <- response:
				return true
			}
		}

		pushToPipeline := func(item streamtransform.Item) (bool, error) {
			items, stop, err := pipeline.Push(item)
			if err != nil {
				return false, err
			}
			for _, out := range items {
				if !emitResponse(out) {
					return true, nil
				}
			}
			return stop, nil
		}

		pushRawText := func(delta string, isEnd bool, errVal error) (bool, error) {
			payload, stop, hookErr := l.emitLLMOutputRaw(ctx, chathooks.LLMOutputRawData{
				Delta:    delta,
				FullText: rawFullText.String(),
				IsEnd:    isEnd,
				Err:      errVal,
			})
			if hookErr != nil {
				log.Warnf("LLM_OUTPUT_RAW hook thực thi thất bại: %v", hookErr)
			}
			if stop {
				log.Infof("LLM_OUTPUT_RAW hook yêu cầu dừng flow hiện tại")
				return true, nil
			}
			if payload.Delta != "" {
				rawFullText.WriteString(payload.Delta)
			}
			return pushToPipeline(streamtransform.Item{
				Kind:  streamtransform.ItemKindTextDelta,
				Text:  payload.Delta,
				IsEnd: payload.IsEnd,
			})
		}

		pushRawToolCalls := func(toolCalls []schema.ToolCall) (bool, error) {
			payload, stop, hookErr := l.emitLLMOutputRaw(ctx, chathooks.LLMOutputRawData{
				FullText:  rawFullText.String(),
				ToolCalls: toolCalls,
			})
			if hookErr != nil {
				log.Warnf("LLM_OUTPUT_RAW hook thực thi thất bại: %v", hookErr)
			}
			if stop {
				log.Infof("LLM_OUTPUT_RAW hook yêu cầu dừng flow hiện tại")
				return true, nil
			}
			if len(payload.ToolCalls) == 0 {
				return false, nil
			}
			return pushToPipeline(streamtransform.Item{
				Kind:      streamtransform.ItemKindToolCalls,
				ToolCalls: payload.ToolCalls,
			})
		}

		for {
			select {
			case <-ctx.Done():
				log.Infof("Context đã cancel, dừng xử lý response LLM: %v, context done, exit", ctx.Err())
				return
			case message, ok := <-msgChan:
				if !ok {
					stop, pushErr := pushRawText("", true, nil)
					if pushErr != nil {
						log.Errorf("Xử lý stream kết thúc LLM thất bại: %v", pushErr)
					}
					if stop || pushErr != nil {
						return
					}
					return
				}
				if message == nil {
					continue
				}
				if llm.IsLLMErrorMessage(message) {
					errMsg := llm.LLMErrorMessage(message)
					log.Warnf("LLM trả lỗi: %s", errMsg)
					stop, pushErr := pushRawText(errMsg, true, nil)
					if pushErr != nil {
						log.Errorf("Xử lý output lỗi LLM thất bại: %v", pushErr)
					}
					if stop || pushErr != nil {
						return
					}
					return
				}
				if message.Content != "" {
					if !llmFirstTokenMarked {
						firstTokenTs := time.Now().UnixMilli()
						l.clientState.MarkLlmFirstToken()
						if l.session != nil {
							l.session.TraceLlmFirstToken(ctx, firstTokenTs)
						}
						llmFirstTokenMarked = true
					}
					stop, pushErr := pushRawText(message.Content, false, nil)
					if pushErr != nil {
						log.Errorf("Xử lý stream text LLM thất bại: %v", pushErr)
						return
					}
					if stop {
						return
					}
				}
				if len(message.ToolCalls) > 0 {
					log.Infof("Xử lý tool call: %+v", message.ToolCalls)
					stop, pushErr := pushRawToolCalls(message.ToolCalls)
					if pushErr != nil {
						log.Errorf("Xử lý stream tool LLM thất bại: %v", pushErr)
						return
					}
					if stop {
						return
					}
				}
			}
		}
	}()

	return responseChannel, nil
}

func (l *LLMManager) Start(ctx context.Context) {
	l.processLLMResponseQueue(ctx)
}

func (l *LLMManager) processLLMResponseQueue(ctx context.Context) {
	for {
		item, err := l.llmResponseQueue.Pop(ctx, 0) // blocking
		if err != nil {
			if err == util.ErrQueueCtxDone {
				return
			}
			// Lỗi khác
			continue
		}

		log.Debugf("processLLMResponseQueue item: %+v", item)
		if item.onStartFunc != nil {
			item.onStartFunc()
		}

		// Gọi handleLLMResponse; nó sẽ lấy fullText và toolCalls từ context rồi điền dữ liệu
		result, err := l.handleLLMResponse(item.ctx, item.userMessage, item.responseChan)
		if waitErr := waitForTTSTurnDrainIfRoot(item.ctx); err == nil && waitErr != nil {
			err = waitErr
		}

		if item.onEndFunc != nil {
			item.onEndFunc(err, result)
		}
	}
}

func (l *LLMManager) ClearLLMResponseQueue() {
	l.llmResponseQueue.Clear()
}

func (l *LLMManager) AddTextToTTSQueue(text string) error {
	return l.AddTextToTTSQueueWithOptions(text, llmResponseChannelOptions{})
}

func (l *LLMManager) AddTextToTTSQueueWithOptions(text string, options llmResponseChannelOptions) error {
	log.Debugf("AddTextToTTSQueue text: %s", text)
	msg := &schema.Message{
		Role:    schema.User,
		Content: text,
	}
	llmResponseChan := make(chan llm_common.LLMResponseStruct, 10)
	llmResponseChan <- llm_common.LLMResponseStruct{
		IsStart: true,
		IsEnd:   true,
		Text:    text,
	}
	close(llmResponseChan)

	sessionCtx := l.clientState.SessionCtx.Get(l.clientState.Ctx)
	ctx := l.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	ctx = withTTSPlaybackStartHook(ctx, options.onTTSPlaybackStart)
	ctx = withTTSTurnEndPolicy(ctx, options.ttsTurnEndPolicy)
	if err := l.HandleLLMResponseChannelAsyncWithOptions(ctx, msg, llmResponseChan, options); err != nil {
		log.Warnf("AddTextToTTSQueue enqueue failed: %v", err)
		return err
	}

	return nil
}

func chainLLMResponseStartHooks(hooks ...func(args ...any)) func(args ...any) {
	filtered := make([]func(args ...any), 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			filtered = append(filtered, hook)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func(args ...any) {
		for _, hook := range filtered {
			hook(args...)
		}
	}
}

func chainLLMResponseEndHooks(hooks ...func(err error, args ...any)) func(err error, args ...any) {
	filtered := make([]func(err error, args ...any), 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			filtered = append(filtered, hook)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func(err error, args ...any) {
		for _, hook := range filtered {
			hook(err, args...)
		}
	}
}

func (l *LLMManager) HandleLLMResponseChannelAsync(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct) error {
	return l.handleLLMResponseChannelAsync(ctx, userMessage, responseChan, llmResponseChannelOptions{})
}

func (l *LLMManager) HandleLLMResponseChannelAsyncWithOptions(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct, options llmResponseChannelOptions) error {
	return l.handleLLMResponseChannelAsync(ctx, userMessage, responseChan, options)
}

func (l *LLMManager) handleLLMResponseChannelAsync(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct, options llmResponseChannelOptions) error {
	ctx = ensureTTSTurnTrackerInContext(ctx)
	ctx = withTTSPlaybackStartHook(ctx, options.onTTSPlaybackStart)
	ctx = withTTSTurnEndPolicy(ctx, options.ttsTurnEndPolicy)

	needSendTtsCmd := true
	val := ctx.Value("nest")
	nest := 0
	log.Debugf("AddLLMResponseChannel nest: %+v", val)
	if n, ok := val.(int); ok {
		nest = n
		if nest > 1 {
			needSendTtsCmd = false
		}
	}
	if options.disableTTSCommands {
		needSendTtsCmd = false
	}

	// Khởi tạo hoặc tái dùng fullText trong context (dùng cho lịch sử chat)
	// Nếu context đã có fullText (tiếp tục request LLM sau tool call) thì tái dùng; nếu không thì tạo mới
	var fullText *strings.Builder
	if existingFullText, ok := ctx.Value(fullTextKey).(*strings.Builder); ok && existingFullText != nil {
		fullText = existingFullText
		log.Debugf("Tái dùng fullText hiện có, độ dài hiện tại: %d", fullText.Len())
	} else {
		fullText = &strings.Builder{}
		ctx = context.WithValue(ctx, fullTextKey, fullText)
		log.Debugf("Tạo fullText mới")
	}

	var onStartFunc func(...any)
	var onEndFunc func(err error, args ...any)

	if needSendTtsCmd {
		onStartFunc = func(...any) {
			// Xác định có phải lần gọi LLM đầu tiên không (qua giá trị nest trong context); chỉ lần đầu mới xóa cache audio TTS
			val := ctx.Value("nest")
			if nest, ok := val.(int); !ok || nest <= 1 {
				// Lần gọi đầu hoặc không có giá trị nest, xóa cache audio TTS
				l.ttsManager.ClearAudioHistory()
				log.Debugf("onStartFunc lần gọi đầu, đã xóa cache audio TTS")
			}
			l.ttsManager.EnqueueTtsStartWithReason(ctx, "LLMManager.handleLLMResponseChannelAsync onStart")
		}
		onEndFunc = func(err error, args ...any) {
			handleResult := llmHandleResultFromArgs(args)
			l.clientState.MarkLlmEnd()
			if l.session != nil {
				l.session.TraceLlmEnd(ctx, time.Now().UnixMilli(), err)
			}
			strFullText := fullText.String()

			l.finishTTSTurnWithReason(ctx, err, handleResult, "LLMManager.handleLLMResponseChannelAsync onEnd")

			// Lấy fullText từ closure
			audioData := l.ttsManager.GetAndClearAudioHistory()

			// Tính tổng kích thước audio (tổng byte của mọi frame)
			audioSize := 0
			for _, frame := range audioData {
				audioSize += len(frame)
			}

			// Chỉ gửi event khi là lần gọi đầu (nest<=1)
			if nest <= 1 {
				// Lấy MessageID từ LLMManager (role Assistant)
				// Nếu không tìm thấy MessageID, nghĩa là lưu giai đoạn 1 chưa hoàn tất, không cập nhật giai đoạn 2
				messageID, ok := l.GetLastMessageID(string(schema.Assistant))
				if !ok {
					log.Warnf("Không tìm thấy MessageID khi TTS hoàn tất, bỏ qua cập nhật audio giai đoạn 2")
					return
				}

				// Publish event: giai đoạn 2 (cập nhật audio)
				assistantMsg := schema.AssistantMessage(strFullText, nil)
				eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
					ClientState: l.clientState,
					Msg:         *assistantMsg,
					MessageID:   messageID,
					AudioData:   audioData, // Giai đoạn 2: có audio
					AudioSize:   audioSize,
					SampleRate:  l.clientState.OutputAudioFormat.SampleRate,
					Channels:    l.clientState.OutputAudioFormat.Channels,
					Timestamp:   time.Now(),
					IsUpdate:    true, // Cập nhật message
				})
			}
		}
	}

	onStartFunc = chainLLMResponseStartHooks(onStartFunc, options.onStartFunc)
	onEndFunc = chainLLMResponseEndHooks(onEndFunc, options.onEndFunc)

	item := LLMResponseChannelItem{
		ctx:          ctx,
		userMessage:  userMessage,
		responseChan: responseChan,
		onStartFunc:  onStartFunc,
		onEndFunc:    onEndFunc,
	}

	err := l.llmResponseQueue.Push(item)
	if err != nil {
		log.Warnf("llmResponseQueue đã đầy hoặc đã đóng, bỏ message")
		return fmt.Errorf("llmResponseQueue đã đầy hoặc đã đóng, bỏ message")
	}
	return nil
}

func (l *LLMManager) HandleLLMResponseChannelSync(ctx context.Context, userMessage *schema.Message, llmResponseChannel chan llm_common.LLMResponseStruct, einoTools []*schema.ToolInfo) (bool, error) {
	ctx = ensureTTSTurnTrackerInContext(ctx)

	needSendTtsCmd := true
	val := ctx.Value("nest")
	nest := 0
	log.Debugf("AddLLMResponseChannel nest: %+v", val)
	if n, ok := val.(int); ok {
		nest = n
		if nest > 1 {
			needSendTtsCmd = false
		}
	}

	// Khởi tạo hoặc tái dùng fullText trong context (dùng cho lịch sử chat)
	// Nếu context đã có fullText (tiếp tục request LLM sau tool call) thì tái dùng; nếu không thì tạo mới
	var fullText *strings.Builder
	if existingFullText, ok := ctx.Value(fullTextKey).(*strings.Builder); ok && existingFullText != nil {
		fullText = existingFullText
		log.Debugf("Tái dùng fullText hiện có, độ dài hiện tại: %d", fullText.Len())
	} else {
		fullText = &strings.Builder{}
		ctx = context.WithValue(ctx, fullTextKey, fullText)
		log.Debugf("Tạo fullText mới")
	}

	if needSendTtsCmd {
		// Xác định có phải lần gọi LLM đầu tiên không (qua giá trị nest trong context); chỉ lần đầu mới xóa cache audio TTS
		if nest <= 1 {
			// Lần gọi đầu hoặc không có giá trị nest, xóa cache audio TTS
			l.ttsManager.ClearAudioHistory()
			log.Debugf("HandleLLMResponseChannelSync lần gọi đầu, đã xóa cache audio TTS")
		}
		l.ttsManager.EnqueueTtsStartWithReason(ctx, "LLMManager.HandleLLMResponseChannelSync start")
	}

	result, err := l.handleLLMResponse(ctx, userMessage, llmResponseChannel)
	if waitErr := waitForTTSTurnDrainIfRoot(ctx); err == nil && waitErr != nil {
		err = waitErr
	}
	l.clientState.MarkLlmEnd()
	if l.session != nil {
		l.session.TraceLlmEnd(ctx, time.Now().UnixMilli(), err)
	}
	strFullText := fullText.String()

	if needSendTtsCmd {
		l.finishTTSTurnWithReason(ctx, err, result, "LLMManager.HandleLLMResponseChannelSync end")

		// Thu thập audio TTS và gửi event lịch sử chat
		// Lưu ý: response LLM sau tool call (nest > 1) cũng tích lũy audio vào cache nhưng không xóa cache
		// Chỉ xóa cache và gửi event khi là lần gọi đầu (nest<=1)
		audioData := l.ttsManager.GetAndClearAudioHistory()

		// Tính tổng kích thước audio (tổng byte của mọi frame)
		audioSize := 0
		for _, frame := range audioData {
			audioSize += len(frame)
		}

		// Chỉ gửi event khi là lần gọi đầu (nest<=1)
		if nest <= 1 {
			// Lấy MessageID từ LLMManager (role Assistant)
			// Nếu không tìm thấy MessageID, nghĩa là lưu giai đoạn 1 chưa hoàn tất, không cập nhật giai đoạn 2
			messageID, ok := l.GetLastMessageID(string(schema.Assistant))
			if !ok {
				log.Warnf("Không tìm thấy MessageID khi TTS hoàn tất, bỏ qua cập nhật audio giai đoạn 2")
				return result.ok, err
			}

			// Publish event: giai đoạn 2 (cập nhật audio)
			assistantMsg := schema.AssistantMessage(strFullText, nil)
			eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
				ClientState: l.clientState,
				Msg:         *assistantMsg,
				MessageID:   messageID,
				AudioData:   audioData, // Giai đoạn 2: có audio
				AudioSize:   audioSize,
				SampleRate:  l.clientState.OutputAudioFormat.SampleRate,
				Channels:    l.clientState.OutputAudioFormat.Channels,
				Timestamp:   time.Now(),
			})
		}
	} else {
		// Trường hợp nest > 1: dù không gửi lệnh TTS, dữ liệu audio vẫn tích lũy vào cache
		// Các audio này sẽ được thu thập cùng lúc khi response đầu kết thúc (nest <= 1)
		log.Debugf("Response LLM sau tool call (nest=%d), dữ liệu audio sẽ tích lũy vào cache", nest)
	}

	return result.ok, err
}

// handleLLMResponse xử lý response LLM
func (l *LLMManager) handleLLMResponse(ctx context.Context, userMessage *schema.Message, llmResponseChannel chan llm_common.LLMResponseStruct) (llmHandleResult, error) {
	log.Debugf("handleLLMResponse start")
	defer log.Debugf("handleLLMResponse end")

	// Lấy fullText từ context (dùng cho lịch sử chat)
	fullText := ctx.Value(fullTextKey).(*strings.Builder)
	state := l.clientState
	// toolCalls dùng biến local (logic gọi tool nội bộ, không liên quan lịch sử chat)
	var toolCalls []schema.ToolCall
	toolExecCtx := context.WithValue(ctx, "nest", 2)
	toolExecCtx = context.WithValue(toolExecCtx, fullTextKey, fullText)
	if speechStartHook := ttsPlaybackStartHookFromContext(ctx); speechStartHook != nil {
		toolExecCtx = withTTSPlaybackStartHook(toolExecCtx, speechStartHook)
	}
	if l.clientState.GetMemoryMode() == MemoryModeNone && userMessage != nil {
		toolExecCtx = appendToolRoundMessagesToContext(toolExecCtx, []*schema.Message{userMessage})
	}
	ttsTracker := ttsTurnTrackerFromContext(ctx)
	var onTTSItemEnqueued func() func(error)
	onTTSPlaybackStart := ttsPlaybackStartHookFromContext(ctx)
	if ttsTracker != nil {
		onTTSItemEnqueued = ttsTracker.Add
	}
	toolExecutor := newToolCallExecutor(l, toolExecCtx)
	assistantSaved := false
	result := llmHandleResult{}

	saveInterruptedAssistant := func() {
		if assistantSaved {
			return
		}
		if ctx.Err() == nil {
			return
		}
		text := strings.TrimSpace(fullText.String())
		if text == "" {
			return
		}
		msg := schema.AssistantMessage(text, nil)
		msg.Extra = map[string]any{
			interruptExtraKey:      true,
			interruptByExtraKey:    "user",
			interruptStageExtraKey: "llm",
		}
		if err := l.AddLlmMessage(ctx, msg); err != nil {
			log.Errorf("Lưu message assistant bị ngắt thất bại: %v", err)
			return
		}
		assistantSaved = true
	}

	select {
	case <-ctx.Done():
		saveInterruptedAssistant()
		log.Debugf("handleLLMResponse ctx done, return")
		return result, nil
	default:
	}

	for {
		select {
		case <-ctx.Done():
			// Context đã cancel, ưu tiên xử lý logic cancel.
			saveInterruptedAssistant()
			log.Infof("%s context đã cancel, dừng xử lý response LLM, context done, exit", state.DeviceID)
			return result, nil
		default:
			// Kiểm tra non-blocking; nếu ctx chưa Done thì tiếp tục xử lý response LLM.
			select {
			case llmResponse, ok := <-llmResponseChannel:
				if !ok {
					// Channel đã đóng, thoát goroutine
					log.Infof("Channel response LLM đã đóng, thoát goroutine")
					result.ok = true
					return result, nil
				}
				if ctx.Err() != nil {
					saveInterruptedAssistant()
					log.Infof("%s Context đã cancel khi fragment LLM tới, bỏ response đến muộn và thoát", state.DeviceID)
					return result, nil
				}

				log.Debugf("Response LLM: %+v", llmResponse)

				if len(llmResponse.ToolCalls) > 0 {
					log.Debugf("Lấy được tool: %+v", llmResponse.ToolCalls)
					toolCalls = append(toolCalls, llmResponse.ToolCalls...)
					toolExecutor.Submit(llmResponse.ToolCalls)
				}

				hasText := strings.TrimSpace(llmResponse.Text) != ""
				if hasText || llmResponse.IsStart || llmResponse.IsEnd {
					// Kết thúc dual-stream phụ thuộc tín hiệu IsEnd của text rỗng, không thể chỉ truyền cho TTS khi có text.
					if err := l.ttsManager.handleTextResponseWithHooks(ctx, llmResponse, false, onTTSItemEnqueued, onTTSPlaybackStart); err != nil {
						result.ok = true
						return result, err
					}
				}
				if hasText {
					fullText.WriteString(llmResponse.Text)
				}

				if llmResponse.IsEnd {
					if len(toolCalls) == 0 {
						//Ghi vào Redis
						if userMessage != nil {
							if userMessage.Role == schema.User {
								// Kiểm tra user message đã được lưu chưa (đã lưu khi xử lý ASR)
								// Xác định bằng cách kiểm tra message cuối có phải user message và content có khớp không
								/*messages := l.clientState.GetMessages(1)
								shouldSave := true
								if len(messages) > 0 {
									lastMsg := messages[len(messages)-1]
									if lastMsg.Role == schema.User && lastMsg.Content == userMessage.Content {
										// User message đã được lưu (khi xử lý ASR), bỏ qua
										shouldSave = false
										log.Debugf("User message đã được lưu khi xử lý ASR, bỏ qua lưu lặp: %s", userMessage.Content)
									}
								}
								if shouldSave {
									if err := l.AddLlmMessage(ctx, userMessage); err != nil {
										log.Errorf("Lưu user message thất bại: %v", err)
									}
								}*/
							}
						}
						strFullText := fullText.String()
						if strings.TrimSpace(strFullText) != "" || len(toolCalls) > 0 {
							if err := l.AddLlmMessage(ctx, schema.AssistantMessage(strFullText, toolCalls)); err != nil {
								log.Errorf("Lưu assistant message thất bại: %v", err)
							} else {
								assistantSaved = true
							}
						}
					}
					if len(toolCalls) > 0 {
						toolSummary, err := l.handleToolCallResponse(toolExecCtx, schema.AssistantMessage(fullText.String(), toolCalls), toolCalls, toolExecutor)
						if err != nil {
							log.Errorf("Xử lý response tool call thất bại: %v", err)
							result.ok = true
							return result, fmt.Errorf("Xử lý response tool call thất bại: %v", err)
						}
						result.suppressProtocolTtsStop = toolSummary.hasMediaOutput
						if !toolSummary.invokeToolSuccess && strings.TrimSpace(llmResponse.Text) != "" {
							if err := l.ttsManager.handleTextResponseWithHooks(ctx, llmResponse, false, nil, onTTSPlaybackStart); err != nil {
								result.ok = true
								return result, err
							}
							fullText.WriteString(llmResponse.Text)
						}
					}

					result.ok = true
					return result, nil
				}
			case <-ctx.Done():
				// Context đã cancel, thoát goroutine.
				saveInterruptedAssistant()
				log.Infof("%s context đã cancel, dừng xử lý response LLM, context done, exit", state.DeviceID)
				return result, nil
			}
		}
	}
}

func (l *LLMManager) DoLLmRequest(ctx context.Context, userMessage *schema.Message, einoTools []*schema.ToolInfo, isSync bool, speakerResult *speaker.IdentifyResult) error {
	log.Debugf("Gửi request LLM kèm tool, seesionID: %s, requestEinoMessages: %+v", l.clientState.SessionID, userMessage)
	clientState := l.clientState

	l.einoTools = einoTools

	//Ghép message lịch sử và message hiện tại của người dùng
	requestMessages := l.GetMessages(ctx, userMessage, MaxMessageCount, speakerResult)

	if l.session != nil {
		payload, stop, hookErr := l.session.hookHub.EmitLLMInput(l.session.hookContext(ctx), chathooks.LLMInputData{
			UserMessage:     userMessage,
			RequestMessages: requestMessages,
			Tools:           einoTools,
		})
		if hookErr != nil {
			log.Warnf("LLM_INPUT hook thực thi thất bại: %v", hookErr)
		}
		userMessage = payload.UserMessage
		requestMessages = payload.RequestMessages
		einoTools = payload.Tools
		if stop {
			log.Infof("LLM_INPUT hook yêu cầu dừng flow hiện tại")
			return nil
		}
	}

	clientState.SetStartLlmTs()
	if l.session != nil {
		l.session.TraceLlmStart(ctx, time.Now().UnixMilli())
	}
	clientState.SetStatus(ClientStatusLLMStart)

	// Gọi method nội bộ xử lý response LLM, resource được quản lý bên trong method
	responseSentences, err := l.handleLLMWithContextAndTools(
		ctx,
		requestMessages,
		einoTools,
	)
	if err != nil {
		log.Errorf("Gửi request LLM kèm tool thất bại, seesionID: %s, error: %v", l.clientState.SessionID, err)
		return fmt.Errorf("Gửi request LLM kèm tool thất bại: %v", err)
	}

	log.Debugf("DoLLmRequest goroutine bắt đầu - SessionID: %s, trạng thái context: %v", l.clientState.SessionID, ctx.Err())

	if isSync {
		// Xử lý đồng bộ: resource sẽ tự release trong defer của handleLLMWithContextAndTools
		_, err := l.HandleLLMResponseChannelSync(ctx, userMessage, responseSentences, einoTools)
		if err != nil {
			log.Errorf("Xử lý response LLM thất bại, seesionID: %s, error: %v", l.clientState.SessionID, err)
			return err
		}
	} else {
		// Xử lý bất đồng bộ: resource sẽ tự release trong defer của handleLLMWithContextAndTools
		err = l.HandleLLMResponseChannelAsync(ctx, userMessage, responseSentences)
		if err != nil {
			log.Errorf("Xử lý response LLM thất bại, seesionID: %s, error: %v", l.clientState.SessionID, err)
		}
	}

	log.Debugf("DoLLmRequest kết thúc - SessionID: %s", l.clientState.SessionID)

	return nil
}

// AddMessage thêm message vào lịch sử chat (entry thống nhất, áp dụng cho mọi loại message)
func (l *LLMManager) AddMessage(ctx context.Context, msg *schema.Message) error {
	if msg == nil {
		log.Warnf("Thử thêm nil message vào lịch sử chat")
		return fmt.Errorf("message không được nil")
	}

	// Tạo MessageID (dùng MD5 hash để rút ngắn, tránh vượt giới hạn varchar(64) trong DB)
	// Format gốc：{SessionID}-{Role}-{Timestamp}
	rawMessageID := fmt.Sprintf("%s-%s-%d",
		l.clientState.SessionID,
		msg.Role,
		time.Now().UnixMilli())
	// Dùng MD5 hash tạo chuỗi hex cố định 32 ký tự
	hash := md5.Sum([]byte(rawMessageID))
	messageID := hex.EncodeToString(hash[:])

	// Thêm đồng bộ vào memory
	l.clientState.AddMessage(msg)

	// Message role Tool: lưu trực tiếp, không liên quan lưu hai giai đoạn (không audio)
	if msg.Role == schema.Tool {
		eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
			ClientState: l.clientState,
			Msg:         *msg,
			MessageID:   messageID,
			AudioData:   nil, // Tool role không có audio
			AudioSize:   0,
			SampleRate:  0,
			Channels:    0,
			Timestamp:   time.Now(),
			IsUpdate:    false, // Lưu một lần
		})
		return nil
	}

	// Role User/Assistant: lưu hai giai đoạn
	// Lưu MessageID vào LLMManager để dùng cập nhật audio sau đó
	if msg.Role == schema.User || msg.Role == schema.Assistant {
		l.lastMessageIDMu.Lock()
		l.lastMessageID[string(msg.Role)] = messageID
		l.lastMessageIDMu.Unlock()
	}

	// Publish event: giai đoạn 1 (chỉ text, không audio)
	eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
		ClientState: l.clientState,
		Msg:         *msg,
		MessageID:   messageID,
		AudioData:   nil, // Giai đoạn 1: không audio
		AudioSize:   0,
		SampleRate:  0,
		Channels:    0,
		Timestamp:   time.Now(),
		IsUpdate:    false, // Thêm message mới
	})

	return nil
}

// AddLlmMessage giữ tương thích ngược, delegate sang AddMessage
func (l *LLMManager) AddLlmMessage(ctx context.Context, msg *schema.Message) error {
	return l.AddMessage(ctx, msg)
}

func (l *LLMManager) GetMessages(ctx context.Context, userMessage *schema.Message, count int, speakerResult *speaker.IdentifyResult) []*schema.Message {
	memoryMode := l.clientState.GetMemoryMode()
	includeHistory := memoryMode != MemoryModeNone

	// Lấy context từ dialogue; ở mode none chỉ cho phép mang message tạm của chuỗi tool call hiện tại
	messageList := make([]*schema.Message, 0)
	if includeHistory {
		messageList = l.clientState.GetMessages(count)
		if userMessage != nil {
			messageList = trimTrailingUserMessages(messageList)
		}
	} else if toolRoundMessages := toolRoundMessagesFromContext(ctx); len(toolRoundMessages) > 0 {
		messageList = toolRoundMessages
	}

	// Tạo system prompt
	systemPrompt := l.clientState.SystemPrompt
	globalSystemPrompt := strings.TrimSpace(viper.GetString("chat.global_system_prompt"))
	if globalSystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt = globalSystemPrompt + "\n\n" + systemPrompt
		} else {
			systemPrompt = globalSystemPrompt
		}
	}

	// Thêm thông tin ngày giờ hiện tại
	now := time.Now()
	systemPrompt += fmt.Sprintf("\nNgày giờ hiện tại: %s %s", now.Format("2006-01-02 15:04:05"), now.Format("Monday"))

	if memoryMode == MemoryModeLong && l.clientState.MemoryContext != "" {
		systemPrompt += fmt.Sprintf("\nThông tin cá nhân hóa người dùng: \n%s", l.clientState.MemoryContext)
	}

	log.Debugf("speakerResult: %+v, voiceIdentify: %+v", speakerResult, l.clientState.DeviceConfig.VoiceIdentify)

	// Tích hợp kết quả nhận diện người nói vào systemPrompt
	if speakerResult != nil && speakerResult.Identified {
		// Dựa vào speakerResult để match thông tin speakerGroup trong userConfig
		if l.clientState.DeviceConfig.VoiceIdentify != nil {
			// Ưu tiên dùng SpeakerName để match (key của VoiceIdentify là speakerGroup.Name)
			if speakerGroupInfo, found := l.clientState.DeviceConfig.VoiceIdentify[speakerResult.SpeakerName]; found {
				// Nếu tìm thấy speakerGroup khớp, tích hợp mô tả vào systemPrompt
				if speakerGroupInfo.Prompt != "" {
					systemPrompt += fmt.Sprintf("\nThông tin người đối thoại nhận diện qua voiceprint: \n%s", speakerGroupInfo.Prompt)
				}
			}
		}
	}

	//search memory
	if memoryMode == MemoryModeLong && l.clientState.MemoryProvider != nil && userMessage != nil {
		memoryContext, err := l.clientState.MemoryProvider.Search(ctx, l.clientState.GetDeviceIDOrAgentID(), userMessage.Content, 10, 180)
		if err != nil {
			log.Errorf("Tìm memory thất bại: %v", err)
		}
		log.Debugf("Tìm memory thành công, input: %s, nội dung memory: %s", userMessage.Content, memoryContext)
		if memoryContext != "" {
			systemPrompt += fmt.Sprintf("\nThông tin liên quan trong lịch sử: \n%s", memoryContext)
		}
	}

	systemPrompt += buildKnowledgeSearchRoutingPolicy(l.clientState.DeviceConfig.KnowledgeBases)

	retMessage := make([]*schema.Message, 0)
	retMessage = append(retMessage, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	})
	// Lọc message assistant rỗng, tránh lỗi 400 khi gửi tới LLM API
	// Message assistant rỗng (Content rỗng và ToolCalls rỗng) sẽ gây lỗi API
	for _, msg := range messageList {
		if msg != nil && msg.Role == schema.Assistant && msg.Content == "" && len(msg.ToolCalls) == 0 {
			log.Debugf("Lọc message assistant rỗng, tránh gửi tới LLM API")
			continue
		}
		msgCopy := cloneMessageForRequest(msg)
		if isInterruptedMessage(msgCopy) {
			msgCopy.Content = decorateInterruptedContent(msgCopy.Content)
		}
		retMessage = append(retMessage, msgCopy)
	}
	if userMessage != nil {
		// Kiểm tra message cuối của retMessage có phải cùng user message không để tránh thêm lặp
		shouldAdd := true
		if len(retMessage) > 0 {
			lastMsg := retMessage[len(retMessage)-1]
			if lastMsg.Role == schema.User && lastMsg.Content == userMessage.Content {
				// Message cuối đã là cùng user message, bỏ qua thêm
				shouldAdd = false
				//log.Debugf("Message cuối đã là cùng user message, bỏ qua thêm lặp: %s", userMessage.Content)
			}
		}
		if shouldAdd {
			retMessage = append(retMessage, userMessage)
		}
	}
	return retMessage
}

func buildKnowledgeSearchRoutingPolicy(knowledgeBases []config_types.KnowledgeBaseRef) string {
	if len(knowledgeBases) == 0 {
		return ""
	}

	availableKBs := make([]string, 0, len(knowledgeBases))
	for _, kb := range knowledgeBases {
		if strings.EqualFold(strings.TrimSpace(kb.Status), "inactive") {
			continue
		}
		if strings.TrimSpace(kb.ExternalKBID) == "" {
			continue
		}
		name := strings.TrimSpace(kb.Name)
		if name == "" {
			name = strings.TrimSpace(kb.ExternalKBID)
		}
		if name == "" {
			continue
		}
		if kb.ID == 0 {
			continue
		}
		desc := strings.TrimSpace(kb.Description)
		if desc == "" {
			desc = "Không có mô tả"
		}
		availableKBs = append(availableKBs, fmt.Sprintf("%d: Tên=%s; Mô tả=%s", kb.ID, name, desc))
		if len(availableKBs) >= 8 {
			break
		}
	}
	if len(availableKBs) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"\nQuy tắc truy xuất knowledge base (tool: search_knowledge):\nKnowledge base khả dụng (id:Tên+Mô tả): %s\n"+
			"1. Điều kiện trigger: người dùng hỏi về sự thật, quy trình, tham số, quy tắc, định nghĩa, điều khoản, so sánh... cần căn cứ tài liệu, hoặc yêu cầu rõ “trả lời theo knowledge base/tài liệu”.\n"+
			"2. Điều kiện không trigger: chào hỏi trò chuyện, đồng hành cảm xúc, sáng tạo thuần túy, gợi ý thuần chủ quan.\n"+
			"3. Cách gọi: mỗi lượt tối đa 1 lần, query rút gọn keyword cốt lõi từ câu hỏi người dùng, top_k mặc định 5; nếu xác định được knowledge base cụ thể, truyền knowledge_base_ids (có thể nhiều).\n"+
			"4. Quy tắc chọn: chỉ truyền knowledge base ID liên quan ngữ nghĩa nhất với câu hỏi hiện tại; nếu không xác định được thì có thể không truyền knowledge_base_ids.\n"+
			"5. Xử lý thiếu thông tin: nếu bằng chứng không đủ, không bịa, hãy yêu cầu người dùng bổ sung keyword cụ thể hơn.\n"+
			"6. Yêu cầu output: khi trả lời không nhắc tới các thông tin nguồn/quy trình như “knowledge base”, “truy xuất”, “MCP”, “tool call”, “kết quả hit”.",
		strings.Join(availableKBs, "、"),
	)
}

func trimTrailingUserMessages(messages []*schema.Message) []*schema.Message {
	end := len(messages)
	for end > 0 {
		msg := messages[end-1]
		if msg == nil || msg.Role != schema.User {
			break
		}
		end--
	}
	return messages[:end]
}

func isInterruptedMessage(msg *schema.Message) bool {
	if msg == nil || msg.Extra == nil {
		return false
	}
	v, ok := msg.Extra[interruptExtraKey]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func decorateInterruptedContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	if strings.HasSuffix(content, interruptContentSuffix) {
		return content
	}
	return content + interruptContentSuffix
}

func cloneMessagesForRequest(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}

	cloned := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		cloned = append(cloned, cloneMessageForRequest(msg))
	}

	return cloned
}

func toolRoundMessagesFromContext(ctx context.Context) []*schema.Message {
	if ctx == nil {
		return nil
	}

	messages, ok := ctx.Value(toolRoundMessagesKey).([]*schema.Message)
	if !ok || len(messages) == 0 {
		return nil
	}

	return cloneMessagesForRequest(messages)
}

func ttsTurnTrackerFromContext(ctx context.Context) *ttsTurnTracker {
	if ctx == nil {
		return nil
	}

	tracker, ok := ctx.Value(ttsTurnTrackerKey).(*ttsTurnTracker)
	if !ok {
		return nil
	}

	return tracker
}

func ensureTTSTurnTrackerInContext(ctx context.Context) context.Context {
	if ttsTurnTrackerFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnTrackerKey, newTTSTurnTracker())
}

func waitForTTSTurnDrainIfRoot(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if nest, ok := ctx.Value("nest").(int); ok && nest > 1 {
		return nil
	}

	tracker := ttsTurnTrackerFromContext(ctx)
	if tracker == nil {
		return nil
	}

	return tracker.Wait(ctx)
}

func appendToolRoundMessagesToContext(ctx context.Context, messages []*schema.Message) context.Context {
	if len(messages) == 0 {
		return ctx
	}

	combined := toolRoundMessagesFromContext(ctx)
	combined = append(combined, cloneMessagesForRequest(messages)...)
	if len(combined) == 0 {
		return ctx
	}

	return context.WithValue(ctx, toolRoundMessagesKey, combined)
}

func cloneMessageForRequest(msg *schema.Message) *schema.Message {
	if msg == nil {
		return nil
	}
	msgCopy := *msg

	if msg.ToolCalls != nil {
		msgCopy.ToolCalls = append([]schema.ToolCall(nil), msg.ToolCalls...)
	}
	if msg.MultiContent != nil {
		msgCopy.MultiContent = append([]schema.ChatMessagePart(nil), msg.MultiContent...)
	}
	if msg.Extra != nil {
		msgCopy.Extra = make(map[string]any, len(msg.Extra))
		for k, v := range msg.Extra {
			msgCopy.Extra[k] = v
		}
	}
	if msg.ResponseMeta != nil {
		respMetaCopy := *msg.ResponseMeta
		msgCopy.ResponseMeta = &respMetaCopy
	}

	return &msgCopy
}
