package speaker

import (
	"context"
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"
)

// AsrServerProvider speaker recognition provider asr_server
type AsrServerProvider struct {
	streamingClient *StreamingClient
	threshold       float32 // ngưỡng speaker recognition
	isActive        bool
	mutex           sync.Mutex
}

// NewAsrServerProvider tạo speaker recognition provider asr_server
func NewAsrServerProvider(config map[string]interface{}) (*AsrServerProvider, error) {
	baseURL, ok := config["base_url"].(string)
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("thiếu field service.base_url trong cấu hình")
	}

	// Đọc cấu hình ngưỡng, mặc định là 0.4
	threshold := float32(0.4)
	if thresholdVal, ok := config["threshold"]; ok {
		switch v := thresholdVal.(type) {
		case float64:
			threshold = float32(v)
		case float32:
			threshold = v
		case int:
			threshold = float32(v)
		case int64:
			threshold = float32(v)
		}
		// Xác thực phạm vi ngưỡng
		if threshold < 0 || threshold > 1 {
			log.Warnf("Ngưỡng %.4f vượt phạm vi hợp lệ [0.0, 1.0], dùng giá trị mặc định 0.4", threshold)
			threshold = 0.4
		}
	}

	streamingClient := NewStreamingClient(baseURL)
	return &AsrServerProvider{
		streamingClient: streamingClient,
		threshold:       threshold,
		isActive:        false,
	}, nil
}

// StartStreaming Khởi động nhận diện streaming
func (p *AsrServerProvider) StartStreaming(ctx context.Context, sampleRate int, agentId string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isActive {
		return nil // đã active, trả về trực tiếp
	}

	err := p.streamingClient.Connect(sampleRate, agentId, p.threshold)
	if err != nil {
		log.Warnf("Khởi động stream speaker recognition thất bại: %v", err)
		return err
	}

	p.isActive = true
	log.Debugf("Stream speaker recognition đã khởi động, sample rate: %d Hz, agent_id: %s, ngưỡng: %.4f", sampleRate, agentId, p.threshold)
	return nil
}

// SendAudioChunk Gửi audio chunk
func (p *AsrServerProvider) SendAudioChunk(ctx context.Context, pcmData []float32) error {
	p.mutex.Lock()
	isActive := p.isActive
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	if !isActive {
		return nil // chưa active, bỏ qua im lặng
	}

	err := streamingClient.SendAudioChunk(pcmData)
	if err != nil {
		log.Warnf("Gửi audio chunk tới dịch vụ speaker recognition thất bại: %v", err)
		// Khi gửi thất bại, đánh dấu trạng thái inactive
		p.mutex.Lock()
		p.isActive = false
		p.mutex.Unlock()
		return err
	}

	return nil
}

// FinishAndIdentify Hoàn tất nhận diện và lấy kết quả
func (p *AsrServerProvider) FinishAndIdentify(ctx context.Context) (*IdentifyResult, error) {
	p.mutex.Lock()
	if !p.isActive {
		p.mutex.Unlock()
		return nil, nil // chưa active, trả về nil
	}
	p.isActive = false
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	result, err := streamingClient.FinishAndIdentify(ctx)

	if err != nil {
		log.Warnf("Lấy kết quả speaker recognition thất bại: %v", err)
		return nil, err
	}

	return result, nil
}

// PeekAndIdentify Lấy kết quả nhận diện trung gian (không kết thúc lượt hiện tại)
// Trả về: kết quả nhận diện, có bị server debounce không, lỗi
func (p *AsrServerProvider) PeekAndIdentify(ctx context.Context, requestID string) (*IdentifyResult, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	p.mutex.Lock()
	isActive := p.isActive
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	if !isActive {
		return nil, false, nil
	}

	result, throttled, err := streamingClient.PeekAndIdentify(ctx, requestID)
	if err != nil {
		if !streamingClient.IsConnected() {
			p.mutex.Lock()
			p.isActive = false
			p.mutex.Unlock()
		}
		log.Warnf("Lấy kết quả speaker recognition trung gian thất bại: %v", err)
		return nil, throttled, err
	}

	return result, throttled, nil
}

// Close Đóng speaker provider
func (p *AsrServerProvider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.isActive = false
	if p.streamingClient != nil {
		return p.streamingClient.Close()
	}
	return nil
}

// IsActive Kiểm tra có đang active không
func (p *AsrServerProvider) IsActive() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.isActive
}
