package eventbus

import (
	"time"
	. "xiaozhi-esp32-server-golang/internal/data/client"

	"github.com/cloudwego/eino/schema"
)

// AddMessageEvent là sự kiện thêm message thống nhất.
type AddMessageEvent struct {
	// Trạng thái client
	ClientState *ClientState

	// Nội dung message, thống nhất dùng schema.Message
	// schema.Message là định dạng message LLM chuẩn, gồm:
	// - Role: role message (User/Assistant/System/Tool)
	// - Content: nội dung text message
	// - ToolCalls: danh sách tool call, tùy chọn
	// - ToolCallID: ID tool call, dùng cho role Tool
	Msg schema.Message

	// Message ID, dùng để liên kết lưu hai giai đoạn
	MessageID string

	// Dữ liệu audio, tùy chọn, không thuộc định dạng chuẩn schema.Message
	// Giai đoạn 1: AudioData = nil, chỉ lưu text
	// Giai đoạn 2: AudioData != nil, update audio
	AudioData [][]byte // Mảng frame audio TTS/ASR, định dạng Opus hoặc PCM
	AudioSize int      // Kích thước audio (byte)

	// Thông tin định dạng audio, không thuộc định dạng chuẩn schema.Message
	SampleRate int // Sample rate
	Channels   int // Số kênh

	// Metadata, không thuộc định dạng chuẩn schema.Message
	Timestamp   time.Time
	TTSDuration int // Thời gian TTS (ms)

	// Dấu hiệu giai đoạn
	IsUpdate bool // true=update audio, false=thêm message mới
}
