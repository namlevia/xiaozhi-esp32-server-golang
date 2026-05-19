package eventbus

import (
	"context"
	"time"
)

// UserMessageEvent là sự kiện message của user.
// Deprecated: dùng AddMessageEvent thay thế, thống nhất dùng sự kiện TopicAddMessage.
type UserMessageEvent struct {
	Ctx       context.Context
	SessionID string
	DeviceID  string
	AgentID   string

	// Kết quả ASR
	Text      string
	AudioData []byte // Dữ liệu audio raw, PCM float32 chuyển sang byte
	AudioSize int    // Số sample audio

	// Thông tin định dạng audio, dùng để chuyển sang WAV
	SampleRate int // Sample rate
	Channels   int // Số kênh

	// Metadata
	Timestamp time.Time
}

// AssistantMessageEvent là sự kiện phản hồi của bot.
// Deprecated: dùng AddMessageEvent thay thế, thống nhất dùng sự kiện TopicAddMessage.
type AssistantMessageEvent struct {
	Ctx       context.Context
	SessionID string
	DeviceID  string
	AgentID   string

	// Kết quả LLM
	Text string

	// Kết quả TTS
	AudioData [][]byte // Dữ liệu audio tổng hợp, định dạng Opus, mảng frame audio
	AudioSize int      // Kích thước audio (byte)

	// Thông tin định dạng audio, dùng để chuyển sang WAV
	SampleRate int // Sample rate
	Channels   int // Số kênh

	// Metadata
	TTSDuration int // ms
	Timestamp   time.Time
}
