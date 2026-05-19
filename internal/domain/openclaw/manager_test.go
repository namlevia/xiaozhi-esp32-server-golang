package openclaw

import (
	"strings"
	"testing"
)

func TestHandleResponseIgnoresSnapshotDuplicateChunk(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-1"
	streamID := "stream-1"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool, phase string) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id": "device-1",
				"seq":       seq,
				"done":      done,
				"stream_id": streamID,
				"phase":     phase,
			},
		}, deliver)
	}

	send(1, "Ngày mai thời tiết Hà Nội đẹp, nhiệt độ 1", false, "chunk")
	send(2, "5 đến", false, "chunk")
	send(3, "22", false, "chunk")
	send(4, " độ.", false, "chunk")
	send(5, "Ngày mai thời tiết Hà Nội đẹp, nhiệt độ 15 đến 22  độ.", false, "chunk")
	send(6, "", true, "final")

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if !events[0].IsStart || events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
	if openClawCanonicalKey(events[0].Text) != openClawCanonicalKey("Ngày mai thời tiết Hà Nội đẹp, nhiệt độ 15 đến 22  độ.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected end event: %+v", events[1])
	}
}

func TestHandleResponseIgnoresChunkReplayWithPunctuationVariants(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-punct-replay"
	streamID := "stream-punct-replay"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool, phase string) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id": "device-1",
				"seq":       seq,
				"done":      done,
				"stream_id": streamID,
				"phase":     phase,
			},
		}, deliver)
	}

	send(1, "Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 1", false, "chunk")
	send(2, "5 đến ", false, "chunk")
	send(3, "19", false, "chunk")
	send(4, " độ,", false, "chunk")
	send(5, "không có mưa", false, "chunk")
	send(6, "thời tiết đẹp.", false, "chunk")
	send(7, "Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 15 đến 19 độ, không có mưa, thời tiết đẹp.", false, "chunk")
	send(8, "", true, "final")

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if !events[0].IsStart || events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
	if openClawComparableKey(events[0].Text) != openClawComparableKey("Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 15 đến 19 độ, không có mưa, thời tiết đẹp.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected end event: %+v", events[1])
	}
}

func TestHandleResponseIgnoresDuplicateSeq(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-dup-seq"
	streamID := "stream-dup-seq"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id": "device-1",
				"seq":       seq,
				"done":      done,
				"stream_id": streamID,
				"phase":     "chunk",
			},
		}, deliver)
	}

	send(1, "Ngày mai Hà Nội th", false)
	send(2, "ời tiết đẹp.", false)
	send(2, "Ngày mai thời tiết Hà Nội đẹp.", false)
	send(3, "", true)

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if openClawComparableKey(events[0].Text) != openClawComparableKey("Ngày mai thời tiết Hà Nội đẹp.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected end event: %+v", events[1])
	}
}

func TestHandleResponseBuffersExplicitSnapshotWithoutReplay(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-explicit-snapshot"
	streamID := "stream-explicit-snapshot"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool, phase string, contentType string) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id":    "device-1",
				"seq":          seq,
				"done":         done,
				"stream_id":    streamID,
				"phase":        phase,
				"content_type": contentType,
			},
		}, deliver)
	}

	send(1, "Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 1", false, "chunk", "")
	send(2, "5 đến 19 độ,không có mưathời tiết đẹp.", false, "chunk", "")
	send(3, "Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 15 đến 19 độ, không có mưa, thời tiết đẹp.", false, "snapshot", "snapshot")
	send(4, "", true, "final", "")

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if openClawComparableKey(events[0].Text) != openClawComparableKey("Đà Nẵng ngày kia nhiều mây rồi nắng, nhiệt độ 15 đến 19 độ,không có mưathời tiết đẹp.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected end event: %+v", events[1])
	}
}

func TestHandleResponseUsesExplicitSnapshotWhenNoDeltaExists(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-only-snapshot"
	streamID := "stream-only-snapshot"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
		Content: "Ngày mai thời tiết Hà Nội đẹp.",
		Metadata: map[string]interface{}{
			"device_id":    "device-1",
			"seq":          int64(1),
			"done":         false,
			"stream_id":    streamID,
			"phase":        "snapshot",
			"content_type": "snapshot",
		},
	}, deliver)
	manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
		Content: "",
		Metadata: map[string]interface{}{
			"device_id":    "device-1",
			"seq":          int64(2),
			"done":         true,
			"stream_id":    streamID,
			"phase":        "final",
			"content_type": "",
		},
	}, deliver)

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if openClawCanonicalKey(events[0].Text) != openClawCanonicalKey("Ngày mai thời tiết Hà Nội đẹp.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if !events[0].IsStart || events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

func TestHandleResponseTreatsGrowingSnapshotAsReplacementBeforeSentenceEnds(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-2"
	streamID := "stream-2"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool, phase string) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id": "device-1",
				"seq":       seq,
				"done":      done,
				"stream_id": streamID,
				"phase":     phase,
			},
		}, deliver)
	}

	send(1, "Ngày mai Hà Nội th", false, "chunk")
	send(2, "Ngày mai thời tiết Hà Nội đẹp", false, "chunk")
	send(3, "。", false, "chunk")
	send(4, "", true, "final")

	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want 1, events=%+v", len(events), events)
	}
	if openClawCanonicalKey(events[0].Text) != openClawCanonicalKey("Ngày mai thời tiết Hà Nội đẹp") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if !events[0].IsStart || !events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
}

func TestHandleResponseTestDeviceKeepsOnlyIncrementalSuffix(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-test"
	streamID := "stream-test"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	send := func(seq int64, content string, done bool) {
		manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
			Content: content,
			Metadata: map[string]interface{}{
				"device_id": "__openclaw_test__:device-1",
				"seq":       seq,
				"done":      done,
				"stream_id": streamID,
				"phase":     "chunk",
			},
		}, deliver)
	}

	send(1, "Ngày mai Hà Nội th", false)
	send(2, "Ngày mai thời tiết Hà Nội đẹp", false)
	send(3, "。", true)

	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want 1, events=%+v", len(events), events)
	}
	if openClawCanonicalKey(events[0].Text) != openClawCanonicalKey("Ngày mai thời tiết Hà Nội đẹp") {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if !events[0].IsStart || !events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
}

func TestHandleResponseFallsBackToSnapshotOnEmptyFinal(t *testing.T) {
	manager := &Manager{offline: make(map[string][]OfflineMessage)}
	session := &AgentSession{agentID: "agent-1"}
	correlationID := "corr-snapshot-final"
	streamID := "stream-snapshot-final"

	var events []ResponseDelivery
	deliver := func(event ResponseDelivery) bool {
		events = append(events, event)
		return true
	}

	manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
		Content: "Ngày mai thời tiết Hà Nội đẹp.",
		Metadata: map[string]interface{}{
			"device_id": "device-1",
			"seq":       int64(1),
			"done":      false,
			"stream_id": streamID,
			"phase":     "chunk",
		},
	}, deliver)
	manager.HandleResponse("agent-1", session, correlationID, ResponsePayload{
		Content: "",
		Metadata: map[string]interface{}{
			"device_id": "device-1",
			"seq":       int64(2),
			"done":      true,
			"stream_id": streamID,
			"phase":     "final",
		},
	}, deliver)

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want 2, events=%+v", len(events), events)
	}
	if openClawCanonicalKey(events[0].Text) != openClawCanonicalKey("Ngày mai thời tiết Hà Nội đẹp.") {
		t.Fatalf("unexpected first event text: %q", events[0].Text)
	}
	if !events[0].IsStart || events[0].IsEnd {
		t.Fatalf("unexpected first event flags: %+v", events[0])
	}
	if events[1].Text != "" || events[1].IsStart || !events[1].IsEnd {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

func TestBuildOpenClawPromptedContentWrapsUserMessage(t *testing.T) {
	got := buildOpenClawPromptedContent("  Thời tiết Hà Nội ngày kia thế nào?  ")

	if !strings.Contains(got, "Bạn đang trò chuyện trực tiếp với người dùng trong vai trò trợ lý giọng nói.") {
		t.Fatalf("missing voice assistant prompt: %q", got)
	}
	if !strings.Contains(got, "Câu trả lời cần ngắn gọn, tự nhiên, giống văn nói và phù hợp để đọc bằng giọng nói.") {
		t.Fatalf("missing concise speech constraint: %q", got)
	}
	if !strings.Contains(got, "Tin nhắn người dùng:\nThời tiết Hà Nội ngày kia thế nào?") {
		t.Fatalf("missing wrapped user message: %q", got)
	}
	if strings.Contains(got, "  Thời tiết Hà Nội ngày kia thế nào?  ") {
		t.Fatalf("user message was not trimmed: %q", got)
	}
}

func TestExtractOpenClawSentencesKeepsLeadingClauseTogether(t *testing.T) {
	text := "Được, tôi sẽ kiểm tra thời tiết TP.HCM hôm nay trước. Sau đó tôi xử lý tiếp"

	sentences, remaining := extractOpenClawSentences(text, openClawSentenceMinLen, true)

	if len(sentences) != 1 {
		t.Fatalf("unexpected sentence count: got %d want 1", len(sentences))
	}
	if sentences[0] != "Được, tôi sẽ kiểm tra thời tiết TP.HCM hôm nay trước." {
		t.Fatalf("unexpected first sentence: %q", sentences[0])
	}
	if remaining != "Sau đó tôi xử lý tiếp" {
		t.Fatalf("unexpected remaining text: %q", remaining)
	}
}

func TestExtractOpenClawSentencesMergesShortClauses(t *testing.T) {
	text := "Được. Tạm vậy trước. Sau đó tôi xử lý tiếp."

	sentences, remaining := extractOpenClawSentences(text, openClawSentenceMinLen, true)

	if len(sentences) != 3 {
		t.Fatalf("unexpected sentence count: got %d want 3", len(sentences))
	}
	if sentences[0] != "Được." || sentences[1] != "Tạm vậy trước." || sentences[2] != "Sau đó tôi xử lý tiếp." {
		t.Fatalf("unexpected sentence split: %+v", sentences)
	}
	if remaining != "" {
		t.Fatalf("unexpected remaining text: %q", remaining)
	}
}

func TestNormalizeOpenClawSpeechTextStripsMarkdownAndBullets(t *testing.T) {
	raw := "🌤️ **Dự báo thời tiết Hà Nội ngày kia (9/3)**\n\n- **Nhiệt độ**: 3°C ~ 12°C\n- **Thời tiết**: trời nắng☀️"

	got := normalizeOpenClawSpeechText(raw)

	if strings.Contains(got, "**") {
		t.Fatalf("unexpected markdown marker in normalized text: %q", got)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("unexpected newline in normalized text: %q", got)
	}
	if !strings.Contains(got, "Nhiệt độ: 3°C ~ 12°C") {
		t.Fatalf("missing normalized temperature segment: %q", got)
	}
	if !strings.Contains(got, "Thời tiết: trời nắng☀️") {
		t.Fatalf("missing normalized weather segment: %q", got)
	}
}

func TestExtractOpenClawSentencesGroupsWeatherListIntoLongerSegments(t *testing.T) {
	text := "🌤️ **Dự báo thời tiết Hà Nội ngày kia (9/3)**\n\n- **Nhiệt độ**: 3°C ~ 12°C\n- **Thời tiết**: trời nắng☀️\n- **Lượng mưa**: không mưa\n- **Độ ẩm**: 15% ~ 38%\n- **Hướng gió**: gió tây nam, tốc độ 2-13km/h\n\nNgày kia thời tiết Hà Nội đẹp, chủ yếu trời nắng, nhiệt độ cao nhất 12°C, thấp nhất 3°C."

	sentences, remaining := extractOpenClawSentences(text, openClawSentenceMinLen, true)

	if len(sentences) == 0 {
		t.Fatal("expected at least one emitted sentence")
	}
	if len(sentences) != 1 {
		t.Fatalf("unexpected sentence count: got %d want 1", len(sentences))
	}
	if remaining != "" {
		t.Fatalf("unexpected remaining text: %q", remaining)
	}
	if strings.Contains(sentences[0], "**") || strings.Contains(sentences[0], "\n") {
		t.Fatalf("unexpected raw markdown in first sentence: %q", sentences[0])
	}
	if !strings.Contains(sentences[0], "Nhiệt độ:") || !strings.Contains(sentences[0], "Thời tiết:") {
		t.Fatalf("first sentence still too short: %q", sentences[0])
	}
	if !strings.Contains(sentences[0], "nhiệt độ cao nhất 12°C") {
		t.Fatalf("missing summary in final sentence: %q", sentences[0])
	}
}
