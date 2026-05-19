package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"sync"

	utypes "xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/memory"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/domain/tts"

	. "xiaozhi-esp32-server-golang/internal/data/audio"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

// Dialogue biểu diễn lịch sử hội thoại
type Dialogue struct {
	mu       sync.RWMutex // khóa bảo vệ đọc/ghi Messages
	Messages []*schema.Message
}

const (
	ClientStatusInit       = "init"
	ClientStatusListening  = "listening"
	ClientStatusListenStop = "listenStop"
	ClientStatusLLMStart   = "llmStart"
	ClientStatusTTSStart   = "ttsStart"

	ListenPhaseIdle      = "idle"
	ListenPhaseStarting  = "starting"
	ListenPhaseListening = "listening"

	CommandTypeDetect      = "detect"
	CommandTypeListenStart = "listen_start"
	CommandTypeListenStop  = "listen_stop"

	MemoryModeNone  = "none"
	MemoryModeShort = "short"
	MemoryModeLong  = "long"

	SpeakerChatModeOff            = "off"
	SpeakerChatModeIdentifiedOnly = "identified_only"
)

func NormalizeMemoryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case MemoryModeNone:
		return MemoryModeNone
	case MemoryModeLong:
		return MemoryModeLong
	default:
		return MemoryModeShort
	}
}

func NormalizeSpeakerChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case SpeakerChatModeIdentifiedOnly:
		return SpeakerChatModeIdentifiedOnly
	default:
		return SpeakerChatModeOff
	}
}

type SendAudioData func(audioData []byte) error

// ClientState biểu diễn trạng thái client
type ClientState struct {
	cmdMu sync.Mutex

	IsActivated bool
	// Lịch sử hội thoại
	Dialogue *Dialogue
	// Trạng thái interrupt
	Abort bool
	// Mode thu âm
	ListenMode string
	// listen start trạng thái flow: idle / starting / listening
	ListenPhase string
	// Device ID
	DeviceID string
	AgentID  string
	// Session ID
	SessionID string

	//Config thiết bị
	DeviceConfig utypes.UConfig

	Vad
	Asr
	Llm

	// TTS provider
	TTSProvider      tts.TTSProvider        // TTS provider mặc định
	SpeakerTTSConfig map[string]interface{} // Config TTS của nhận diện voiceprint (config đầy đủ, ưu tiên dùng)
	// Memory provider
	MemoryProvider memory.MemoryProvider
	MemoryContext  string //memory context

	// Điều khiển context
	Ctx    context.Context
	Cancel context.CancelFunc

	SessionCtx         Ctx //context của một lượt hội thoại
	AfterAsrSessionCtx Ctx //context flow sau ASR

	//prompt, system prompt
	SystemPrompt string

	InputAudioFormat  AudioFormat //format audio input
	OutputAudioFormat AudioFormat //format audio output

	// buffer dữ liệu audio nhận Opus
	OpusAudioBuffer chan []byte

	// buffer dữ liệu audio nhận PCM
	AsrAudioBuffer *AsrAudioBuffer

	VoiceStatus
	AudioIdle AudioIdleClock

	UdpSendAudioData SendAudioData //gửi dữ liệu audio
	Statistic        Statistic     //thống kê thời gian
	MqttLastActiveTs int64         //thời gian active cuối
	VadLastActiveTs  int64         // thời gian active cuối của VAD, quá 60s && không ở TTS thì ngắt kết nối

	Status string //trạng thái listening, llmStart, ttsStart

	IsTtsStart        bool //TTS đã bắt đầu chưa
	IsWelcomeSpeaking bool //đã phát lời chào chưa
	IsWelcomePlaying  bool //đang phát lời chào chưa

	LastCmdType string
	LastCmdAt   time.Time

	// Liên quan nhận diện voiceprint
	SpeakerProvider speaker.SpeakerProvider // provider nhận diện voiceprint (khởi tạo trong session)

	// callback lấy kết quả voiceprint bất đồng bộ (set trong session)
	OnVoiceSilenceSpeakerCallback func(ctx context.Context)

	// callback metric event voice silence (set trong session)
	OnVoiceSilenceMetricCallback func(ctx context.Context, ts int64)

	// callback ký tự đầu ASR trả về (set trong session)
	OnAsrFirstTextCallback func(text string, isFinal bool)
}

// IsSpeakerEnabled kiểm tra có bật nhận diện voiceprint không (đọc từ global config)
func (c *ClientState) IsSpeakerEnabled() bool {
	// Lấy field enable từ global config (viper)
	enabled := viper.GetBool("voice_identify.enable")
	return enabled
}

// HasSpeakerGroups kiểm tra config thiết bị có voiceprint group không.
func (c *ClientState) HasSpeakerGroups() bool {
	// Kiểm tra config thiết bị có config voiceprint group không.
	return len(c.DeviceConfig.VoiceIdentify) > 0
}

func (c *ClientState) IsRealTime() bool {
	return c.ListenMode == "realtime"
}

func (c *ClientState) GetMemoryMode() string {
	return NormalizeMemoryMode(c.DeviceConfig.MemoryMode)
}

func (c *ClientState) GetSpeakerChatMode() string {
	return NormalizeSpeakerChatMode(c.DeviceConfig.SpeakerChatMode)
}

func (c *ClientState) RequireMatchedSpeakerForChat() bool {
	return c.HasSpeakerGroups() && c.GetSpeakerChatMode() == SpeakerChatModeIdentifiedOnly
}

func (c *ClientState) HasMatchedConfiguredSpeaker(result *speaker.IdentifyResult) bool {
	if result == nil || !result.Identified {
		return false
	}
	_, ok := c.DeviceConfig.VoiceIdentify[result.SpeakerName]
	return ok
}

func (c *ClientState) GetDeviceIDOrAgentID() string {
	if c.AgentID != "" {
		return c.AgentID
	}
	return c.DeviceID
}

// Bắt đầu các method liên quan lịch sử message
func (c *ClientState) AddMessage(msg *schema.Message) {
	if msg == nil {
		log.Warnf("Thử thêm nil message vào lịch sử hội thoại")
		return
	}
	c.Dialogue.mu.Lock()
	defer c.Dialogue.mu.Unlock()
	c.Dialogue.Messages = append(c.Dialogue.Messages, msg)
}

func (c *ClientState) GetMessages(count int) []*schema.Message {
	c.Dialogue.mu.RLock()
	defer c.Dialogue.mu.RUnlock()

	// Thêm kiểm tra biên, tránh vượt mảng
	if len(c.Dialogue.Messages) == 0 {
		return []*schema.Message{}
	}

	// Tính index bắt đầu, đảm bảo không vượt biên
	startIndex := len(c.Dialogue.Messages) - count
	if startIndex < 0 {
		startIndex = 0
	}

	return AlignToolMessages(c.Dialogue.Messages[startIndex:])
}

/*
func AlignMessage(messages []*schema.Message) []*schema.Message {
	findMsgTypeUser := false
	// Để đảm bảo tính toàn vẹn message, duyệt để tìm message sau User đầu tiên
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		if msg == nil {
			continue
		}
		if !findMsgTypeUser {
			if msg.Role == schema.User {
				return messages[i:]
			}
			continue
		}
	}
	return messages
}
*/
// AlignToolMessages đảm bảo tool_call_id trong message role:tool khớp với id của tool_calls trong message role:assistant
// Nếu không khớp thì xóa message tool tương ứng, đồng thời xử lý trường hợp không khớp ngược
func AlignToolMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}

	// Thu thập mọi tool_calls id trong message assistant
	validToolCallIDs := make(map[string]bool)
	// Thu thập mọi tool_call_id trong message tool
	usedToolCallIDs := make(map[string]bool)

	// Lượt duyệt đầu: thu thập tool_calls id trong message assistant và tool_call_id trong message tool
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.ID != "" {
					validToolCallIDs[toolCall.ID] = true
				}
			}
		}

		if msg.Role == schema.Tool && msg.ToolCallID != "" {
			usedToolCallIDs[msg.ToolCallID] = true
		}
	}

	// Lọc message, xử lý trường hợp không khớp hai chiều
	var alignedMessages []*schema.Message
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		// Nếu là message tool, kiểm tra tool_call_id có hợp lệ không
		if msg.Role == schema.Tool {
			if msg.ToolCallID != "" && validToolCallIDs[msg.ToolCallID] {
				alignedMessages = append(alignedMessages, msg)
			}
		} else if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			// Xử lý message assistant, kiểm tra có tool_calls chưa dùng không
			for _, toolCall := range msg.ToolCalls {
				if toolCall.ID != "" {
					if usedToolCallIDs[toolCall.ID] {
						alignedMessages = append(alignedMessages, msg)
					} else {
						continue
					}
				}
			}
		} else {
			// Các loại message khác giữ nguyên
			alignedMessages = append(alignedMessages, msg)
		}
	}

	return alignedMessages
}

func (c *ClientState) InitMessages(messages []*schema.Message) error {
	c.Dialogue.mu.Lock()
	defer c.Dialogue.mu.Unlock()
	c.Dialogue.Messages = AlignToolMessages(messages)
	return nil
}

//Kết thúc các method liên quan lịch sử message

func (c *ClientState) SetTtsStart(isStart bool) {
	c.IsTtsStart = isStart
}

func (c *ClientState) GetTtsStart() bool {
	return c.IsTtsStart
}

func (c *ClientState) GetMaxIdleDuration() int64 {
	if !viper.IsSet("chat.max_idle_duration") {
		return 30000
	}

	maxIdleDuration := viper.GetInt64("chat.max_idle_duration")
	if maxIdleDuration <= 0 {
		return math.MaxInt64
	}
	return maxIdleDuration
}

func (c *ClientState) UsesAudioIdleClock() bool {
	if c == nil {
		return false
	}
	return c.ListenMode == "auto" || c.IsRealTime()
}

func (c *ClientState) ShouldCountAudioIdleTimeout() bool {
	if c == nil || !c.IsRealTime() {
		return true
	}
	if c.GetTtsStart() {
		return false
	}
	switch c.GetStatus() {
	case ClientStatusLLMStart, ClientStatusTTSStart:
		return false
	default:
		return true
	}
}

func (c *ClientState) StartAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Start(now)
	c.SetClientVoiceStop(false)
}

func (c *ClientState) PauseAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Pause(now)
}

func (c *ClientState) ResumeAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Resume(now)
	c.SetClientVoiceStop(false)
}

func (c *ClientState) ResetAudioIdleWindow() {
	if c == nil {
		return
	}
	c.AudioIdle.Reset()
}

func (c *ClientState) GetAudioIdleElapsed(now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	return c.AudioIdle.Elapsed(now)
}

func (c *ClientState) AudioIdleStarted() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.Started()
}

func (c *ClientState) AudioIdlePaused() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.Paused()
}

func (c *ClientState) MarkAudioIdleTimeoutPending() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.MarkTimeoutPending()
}

func (c *ClientState) ClearAudioIdleTimeoutPending() {
	if c == nil {
		return
	}
	c.AudioIdle.ClearTimeoutPending()
}

func (c *ClientState) AudioIdleTimeoutPending() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.TimeoutPending()
}

func (c *ClientState) GetPreAsrTextSilenceDuration() int64 {
	if viper.IsSet("chat.pre_asr_text_silence_duration") {
		preTextSilenceDuration := viper.GetInt64("chat.pre_asr_text_silence_duration")
		if preTextSilenceDuration <= 0 {
			return math.MaxInt64
		}
		return preTextSilenceDuration
	}

	base := c.VoiceStatus.SilenceThresholdTime
	if base <= 0 {
		base = 400
	}
	preTextSilenceDuration := base * 4
	if preTextSilenceDuration < 1000 {
		preTextSilenceDuration = 1000
	}
	return preTextSilenceDuration
}

func (c *ClientState) UpdateLastActiveTs() {
	c.MqttLastActiveTs = time.Now().Unix()
}

func (c *ClientState) IsActive() bool {
	diff := time.Now().Unix() - c.MqttLastActiveTs
	return c.MqttLastActiveTs > 0 && diff <= ClientActiveTs
}

func (c *ClientState) SetStatus(status string) {
	c.Status = status
}

func (c *ClientState) GetStatus() string {
	return c.Status
}

func (c *ClientState) SetListenPhase(phase string) {
	c.ListenPhase = phase
}

func (c *ClientState) GetListenPhase() string {
	return c.ListenPhase
}

type Ctx struct {
	sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Ctx) Reset() {
	c.ResetWithReason("Ctx.Reset")
}

func (c *Ctx) ResetWithReason(reason string) {
	c.Lock()
	defer c.Unlock()
	if c.ctx != nil {
		log.Debugf("Ctx.ResetWithReason: reason=%s", reason)
		c.cancel()
		c.ctx = nil
		c.cancel = nil
	}
}

func (c *Ctx) Get(parentCtx context.Context) context.Context {
	c.Lock()
	defer c.Unlock()
	if c.ctx == nil || c.ctx.Err() != nil {
		if c.ctx != nil {
			c.cancel()
		}
		c.ctx, c.cancel = context.WithCancel(parentCtx)
	}
	return c.ctx
}

func (c *Ctx) Cancel() {
	c.CancelWithReason("Ctx.Cancel")
}

func (c *Ctx) CancelWithReason(reason string) {
	c.Lock()
	defer c.Unlock()
	if c.ctx != nil {
		log.Debugf("Ctx.CancelWithReason: reason=%s", reason)
		c.cancel()
		c.ctx = nil
		c.cancel = nil
	}
}

func (s *ClientState) getLLMProvider() (llm.LLMProvider, error) {
	llmConfig := s.DeviceConfig.Llm
	providerName := llmConfig.Provider
	if providerName == "" {
		providerName = "openai"
	}
	llmProvider, err := llm.GetLLMProvider(providerName, llmConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("Tạo LLM provider thất bại: %v", err)
	}
	return llmProvider, nil
}

func (s *ClientState) InitLlm() error {
	ctx, cancel := context.WithCancel(s.Ctx)

	llmProvider, err := s.getLLMProvider()
	if err != nil {
		log.Errorf("Tạo LLM provider thất bại: %v", err)
		return err
	}

	s.Llm = Llm{
		Ctx:         ctx,
		Cancel:      cancel,
		LLMProvider: llmProvider,
	}
	return nil
}

func (s *ClientState) InitAsr() error {
	asrConfig := s.DeviceConfig.Asr

	log.Infof("Khởi tạo ASR, asrConfig: %+v", asrConfig)

	//Khởi tạo ASR (không tạo trực tiếp AsrProvider nữa, chuyển sang dùng resource pool)
	ctx, cancel := context.WithCancel(s.Ctx)
	s.Asr = Asr{
		Ctx:             ctx,
		Cancel:          cancel,
		AsrAudioChannel: make(chan []float32, 100),
		AsrEnd:          make(chan bool, 1),
		AsrResult:       bytes.Buffer{},
		AsrType:         asrConfig.Provider,
		ClientState:     s, // Set tham chiếu ClientState
	}

	// Set mode ASR
	if mode, ok := asrConfig.Config["mode"].(string); ok {
		s.Asr.Mode = mode
	}

	if rawAutoEnd, ok := asrConfig.Config["auto_end"]; ok {
		if autoEnd, ok := rawAutoEnd.(bool); ok {
			s.Asr.AutoEnd = autoEnd
		}
	}
	return nil
}

func (c *ClientState) Destroy() {
	c.Asr.StopWithReason("ClientState.Destroy")
	c.Vad.Reset()
	c.ResetAudioIdleWindow()
	c.ClearAudioIdleTimeoutPending()

	// Trả resource ASR (nếu có)
	// Lưu ý: ở đây cần import package pool, nhưng để tránh phụ thuộc vòng thì xử lý tại nơi gọi
	// Hoặc dùng type assertion tại đây, nhưng cần import package pool
	// Tạm thời xử lý trả resource tại nơi gọi (ChatSession.Close)

	c.VoiceStatus.Reset()
	c.AsrAudioBuffer.ClearAsrAudioData()

	c.SessionCtx.ResetWithReason("ClientState.Destroy: session_ctx")
	c.AfterAsrSessionCtx.ResetWithReason("ClientState.Destroy: after_asr_ctx")

	c.Statistic.Reset()
	c.SetStatus(ClientStatusInit)
	c.SetListenPhase(ListenPhaseIdle)
	c.SetTtsStart(false)
}

type CommandHistorySnapshot struct {
	LastCmdType string
	LastCmdAt   time.Time
}

func (s CommandHistorySnapshot) DebugString(now time.Time) string {
	formatAt := func(at time.Time) string {
		if at.IsZero() {
			return "zero"
		}
		return at.Format(time.RFC3339Nano)
	}
	formatAge := func(at time.Time) string {
		if at.IsZero() {
			return "n/a"
		}
		return now.Sub(at).Truncate(time.Millisecond).String()
	}

	return fmt.Sprintf(
		"lastCmd=%q lastCmdAt=%s lastCmdAge=%s",
		s.LastCmdType,
		formatAt(s.LastCmdAt),
		formatAge(s.LastCmdAt),
	)
}

func (c *ClientState) RecordCommandArrival(cmdType string, at time.Time) {
	c.cmdMu.Lock()
	c.LastCmdType = cmdType
	c.LastCmdAt = at
	c.cmdMu.Unlock()
}

func (c *ClientState) GetCommandHistorySnapshot() CommandHistorySnapshot {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	return CommandHistorySnapshot{
		LastCmdType: c.LastCmdType,
		LastCmdAt:   c.LastCmdAt,
	}
}

func (state *ClientState) OnManualStop() {
	state.ClearAudioIdleTimeoutPending()
	state.OnVoiceSilence()
}

func (state *ClientState) OnVoiceSilence() {
	silenceTs := time.Now().UnixMilli()
	log.Debugf("OnVoiceSilence, voiceDuration: %d, voiceDurationInSession: %d", state.Vad.GetVoiceDuration(), state.Vad.GetVoiceDurationInSession())
	if state.MarkVoiceSilenceAt(silenceTs) && state.OnVoiceSilenceMetricCallback != nil {
		state.OnVoiceSilenceMetricCallback(state.Ctx, silenceTs)
	}
	state.Asr.ResetReceivedText()
	state.SetClientVoiceStop(true) //set cờ dừng nói, lúc này dữ liệu audio nhận được sẽ không vào VAD
	//Client dừng nói
	state.Asr.StopWithReason("ClientState.OnVoiceSilence") //dừng ASR và lấy kết quả, chạy LLM
	//release VAD
	state.Vad.Reset() // release instance VAD

	state.SetStatus(ClientStatusListenStop)
	state.SetListenPhase(ListenPhaseIdle)

	// Nếu đã set callback lấy kết quả voiceprint bất đồng bộ thì gọi
	if state.OnVoiceSilenceSpeakerCallback != nil {
		state.OnVoiceSilenceSpeakerCallback(state.Ctx)
	}
}

type Llm struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	// LLM provider
	LLMProvider llm.LLMProvider
	//channel nhận ASR to text
	LLmRecvChannel chan llm_common.LLMResponseStruct
}

type SpeakReadyUDPConfig struct {
	Ready         bool `json:"ready"`
	ReuseExisting bool `json:"reuse_existing,omitempty"`
}

// ClientMessage biểu diễn message client
type ClientMessage struct {
	Type           string               `json:"type"`
	DeviceID       string               `json:"device_id,omitempty"`
	SessionID      string               `json:"session_id,omitempty"`
	Text           string               `json:"text,omitempty"`
	Mode           string               `json:"mode,omitempty"`
	State          string               `json:"state,omitempty"`
	Token          string               `json:"token,omitempty"`
	DeviceMac      string               `json:"device_mac,omitempty"`
	Version        int                  `json:"version,omitempty"`
	Transport      string               `json:"transport,omitempty"`
	Features       map[string]bool      `json:"features,omitempty"`
	AudioParams    *AudioFormat         `json:"audio_params,omitempty"`
	SpeakUDPConfig *SpeakReadyUDPConfig `json:"udp_config,omitempty"`
	PayLoad        json.RawMessage      `json:"payload,omitempty"`
}
