package asr

import (
	"context"
	"fmt"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/asr/doubao"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// Asr là interface nhận diện giọng nói.
type AsrProvider interface {
	// Process xử lý toàn bộ đoạn audio một lần và trả về kết quả nhận diện đầy đủ.
	Process(pcmData []float32) (string, error)

	// StreamingRecognize là interface nhận diện streaming.
	// Dữ liệu audio input đi qua channel audioStream, kết quả nhận diện lấy từ channel trả về.
	// Khi audioStream đóng nghĩa là input kết thúc; kết quả cuối sẽ được gửi qua channel trả về rồi đóng channel đó.
	// Có thể dùng ctx để điều khiển hủy và timeout quá trình nhận diện.
	StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error)
	// Close đóng tài nguyên và giải phóng kết nối.
	Close() error
	// IsValid kiểm tra tài nguyên có hợp lệ hay không.
	IsValid() bool
}

// NewAsrProvider tạo instance ASR mới.
// asrType: loại engine ASR, hiện hỗ trợ "funasr".
// config: config engine ASR, kiểu map[string]interface{}.
func NewAsrProvider(asrType string, config map[string]interface{}) (AsrProvider, error) {
	// Ưu tiên dùng provider trong config, nếu không có thì dùng provider từ tham số.
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		asrType = configProvider
	}
	switch asrType {
	case constants.AsrTypeFunAsr:
		return NewFunasrAdapter(config)
	case constants.AsrTypeAliyunFunASR:
		return NewAliyunFunASRAdapter(config)
	case constants.AsrTypeDoubao:
		log.Info("Dùng provider ASR Doubao")
		provider, err := doubao.NewDoubaoV2Adapter(config)
		if err != nil {
			log.Errorf("Tạo adapter ASR Doubao thất bại: %v", err)
		} else {
			log.Info("Tạo adapter ASR Doubao thành công")
		}
		return provider, err
	case constants.AsrTypeAliyunQwen3:
		log.Info("Dùng provider ASR Aliyun Qwen3")
		provider, err := NewAliyunQwen3Adapter(config)
		if err != nil {
			log.Errorf("Tạo adapter ASR Aliyun Qwen3 thất bại: %v", err)
		} else {
			log.Info("Tạo adapter ASR Aliyun Qwen3 thành công")
		}
		return provider, err
	case constants.AsrTypeXunfei:
		log.Info("Sử dụng provider ASR Xunfei")
		provider, err := NewXunfeiAdapter(config)
		if err != nil {
			log.Errorf("Tạo adapter ASR Xunfei thất bại: %v", err)
		} else {
			log.Info("Tạo adapter ASR Xunfei thành công")
		}
		return provider, err
	case constants.AsrTypeWyomingVietnamese:
		log.Info("Sử dụng provider ASR tiếng Việt Wyoming")
		provider, err := NewWyomingVietnameseAdapter(config)
		if err != nil {
			log.Errorf("Tạo adapter ASR tiếng Việt Wyoming thất bại: %v", err)
		} else {
			log.Info("Tạo adapter ASR tiếng Việt Wyoming thành công")
		}
		return provider, err
	default:
		return nil, fmt.Errorf("loại ASR không được hỗ trợ: %s; hiện hỗ trợ 'funasr', 'aliyun_funasr', 'doubao', 'aliyun_qwen3', 'xunfei', 'wyoming_vietnamese_asr'", asrType)
	}
}
