package plugins

import "xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"

// Init khởi tạo các transform liên quan đến output.
func Init(registry *streamtransform.Registry) {
	if registry == nil {
		return
	}

	// Đăng ký plugin định hình output (phân đoạn text + gom tool call).
	RegisterOutputSegmenter(registry)
}
