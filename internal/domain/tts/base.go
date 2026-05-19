package tts

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/tts/cosyvoice"
	"xiaozhi-esp32-server-golang/internal/domain/tts/doubao"
	"xiaozhi-esp32-server-golang/internal/domain/tts/edge"
	"xiaozhi-esp32-server-golang/internal/domain/tts/edge_offline"
	"xiaozhi-esp32-server-golang/internal/domain/tts/minimax"
	"xiaozhi-esp32-server-golang/internal/domain/tts/openai"
	"xiaozhi-esp32-server-golang/internal/domain/tts/piper"
	"xiaozhi-esp32-server-golang/internal/domain/tts/qwen"
	"xiaozhi-esp32-server-golang/internal/domain/tts/streaming"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xiaozhi"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xunfei"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xunfei_super_tts"
	"xiaozhi-esp32-server-golang/internal/domain/tts/zhipu"
)

// Interface provider TTS cơ bản (không gồm method Context)
type BaseTTSProvider interface {
	TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error)
	TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error)
}

// DualStreamProvider interface tùy chọn khi cả input và output TTS đều streaming: vừa nhận text vừa tổng hợp output. Provider hỗ trợ thì triển khai interface này.
type DualStreamProvider interface {
	StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (outputChan chan streaming.SynthesisEvent, err error)
}

// Interface provider TTS đầy đủ (gồm method Context)
type TTSProvider interface {
	BaseTTSProvider
	// SetVoice Thiết lập động tham số voice
	// voiceConfig: map chứa cấu hình liên quan voice, ví dụ {"voice": "xxx"} hoặc {"spk_id": "xxx"}
	SetVoice(voiceConfig map[string]interface{}) error
	// Close Đóng tài nguyên, giải phóng kết nối, v.v.
	Close() error
	// IsValid Kiểm tra tài nguyên có hợp lệ không (kết nối còn sống, v.v.)
	IsValid() bool
}

// GetTTSProvider Lấy provider TTS đầy đủ (hỗ trợ Context)
// providerName: có thể là config_id/provider hoặc resource pool key (ví dụ "edge_tts:zh-CN-XiaoxiaoNeural"）
// config: config map parse từ field json_data trong bảng configs của database
// Ưu tiên dùng field provider trong config, nếu không thì parse từ providerName (lấy phần trước ":")
func GetTTSProvider(providerName string, config map[string]interface{}) (TTSProvider, error) {
	effectiveName := providerName
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		effectiveName = configProvider
	}
	// Resource pool key có dạng "provider:voiceID", lấy nửa đầu làm loại provider
	if idx := strings.Index(effectiveName, ":"); idx > 0 {
		effectiveName = effectiveName[:idx]
	}
	var baseProvider BaseTTSProvider

	switch effectiveName {
	case constants.TtsTypeDoubao:
		baseProvider = doubao.NewDoubaoTTSProvider(config)
	case constants.TtsTypeDoubaoWS:
		baseProvider = doubao.NewDoubaoWSProvider(config)
	case constants.TtsTypeCosyvoice:
		baseProvider = cosyvoice.NewCosyVoiceTTSProvider(config)
	case constants.TtsTypeEdge:
		baseProvider = edge.NewEdgeTTSProvider(config)
	case constants.TtsTypeEdgeOffline:
		baseProvider = edge_offline.NewEdgeOfflineTTSProvider(config)
	case constants.TtsTypeXiaozhi:
		baseProvider = xiaozhi.NewXiaozhiProvider(config)
	case constants.TtsTypeXunfei:
		baseProvider = xunfei.NewXunfeiTTSProvider(config)
	case constants.TtsTypeXunfeiSuper:
		baseProvider = xunfei_super_tts.NewXunfeiSuperTTSProvider(config)
	case constants.TtsTypeOpenAI:
		baseProvider = openai.NewOpenAITTSProvider(config)
	case constants.TtsTypeZhipu:
		baseProvider = zhipu.NewZhipuTTSProvider(config)
	case constants.TtsTypeMinimax:
		baseProvider = minimax.NewMinimaxTTSProvider(config)
	case constants.TtsTypeAliyunQwen:
		baseProvider = qwen.NewQwenTTSProvider(config)
	case constants.TtsTypeIndexTTSVLLM:
		baseProvider = openai.NewOpenAITTSProvider(buildIndexTTSOpenAIConfig(config))
	case constants.TtsTypePiper:
		baseProvider = piper.NewPiperTTSProvider(config)
	default:
		return nil, fmt.Errorf("Provider TTS không được hỗ trợ: %s", effectiveName)
	}

	if baseProvider == nil {
		return nil, fmt.Errorf("Không thể tạo provider TTS: %s", effectiveName)
	}

	// Dùng adapter bọc provider cơ bản để chuyển thành TTSProvider đầy đủ
	provider := &ContextTTSAdapter{baseProvider}

	return provider, nil
}

func buildIndexTTSOpenAIConfig(config map[string]interface{}) map[string]interface{} {
	const (
		defaultIndexTTSURL   = "http://127.0.0.1:7860/audio/speech"
		defaultIndexTTSModel = "indextts-vllm"
	)

	normalized := make(map[string]interface{}, len(config)+4)
	for k, v := range config {
		normalized[k] = v
	}

	apiURL, _ := normalized["api_url"].(string)
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = defaultIndexTTSURL
	} else {
		parsed, err := url.Parse(apiURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			trimmed := strings.TrimRight(apiURL, "/")
			if !strings.HasSuffix(strings.ToLower(trimmed), "/audio/speech") {
				trimmed += "/audio/speech"
			}
			apiURL = trimmed
		} else {
			if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
				parsed.Path = "/audio/speech"
				parsed.RawPath = ""
				apiURL = parsed.String()
			}
		}
	}
	normalized["api_url"] = strings.TrimRight(apiURL, "/")

	if model, _ := normalized["model"].(string); strings.TrimSpace(model) == "" {
		normalized["model"] = defaultIndexTTSModel
	}
	if responseFormat, _ := normalized["response_format"].(string); strings.TrimSpace(responseFormat) == "" {
		normalized["response_format"] = "wav"
	}
	if _, exists := normalized["stream"]; !exists {
		normalized["stream"] = false
	}
	if _, exists := normalized["speed"]; !exists {
		normalized["speed"] = float64(1.0)
	}

	return normalized
}

// ContextTTSAdapter là adapter thêm hỗ trợ Context cho provider TTS cơ bản
type ContextTTSAdapter struct {
	Provider BaseTTSProvider
}

// StreamingSynthesize proxy tới interface tổng hợp dual-streaming của provider gốc
func (a *ContextTTSAdapter) StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (outputChan chan streaming.SynthesisEvent, err error) {
	// Kiểm tra provider bên dưới có hỗ trợ dual-streaming không
	if dsProvider, ok := a.Provider.(DualStreamProvider); ok {
		return dsProvider.StreamingSynthesize(ctx, textChan, sampleRate, channels, frameDuration)
	}
	return nil, fmt.Errorf("Provider bên dưới không hỗ trợ tổng hợp dual-streaming")
}

// TextToSpeech Proxy tới provider gốc
func (a *ContextTTSAdapter) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	return a.Provider.TextToSpeech(ctx, text, sampleRate, channels, frameDuration)
}

// TextToSpeechStream Proxy tới provider gốc
func (a *ContextTTSAdapter) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	return a.Provider.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
}

// SetVoice Proxy tới method SetVoice của provider bên dưới
func (a *ContextTTSAdapter) SetVoice(voiceConfig map[string]interface{}) error {
	// Nếu provider bên dưới triển khai method SetVoice thì gọi trực tiếp
	if setter, ok := a.Provider.(interface {
		SetVoice(map[string]interface{}) error
	}); ok {
		return setter.SetVoice(voiceConfig)
	}
	// Nếu không thì trả về lỗi không hỗ trợ
	return fmt.Errorf("Provider bên dưới không hỗ trợ method SetVoice")
}

// TextToSpeechWithContext Dùng phiên bản text-to-speech có Context
func (a *ContextTTSAdapter) TextToSpeechWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	// Kiểm tra provider có hỗ trợ trực tiếp phiên bản Context không
	if provider, ok := a.Provider.(interface {
		TextToSpeechWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error)
	}); ok {
		// Provider hỗ trợ trực tiếp phiên bản Context
		return provider.TextToSpeechWithContext(ctx, text, sampleRate, channels, frameDuration)
	}

	// Nếu không thì dùng bản chuẩn và điều khiển context bằng goroutine/channel
	resultChan := make(chan struct {
		frames [][]byte
		err    error
	})

	go func() {
		frames, err := a.Provider.TextToSpeech(ctx, text, sampleRate, channels, frameDuration)
		select {
		case <-ctx.Done():
			// Context đã hủy, không gửi kết quả
			return
		case resultChan <- struct {
			frames [][]byte
			err    error
		}{frames, err}:
			// Kết quả đã được gửi
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result.frames, result.err
	}
}

// TextToSpeechStreamWithContext Dùng phiên bản streaming text-to-speech có Context
func (a *ContextTTSAdapter) TextToSpeechStreamWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, cancelFunc func(), err error) {
	// Kiểm tra provider có hỗ trợ trực tiếp phiên bản Context không
	if provider, ok := a.Provider.(interface {
		TextToSpeechStreamWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, func(), error)
	}); ok {
		// Provider hỗ trợ trực tiếp phiên bản Context
		return provider.TextToSpeechStreamWithContext(ctx, text, sampleRate, channels, frameDuration)
	}

	// Nếu không thì dùng bản chuẩn nhưng tạo wrapper để xử lý context cancel
	streamCtx, cancel := context.WithCancel(ctx)
	streamChan, err := a.Provider.TextToSpeechStream(streamCtx, text, sampleRate, channels, frameDuration)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	cancelFunc = cancel

	// Tạo output channel mới để forward và xử lý cancel
	outputChan = make(chan []byte, 10)

	// Tạo goroutine để forward dữ liệu và lắng nghe context cancel
	go func() {
		defer close(outputChan)

		for {
			select {
			case <-streamCtx.Done():
				// Context đã hủy, gọi hàm cancel gốc và thoát
				cancelFunc()
				return
			case frame, ok := <-streamChan:
				if !ok {
					// Channel gốc đã đóng
					return
				}
				// Forward dữ liệu
				select {
				case <-streamCtx.Done():
					// Context đã hủy
					cancelFunc()
					return
				case outputChan <- frame:
					// thành côngForward dữ liệu
				}
			}
		}
	}()

	return outputChan, cancelFunc, nil
}

// Close đóngtài nguyên
func (a *ContextTTSAdapter) Close() error {
	// Nếu provider bên dưới triển khai method Close thì gọi trực tiếp
	if closer, ok := a.Provider.(interface {
		Close() error
	}); ok {
		return closer.Close()
	}
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (a *ContextTTSAdapter) IsValid() bool {
	// Nếu provider bên dưới triển khai method IsValid thì gọi trực tiếp
	if validator, ok := a.Provider.(interface {
		IsValid() bool
	}); ok {
		return validator.IsValid()
	}
	// Nếu không thì kiểm tra provider có nil không
	return a.Provider != nil
}
