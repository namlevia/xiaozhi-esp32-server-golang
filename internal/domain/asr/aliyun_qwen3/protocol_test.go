package aliyun_qwen3

import (
	"encoding/json"
	"testing"
)

func TestServerEventUnmarshalConversationItemCreatedStringID(t *testing.T) {
	raw := []byte(`{"event_id":"event_1","type":"conversation.item.created","item":{"id":"item_123","object":"realtime.item","type":"message","status":"in_progress","role":"assistant","content":[{"type":"input_audio"}]}}`)

	var event ServerEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if event.Item == nil {
		t.Fatal("expected item to be parsed")
	}
	if event.Item.ID != "item_123" {
		t.Fatalf("expected string item id, got %q", event.Item.ID)
	}
}

func TestGetTranscriptionTextPrefersTranscript(t *testing.T) {
	event := &ServerEvent{
		Transcript: "các bạn đang ngắm hoa à?",
		Stash:      "các bạn đang ngắm hoa à",
	}

	if got := GetTranscriptionText(event); got != "các bạn đang ngắm hoa à?" {
		t.Fatalf("expected transcript text, got %q", got)
	}
}

func TestGetTranscriptionTextFallsBackToStash(t *testing.T) {
	event := &ServerEvent{
		Text:  "",
		Stash: "các bạn đang ngắm hoa à",
	}

	if got := GetTranscriptionText(event); got != "các bạn đang ngắm hoa à" {
		t.Fatalf("expected stash fallback, got %q", got)
	}
}
