package play_music

import (
	"context"
)

// MusicPlayerInterface interface music player
type MusicPlayerInterface interface {
	// PlayMusicStream phát nhạc từ URL, trả về channel stream audio
	PlayMusicStream(ctx context.Context, url string) (chan []byte, error)

	// GetPlayerInfo Lấy thông tin player
	GetPlayerInfo() map[string]interface{}

	// Stop Dừng player
	Stop() error
}

// MusicPlayerConfig Cấu hình music player
type MusicPlayerConfig struct {
	FrameDuration int    `json:"frame_duration"` // thời lượng frame (ms), mặc định 20ms
	AudioFormat   string `json:"audio_format"`   // định dạng audio, mặc định "mp3"
}

// DefaultMusicPlayerConfig trả về cấu hình music player mặc định
func DefaultMusicPlayerConfig() *MusicPlayerConfig {
	return &MusicPlayerConfig{
		FrameDuration: 20,    // 20ms
		AudioFormat:   "mp3", // định dạng MP3
	}
}

// ToMap Chuyển config thành map
func (c *MusicPlayerConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"frame_duration": c.FrameDuration,
		"audio_format":   c.AudioFormat,
	}
}

// AudioStreamInfo Thông tin stream audio
type AudioStreamInfo struct {
	URL           string `json:"url"`
	Format        string `json:"format"`         // định dạng audio, ví dụ "mp3", "wav"
	SampleRate    int    `json:"sample_rate"`    // sample rate
	Channels      int    `json:"channels"`       // số channel
	Duration      int64  `json:"duration"`       // thời lượng (mili giây)
	ContentLength int64  `json:"content_length"` // độ dài nội dung (byte)
}

// PlaybackStatus Trạng thái phát
type PlaybackStatus int

const (
	StatusIdle PlaybackStatus = iota
	StatusPlaying
	StatusPaused
	StatusStopped
	StatusError
)

// String Trả về biểu diễn string của trạng thái
func (s PlaybackStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusPlaying:
		return "playing"
	case StatusPaused:
		return "paused"
	case StatusStopped:
		return "stopped"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// PlaybackEvent Sự kiện phát
type PlaybackEvent struct {
	Type      string      `json:"type"`      // loại event: "started", "progress", "finished", "error"
	Timestamp int64       `json:"timestamp"` // timestamp
	Message   string      `json:"message"`   // message event
	Data      interface{} `json:"data"`      // dữ liệu bổ sung
}

// StreamingStats Thống kê phát streaming
type StreamingStats struct {
	BytesDownloaded int64          `json:"bytes_downloaded"` // số byte đã tải
	BytesDecoded    int64          `json:"bytes_decoded"`    // số byte đã decode
	FramesGenerated int64          `json:"frames_generated"` // số frame đã tạo
	StartTime       int64          `json:"start_time"`       // thời gian bắt đầu
	FirstFrameTime  int64          `json:"first_frame_time"` // thời gian frame đầu
	Status          PlaybackStatus `json:"status"`           // trạng thái hiện tại
	ErrorCount      int            `json:"error_count"`      // số lỗi
}
