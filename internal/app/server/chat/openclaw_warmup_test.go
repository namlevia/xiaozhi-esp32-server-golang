package chat

import (
	"strings"
	"testing"
)

func TestParseOpenClawWarmupPlanObjects(t *testing.T) {
	got := parseOpenClawWarmupPlan(`[{"text":"Tôi xem thời tiết trước."},{"text":"Tôi tiếp tục theo dõi tình hình thời tiết."},{"text":"Tôi vẫn đang xử lý vấn đề này."},{"text":"Kết quả thời tiết vẫn đang được cập nhật."},{"text":"Tôi tiếp tục theo dõi thời tiết."},{"text":"Tôi tiếp tục kiểm tra thời tiết."},{"text":"Dữ liệu thời tiết vẫn đang cập nhật."},{"text":"Tôi tiếp tục theo dõi dự báo mới nhất."},{"text":"Tôi vẫn đang xác nhận lần cuối."},{"text":"Có kết quả tôi sẽ báo bạn ngay."},{"text":"Tôi xem lại thời tiết một chút."}]`)

	if len(got) != openClawWarmupPlanSize {
		t.Fatalf("unexpected plan size: got %d want %d", len(got), openClawWarmupPlanSize)
	}
	if got[0] != "Tôi xem thời tiết trước." {
		t.Fatalf("unexpected first line: %q", got[0])
	}
	if got[4] != "Tôi tiếp tục theo dõi thời tiết." {
		t.Fatalf("unexpected last line: %q", got[4])
	}
	if got[9] != "Có kết quả tôi sẽ báo bạn ngay." {
		t.Fatalf("unexpected tenth line: %q", got[9])
	}
	if got[10] != "Tôi xem lại thời tiết một chút." {
		t.Fatalf("unexpected eleventh line: %q", got[10])
	}
}

func TestParseOpenClawWarmupPlanReturnsEmptyOnInvalidJSON(t *testing.T) {
	got := parseOpenClawWarmupPlan("not-json")

	for idx, line := range got {
		if line != "" {
			t.Fatalf("expected empty line at %d, got %q", idx, line)
		}
	}
}

func TestBuildOpenClawWarmupHint(t *testing.T) {
	got := buildOpenClawWarmupHint("giúp tôi tra thời tiết Hà Nội hôm nay thế nào?")
	if got == "" {
		t.Fatal("expected non-empty hint")
	}
	if strings.Contains(got, "giúp tôi") {
		t.Fatalf("hint should not contain user command: %q", got)
	}
	if len([]rune(got)) > 10 {
		t.Fatalf("hint too long: %q", got)
	}
}

func TestBuildOpenClawWarmupHintWeatherTopic(t *testing.T) {
	got := buildOpenClawWarmupHint("thời tiết Đà Nẵng ngày kia thế nào?")
	if got != "thời tiết Đà Nẵng ngày kia" {
		t.Fatalf("unexpected weather hint: %q", got)
	}
}

func TestBuildOpenClawWarmupUserPromptIncludesTimeline(t *testing.T) {
	got := buildOpenClawWarmupUserPrompt("thời tiết Đà Nẵng ngày kia thế nào?")
	if !strings.Contains(got, "Nhiệm vụ lượt này của người dùng:") {
		t.Fatalf("task label missing from prompt: %q", got)
	}
	if !strings.Contains(got, "chỉ được rút gọn thành cụm danh từ “thời tiết Đà Nẵng ngày kia”") {
		t.Fatalf("topic hint missing from prompt: %q", got)
	}
	if !strings.Contains(got, "giây 1, giây 10, giây 20, giây 30, giây 40, giây 50, giây 60, giây 70, giây 80, giây 90, giây 100") {
		t.Fatalf("timeline missing from prompt: %q", got)
	}
}

func TestFormatOpenClawWarmupTopicWeather(t *testing.T) {
	got := formatOpenClawWarmupTopic("thời tiết Đà Nẵng ngày kia")
	if got != "thời tiết Đà Nẵng ngày kia" {
		t.Fatalf("unexpected formatted topic: %q", got)
	}
}

func TestSanitizeOpenClawWarmupTextRejectsUserCommandEcho(t *testing.T) {
	got := sanitizeOpenClawWarmupText("Tôi xem thử giúp tôi tra cứu một chút.")
	if got != "" {
		t.Fatalf("expected invalid warmup text to be rejected, got %q", got)
	}
}

func TestTakeWarmupSegmentStartFlagOnlyMarksFirstWarmupSentence(t *testing.T) {
	task := &openClawWarmupTask{nextWarmupSegmentIsStart: true}

	if !task.takeWarmupSegmentStartFlag() {
		t.Fatal("expected first warmup sentence to carry start flag")
	}
	if task.takeWarmupSegmentStartFlag() {
		t.Fatal("expected subsequent warmup sentence to clear start flag")
	}
}
