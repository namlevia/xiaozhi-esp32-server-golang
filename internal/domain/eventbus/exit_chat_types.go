package eventbus

import (
	"time"

	. "xiaozhi-esp32-server-golang/internal/data/client"
)

// ExitChatEvent là sự kiện thoát chat.
type ExitChatEvent struct {
	// Trạng thái client
	ClientState *ClientState

	// Lý do thoát
	Reason string // "user chủ động thoát", "tool call thoát", "timeout thoát"...

	// Cách trigger thoát
	TriggerType string // "exit_words" (phát hiện từ thoát), "tool_call" (tool call), "timeout"...

	// Text raw user nhập, nếu có
	UserText string

	// Timestamp
	Timestamp time.Time
}
