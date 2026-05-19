package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/schema"

	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"
)

var openClawWarmupSchedule = []time.Duration{
	1 * time.Second,
	10 * time.Second,
	20 * time.Second,
	30 * time.Second,
	40 * time.Second,
	50 * time.Second,
	60 * time.Second,
	70 * time.Second,
	80 * time.Second,
	90 * time.Second,
	100 * time.Second,
}

const (
	openClawWarmupPlanTimeout = 8 * time.Second
	openClawWarmupPlanSize    = 11
)

const openClawWarmupSystemPrompt = `Bạn là trợ lý mở lời trong hội thoại giọng nói realtime, không phải người trả lời chính.

Nhiệm vụ của bạn: trước khi câu trả lời chính quay về, tạo 11 câu nối rất ngắn bằng tiếng Việt để quá trình chờ nghe như luôn có người phản hồi.

Yêu cầu bắt buộc:
1. Chỉ phụ trách mở lời trong lúc chờ, không trả lời trực tiếp câu hỏi, không đưa sự thật, kết luận, đề xuất, bước làm, phân tích, giải thích hoặc suy đoán.
2. Giọng điệu giống người thật đang nói nhẹ trong cuộc gọi: ngắn, tự nhiên, khẩu ngữ, kiên nhẫn.
3. Không giống chăm sóc khách hàng, không giống thông báo hệ thống, không giống bản tin, không giống viết quảng cáo.
4. Cấm lặp lại nguyên văn lời người dùng, nhất là các chỉ thị như “giúp tôi tra”, “xem giúp tôi”, “tra giúp tôi”, “nói cho tôi”.
5. Nếu cần nhắc tới chủ đề, chỉ được rút gọn thành cụm danh từ theo góc nhìn trợ lý, ví dụ “thời tiết Hà Nội ngày kia”, “lịch này”; không dùng câu mệnh lệnh.
6. 1 đến 2 câu đầu nên thật nhẹ, không nhất thiết có từ khóa chủ đề, ví dụ “Tôi xem chút nhé”, “Chờ tôi một chút”; đừng mở đầu bằng câu trấn an quá nặng.
7. Các câu sau dần thể hiện “tôi vẫn đang xem”, “tôi vẫn đang kiểm tra”, nhưng phải tự nhiên, không lặp máy móc.
8. Tránh dùng các cách nói cứng như “đang xử lý cho bạn”, “vui lòng chờ”, “liên tục theo dõi”, “truy xuất dữ liệu”, “đang kết nối dịch vụ”.
9. Mỗi mục phải là một câu tiếng Việt ngắn, phù hợp đọc bằng giọng nói, dài khoảng 3 đến 12 từ.
10. Bạn sẽ nhận các mốc phát thực tế. 11 câu phải được thiết kế đúng thứ tự các mốc này:
   - Giây 1: như vừa nhận câu hỏi, nối nhẹ một câu.
   - Giây 10: bổ sung tự nhiên một câu, giọng vẫn nhẹ.
   - Giây 20, 30: bắt đầu thể hiện “tôi vẫn đang xem”, nhưng không máy móc.
   - Giây 40, 50, 60: tiếp tục trấn an, có thể nói rõ hơn là “vẫn đang kiểm tra”.
   - Giây 70, 80, 90, 100: thừa nhận hơi lâu nhưng vẫn tự nhiên, bình tĩnh, không than phiền.
11. Chỉ output JSON array nghiêm ngặt, độ dài phải là 11.
12. Mỗi item JSON có format: {"text":"câu mở lời"}.
13. Cấm output số thứ tự, Markdown, giải thích, code block hoặc bất kỳ nội dung nào ngoài JSON.`

type openClawWarmupTask struct {
	correlationID string
	sessionCtx    context.Context
	warmupCtx     context.Context
	cancelWarmup  context.CancelFunc

	linesMu sync.RWMutex
	lines   []string

	stateMu                  sync.Mutex
	speechStarted            bool
	speechEnded              bool
	nextWarmupSegmentIsStart bool
	planReadyAt              time.Time
	planReadySignaled        bool

	spokeAny    atomic.Bool
	planReadyCh chan struct{}
}

type openClawWarmupLine struct {
	Text string `json:"text"`
}

func (s *ChatSession) startOpenClawWarmup(correlationID string, userText string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" || s == nil || s.clientState == nil {
		return
	}

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	parentCtx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	warmupCtx, cancelWarmup := context.WithCancel(parentCtx)
	task := &openClawWarmupTask{
		correlationID:            correlationID,
		sessionCtx:               parentCtx,
		warmupCtx:                warmupCtx,
		cancelWarmup:             cancelWarmup,
		lines:                    make([]string, openClawWarmupPlanSize),
		nextWarmupSegmentIsStart: true,
		planReadyCh:              make(chan struct{}),
	}

	s.replaceOpenClawWarmup(task)
	log.Infof("OpenClaw warmup started: device=%s correlation_id=%s", s.clientState.DeviceID, correlationID)

	go s.runOpenClawWarmupTask(task, userText)
}

func (s *ChatSession) replaceOpenClawWarmup(task *openClawWarmupTask) {
	s.openClawWarmupMu.Lock()
	oldTask := s.openClawWarmup
	s.openClawWarmup = task
	s.openClawWarmupMu.Unlock()

	if oldTask != nil {
		oldTask.cancelWarmupOnly()
	}
}

func (task *openClawWarmupTask) cancelWarmupOnly() {
	if task == nil || task.cancelWarmup == nil {
		return
	}
	task.cancelWarmup()
}

func (task *openClawWarmupTask) markSpeechStarted() bool {
	if task == nil {
		return false
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if task.speechStarted || task.speechEnded {
		return false
	}
	task.speechStarted = true
	return true
}

func (task *openClawWarmupTask) markSpeechEnded() bool {
	if task == nil {
		return false
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if !task.speechStarted || task.speechEnded {
		return false
	}
	task.speechEnded = true
	return true
}

func (task *openClawWarmupTask) takeWarmupSegmentStartFlag() bool {
	if task == nil {
		return true
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	isStart := task.nextWarmupSegmentIsStart
	task.nextWarmupSegmentIsStart = false
	return isStart
}

func (task *openClawWarmupTask) markPlanReady(readyAt time.Time) {
	if task == nil {
		return
	}
	task.stateMu.Lock()
	if task.planReadySignaled {
		task.stateMu.Unlock()
		return
	}
	task.planReadyAt = readyAt
	task.planReadySignaled = true
	close(task.planReadyCh)
	task.stateMu.Unlock()
}

func (task *openClawWarmupTask) waitPlanReady(ctx context.Context) (time.Time, bool) {
	if task == nil {
		return time.Time{}, false
	}

	select {
	case <-ctx.Done():
		return time.Time{}, false
	case <-task.planReadyCh:
	}

	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if task.planReadyAt.IsZero() {
		return time.Time{}, false
	}
	return task.planReadyAt, true
}

func (task *openClawWarmupTask) hasSpokenAny() bool {
	if task == nil {
		return false
	}
	return task.spokeAny.Load()
}

func (s *ChatSession) getOpenClawWarmupTask(correlationID string) *openClawWarmupTask {
	if s == nil {
		return nil
	}
	correlationID = strings.TrimSpace(correlationID)
	s.openClawWarmupMu.Lock()
	defer s.openClawWarmupMu.Unlock()
	task := s.openClawWarmup
	if task == nil {
		return nil
	}
	if correlationID != "" && task.correlationID != correlationID {
		return nil
	}
	return task
}

func (s *ChatSession) takeOpenClawWarmupTask(correlationID string) *openClawWarmupTask {
	if s == nil {
		return nil
	}
	correlationID = strings.TrimSpace(correlationID)
	s.openClawWarmupMu.Lock()
	defer s.openClawWarmupMu.Unlock()
	task := s.openClawWarmup
	if task == nil {
		return nil
	}
	if correlationID != "" && task.correlationID != correlationID {
		return nil
	}
	s.openClawWarmup = nil
	return task
}

func (s *ChatSession) cancelOpenClawWarmup(correlationID string, interrupt bool) bool {
	if s == nil {
		return false
	}

	task := s.getOpenClawWarmupTask(correlationID)
	if task == nil {
		return false
	}
	if task.warmupCtx.Err() != nil {
		return false
	}

	task.cancelWarmupOnly()
	if interrupt && task.hasSpokenAny() {
		s.InterruptAndClearTTSQueueWithReason(fmt.Sprintf("OpenClaw warmup canceled correlation_id=%s", correlationID))
	}

	log.Infof(
		"OpenClaw warmup canceled: device=%s correlation_id=%s interrupt=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		interrupt,
		task.hasSpokenAny(),
	)
	return true
}

func (s *ChatSession) finishOpenClawWarmup(correlationID string, interrupt bool) bool {
	task := s.takeOpenClawWarmupTask(correlationID)
	if task == nil {
		return false
	}

	task.cancelWarmupOnly()
	if interrupt {
		s.InterruptAndClearTTSQueueWithReason(fmt.Sprintf("OpenClaw warmup finished correlation_id=%s interrupt", correlationID))
	}
	s.endOpenClawSpeech(task)

	log.Infof(
		"OpenClaw warmup finished: device=%s correlation_id=%s interrupt=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		interrupt,
		task.hasSpokenAny(),
	)
	return true
}

func (s *ChatSession) beginOpenClawSpeech(task *openClawWarmupTask) {
	if task == nil {
		return
	}
	if !task.markSpeechStarted() {
		return
	}
	s.ttsManager.ClearAudioHistory()
	s.ttsManager.EnqueueTtsStartWithReason(task.sessionCtx, fmt.Sprintf("OpenClaw warmup start correlation_id=%s", task.correlationID))
}

func (s *ChatSession) endOpenClawSpeech(task *openClawWarmupTask) {
	if task == nil {
		return
	}
	if !task.markSpeechEnded() {
		return
	}
	s.ttsManager.GetAndClearAudioHistory()
}

func (s *ChatSession) runOpenClawWarmupTask(task *openClawWarmupTask, userText string) {
	planCtx, cancel := context.WithTimeout(task.warmupCtx, openClawWarmupPlanTimeout)
	defer cancel()
	defer log.Infof(
		"OpenClaw warmup task stopped: device=%s correlation_id=%s warmup_err=%v session_err=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		task.warmupCtx.Err(),
		task.sessionCtx.Err(),
		task.hasSpokenAny(),
	)

	go func() {
		lines, err := s.generateOpenClawWarmupPlan(planCtx, task.correlationID, userText)
		if err != nil {
			if planCtx.Err() == nil {
				log.Warnf("OpenClaw warmup plan generation failed: device=%s correlation_id=%s err=%v", s.clientState.DeviceID, task.correlationID, err)
			}
			task.markPlanReady(time.Time{})
			return
		}
		task.setLines(lines)
		task.markPlanReady(time.Now())
		log.Infof("OpenClaw warmup plan ready: device=%s correlation_id=%s line_count=%d", s.clientState.DeviceID, task.correlationID, len(lines))
	}()

	baseAt, ok := task.waitPlanReady(task.warmupCtx)
	if !ok {
		return
	}

	for idx, delay := range openClawWarmupSchedule {
		if !waitOpenClawWarmupUntil(task.warmupCtx, baseAt.Add(delay)) {
			return
		}
		if task.warmupCtx.Err() != nil {
			return
		}

		text := task.lineAt(idx)
		if text == "" {
			continue
		}

		log.Infof(
			"OpenClaw warmup speaking: device=%s correlation_id=%s slot=%d text=%q",
			s.clientState.DeviceID,
			task.correlationID,
			idx,
			text,
		)
		if err := s.speakOpenClawWarmupLine(task, text); err != nil && task.sessionCtx.Err() == nil {
			log.Warnf("OpenClaw warmup speak failed: device=%s correlation_id=%s slot=%d err=%v", s.clientState.DeviceID, task.correlationID, idx, err)
			return
		}
		task.spokeAny.Store(true)
	}

	// Không dọn active task ở đây: audio warmup cuối có thể vẫn đang gửi/phát,
	// cần tiếp tục cho phép câu đầu từ OpenClaw tới và preempt để ngắt.
}

func waitOpenClawWarmupUntil(ctx context.Context, deadline time.Time) bool {
	wait := time.Until(deadline)
	if wait <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (task *openClawWarmupTask) setLines(lines []string) {
	if task == nil || len(lines) == 0 {
		return
	}

	task.linesMu.Lock()
	defer task.linesMu.Unlock()

	if task.lines == nil {
		task.lines = make([]string, openClawWarmupPlanSize)
	}
	for idx := 0; idx < openClawWarmupPlanSize && idx < len(lines); idx++ {
		if text := sanitizeOpenClawWarmupText(lines[idx]); text != "" {
			task.lines[idx] = text
		}
	}
}

func (task *openClawWarmupTask) lineAt(index int) string {
	if task == nil || index < 0 {
		return ""
	}

	task.linesMu.RLock()
	defer task.linesMu.RUnlock()

	if index >= len(task.lines) {
		return ""
	}
	return strings.TrimSpace(task.lines[index])
}

func (s *ChatSession) speakOpenClawWarmupLine(task *openClawWarmupTask, text string) error {
	text = sanitizeOpenClawWarmupText(text)
	if text == "" {
		return nil
	}
	if task == nil {
		return nil
	}
	if task.sessionCtx.Err() != nil {
		return task.sessionCtx.Err()
	}

	s.beginOpenClawSpeech(task)
	if task.sessionCtx.Err() != nil {
		return task.sessionCtx.Err()
	}

	resp := llm_common.LLMResponseStruct{
		Text:    text,
		IsStart: task.takeWarmupSegmentStartFlag(),
		IsEnd:   true,
	}
	// Cần đảm bảo câu warmup đã đi vào luồng gửi để tránh bị response chính sau đó làm trông như không có hiệu lực.
	return s.ttsManager.handleTextResponse(task.sessionCtx, resp, true)
}

func (s *ChatSession) generateOpenClawWarmupPlan(ctx context.Context, correlationID string, userText string) ([]string, error) {
	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		s.clientState.DeviceConfig.Llm.Provider,
		s.clientState.DeviceConfig.Llm.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("acquire llm provider: %w", err)
	}
	defer pool.Release(llmWrapper)

	dialogue := []*schema.Message{
		schema.SystemMessage(openClawWarmupSystemPrompt),
		schema.UserMessage(buildOpenClawWarmupUserPrompt(userText)),
	}

	msgChan := llmWrapper.GetProvider().ResponseWithContext(
		ctx,
		buildOpenClawWarmupSessionID(s.clientState.SessionID, correlationID),
		dialogue,
		nil,
	)

	raw, err := collectOpenClawWarmupResponse(ctx, msgChan)
	if err != nil {
		return nil, err
	}
	lines := parseOpenClawWarmupPlan(raw)
	if countOpenClawWarmupLines(lines) == 0 {
		return nil, fmt.Errorf("empty warmup plan")
	}
	return lines, nil
}

func buildOpenClawWarmupUserPrompt(userText string) string {
	trimmed := strings.TrimSpace(userText)
	topic := formatOpenClawWarmupTopic(buildOpenClawWarmupHint(userText))
	topicLine := "Không lặp lại các chỉ thị của người dùng như “giúp tôi tra”."
	if topic != "" {
		topicLine = fmt.Sprintf("Nếu cần nhắc tới chủ đề, chỉ được rút gọn thành cụm danh từ “%s”, không lặp lại chỉ thị của người dùng như “giúp tôi tra”.", topic)
	}
	return fmt.Sprintf(
		"Nhiệm vụ lượt này của người dùng:\n%s\n\n%s\n\nMốc phát thực tế lần lượt là: giây 1, giây 10, giây 20, giây 30, giây 40, giây 50, giây 60, giây 70, giây 80, giây 90, giây 100.\nHãy output 11 câu warmup, tương ứng một-một với 11 mốc thời gian trên.",
		trimmed,
		topicLine,
	)
}

func buildOpenClawWarmupSessionID(sessionID string, correlationID string) string {
	base := strings.TrimSpace(sessionID)
	if base == "" {
		base = "openclaw"
	}
	correlationID = strings.TrimSpace(correlationID)
	if len(correlationID) > 12 {
		correlationID = correlationID[:12]
	}
	if correlationID == "" {
		return base + ":warmup"
	}
	return base + ":warmup:" + correlationID
}

func collectOpenClawWarmupResponse(ctx context.Context, msgChan chan *schema.Message) (string, error) {
	var builder strings.Builder

	for {
		select {
		case <-ctx.Done():
			return builder.String(), ctx.Err()
		case msg, ok := <-msgChan:
			if !ok {
				return builder.String(), nil
			}
			if msg == nil {
				continue
			}
			if llm.IsLLMErrorMessage(msg) {
				errMsg := strings.TrimSpace(llm.LLMErrorMessage(msg))
				if errMsg == "" {
					errMsg = "unknown llm error"
				}
				return builder.String(), fmt.Errorf("llm returned error: %s", errMsg)
			}
			if msg.Content != "" {
				builder.WriteString(msg.Content)
			}
		}
	}
}

func parseOpenClawWarmupPlan(raw string) []string {
	lines := make([]string, openClawWarmupPlanSize)

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return lines
	}

	candidate := raw
	start := strings.Index(candidate, "[")
	end := strings.LastIndex(candidate, "]")
	if start >= 0 && end > start {
		candidate = candidate[start : end+1]
	}

	var objectItems []openClawWarmupLine
	if err := json.Unmarshal([]byte(candidate), &objectItems); err == nil {
		return buildOpenClawWarmupPlanLines(objectItemsToStrings(objectItems))
	}

	var stringItems []string
	if err := json.Unmarshal([]byte(candidate), &stringItems); err == nil {
		return buildOpenClawWarmupPlanLines(stringItems)
	}

	log.Warnf("OpenClaw warmup plan parse failed, ignored: raw=%q", raw)
	return lines
}

func objectItemsToStrings(items []openClawWarmupLine) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Text)
	}
	return lines
}

func buildOpenClawWarmupPlanLines(items []string) []string {
	lines := make([]string, openClawWarmupPlanSize)
	for idx := 0; idx < openClawWarmupPlanSize && idx < len(items); idx++ {
		if text := sanitizeOpenClawWarmupText(items[idx]); text != "" {
			lines[idx] = text
		}
	}
	return lines
}

func countOpenClawWarmupLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func sanitizeOpenClawWarmupText(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\"'`[]{}")
	text = strings.TrimLeft(text, "0123456789.、- ")
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	if isInvalidOpenClawWarmupText(text) {
		return ""
	}

	runes := []rune(text)
	if len(runes) > 16 {
		return ""
	}
	return text
}

func isInvalidOpenClawWarmupText(text string) bool {
	for _, bad := range []string{
		"giúp tôi",
		"cho tôi",
		"nói cho tôi",
		"vui lòng giúp",
		"phiền bạn giúp",
		"có thể giúp tôi",
		"bạn có thể giúp tôi",
		"giúp tra",
		"giúp xem",
		"giúp hỏi",
	} {
		if strings.Contains(text, bad) {
			return true
		}
	}
	return false
}

func buildOpenClawWarmupHint(userText string) string {
	trimmed := strings.TrimSpace(userText)
	if trimmed == "" {
		return ""
	}

	normalized := removePunctuation(trimmed)
	if normalized == "" {
		return ""
	}
	normalized = trimOpenClawWarmupCommandPrefix(normalized)
	normalized = trimOpenClawWarmupQuestionSuffix(normalized)
	if normalized == "" {
		return ""
	}

	for _, keyword := range []string{"thời tiết", "nhiệt độ", "dự báo"} {
		if idx := strings.Index(normalized, keyword); idx >= 0 {
			limit := idx + len([]rune(keyword))
			runes := []rune(normalized)
			if limit > len(runes) {
				limit = len(runes)
			}
			normalized = string(runes[:limit])
			break
		}
	}

	runes := []rune(normalized)
	if len(runes) > 10 {
		runes = runes[:10]
	}
	for len(runes) > 0 {
		last := runes[len(runes)-1]
		if last == 'v' || last == 'à' || last == '?' {
			runes = runes[:len(runes)-1]
			continue
		}
		break
	}
	return string(runes)
}

func trimOpenClawWarmupCommandPrefix(text string) string {
	trimmed := strings.TrimSpace(text)
	for {
		changed := false
		for _, prefix := range []string{
			"phiền bạn giúp tôi tra cứu",
			"phiền bạn giúp tôi tra",
			"phiền bạn xem giúp tôi",
			"vui lòng giúp tôi tra cứu",
			"vui lòng giúp tôi tra",
			"vui lòng xem giúp tôi",
			"giúp tôi tra cứu",
			"giúp tôi tra",
			"xem giúp tôi",
			"hỏi giúp tôi",
			"tra cứu cho tôi",
			"tra cho tôi",
			"xem cho tôi",
			"bạn có thể giúp tôi tra",
			"bạn có thể xem giúp tôi",
			"có thể giúp tôi tra",
			"có thể xem giúp tôi",
			"tôi muốn biết",
			"tôi muốn hỏi",
			"tôi muốn hỏi",
			"cho tôi hỏi",
			"cho hỏi",
			"tra cứu",
			"tra",
			"xem",
			"hỏi",
			"giúp tôi tra cứu",
			"giúp tôi tra",
			"xem giúp tôi",
			"hỏi giúp tôi",
			"tra cứu cho tôi",
			"tra cho tôi",
			"xem cho tôi",
			"tra cứu",
			"tra",
			"xem",
			"hỏi",
		} {
			if strings.HasPrefix(trimmed, prefix) {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return trimmed
}

func trimOpenClawWarmupQuestionSuffix(text string) string {
	trimmed := strings.TrimSpace(text)
	for _, suffix := range []string{
		"thế nào",
		"ra sao",
		"bao nhiêu",
		"là gì",
		"là gì",
		"không",
		"nhỉ",
		"nhé",
		"đi",
	} {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, suffix))
	}
	return trimmed
}

func formatOpenClawWarmupTopic(hint string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return ""
	}
	for _, keyword := range []string{"thời tiết", "nhiệt độ", "dự báo"} {
		if idx := strings.Index(hint, keyword); idx > 0 {
			prefix := strings.TrimSpace(hint[:idx])
			if prefix == "" || strings.HasSuffix(prefix, "về") {
				return hint
			}
			return strings.TrimSpace(prefix + " " + hint[idx:])
		}
	}
	return hint
}
