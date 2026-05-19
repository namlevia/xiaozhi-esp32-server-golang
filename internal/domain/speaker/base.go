package speaker

import (
	"context"
)

// SpeakerProvider interface speaker recognition provider
type SpeakerProvider interface {
	// StartStreaming Khởi động nhận diện streaming
	StartStreaming(ctx context.Context, sampleRate int, agentId string) error

	// SendAudioChunk Gửi audio data chunk
	SendAudioChunk(ctx context.Context, audioData []float32) error

	// FinishAndIdentify Hoàn tất input và lấy kết quả nhận diện
	FinishAndIdentify(ctx context.Context) (*IdentifyResult, error)

	// IsActive Kiểm tra có đang active không
	IsActive() bool

	// Close Đóng kết nối
	Close() error
}

// GetSpeakerProvider Lấy speaker recognition provider
func GetSpeakerProvider(config map[string]interface{}) (SpeakerProvider, error) {
	return NewAsrServerProvider(config)
}
