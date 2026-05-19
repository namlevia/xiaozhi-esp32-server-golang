package piper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

type PiperTTSProvider struct {
	APIURL          string
	Voice           string
	ModelPath       string
	ModelConfigPath string
	ResponseFormat  string
	SampleRate      int
	FrameDuration   int
	Timeout         time.Duration
	LengthScale     float64
	NoiseScale      float64
	NoiseW          float64
}

type piperRequest struct {
	Text            string   `json:"text"`
	Voice           string   `json:"voice,omitempty"`
	ModelPath       string   `json:"model_path,omitempty"`
	ModelConfigPath string   `json:"model_config_path,omitempty"`
	ResponseFormat  string   `json:"response_format,omitempty"`
	SampleRate      int      `json:"sample_rate,omitempty"`
	LengthScale     *float64 `json:"length_scale,omitempty"`
	NoiseScale      *float64 `json:"noise_scale,omitempty"`
	NoiseW          *float64 `json:"noise_w,omitempty"`
}

func NewPiperTTSProvider(config map[string]interface{}) *PiperTTSProvider {
	apiURL, _ := config["api_url"].(string)
	voice, _ := config["voice"].(string)
	modelPath, _ := config["model_path"].(string)
	modelConfigPath, _ := config["model_config_path"].(string)
	responseFormat, _ := config["response_format"].(string)

	if apiURL == "" {
		apiURL = "http://piper-tts:9002/tts"
	}
	apiURL = normalizeAPIURL(apiURL)
	if voice == "" {
		voice = "banmai"
	}
	if responseFormat == "" {
		responseFormat = "wav"
	}

	sampleRate := intFromConfig(config, "sample_rate", 22050)
	frameDuration := intFromConfig(config, "frame_duration", audio.FrameDuration)
	timeoutSeconds := intFromConfig(config, "timeout", 60)

	return &PiperTTSProvider{
		APIURL:          apiURL,
		Voice:           voice,
		ModelPath:       modelPath,
		ModelConfigPath: modelConfigPath,
		ResponseFormat:  strings.ToLower(strings.TrimSpace(responseFormat)),
		SampleRate:      sampleRate,
		FrameDuration:   frameDuration,
		Timeout:         time.Duration(timeoutSeconds) * time.Second,
		LengthScale:     floatFromConfig(config, "length_scale", 1.0),
		NoiseScale:      floatFromConfig(config, "noise_scale", 0.667),
		NoiseW:          floatFromConfig(config, "noise_w", 0.8),
	}
}

func (p *PiperTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	streamChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	frames := make([][]byte, 0, 32)
	for frame := range streamChan {
		frames = append(frames, frame)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("Piper TTS trả về audio rỗng")
	}
	return frames, nil
}

func (p *PiperTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	if p == nil {
		return nil, fmt.Errorf("Piper TTS provider chưa khởi tạo")
	}
	if strings.TrimSpace(p.APIURL) == "" {
		return nil, fmt.Errorf("Piper TTS thiếu api_url")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("Piper TTS thiếu văn bản")
	}

	requestSampleRate := sampleRate
	if requestSampleRate <= 0 {
		requestSampleRate = p.SampleRate
	}
	requestFrameDuration := frameDuration
	if requestFrameDuration <= 0 {
		requestFrameDuration = p.FrameDuration
	}

	lengthScale := p.LengthScale
	noiseScale := p.NoiseScale
	noiseW := p.NoiseW
	reqBody := piperRequest{
		Text:            text,
		Voice:           p.Voice,
		ModelPath:       p.ModelPath,
		ModelConfigPath: p.ModelConfigPath,
		ResponseFormat:  p.ResponseFormat,
		SampleRate:      requestSampleRate,
		LengthScale:     &lengthScale,
		NoiseScale:      &noiseScale,
		NoiseW:          &noiseW,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tạo request Piper TTS thất bại: %v", err)
	}

	requestCtx := ctx
	var cancel context.CancelFunc
	if p.Timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, p.Timeout)
	}

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("tạo request Piper TTS thất bại: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	outputChan := make(chan []byte, 100)
	startTs := time.Now().UnixMilli()

	go func() {
		defer func() {
			if cancel != nil {
				cancel()
			}
		}()

		client := &http.Client{Timeout: p.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("gửi Piper TTS request thất bại: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			log.Errorf("Piper TTS request thất bại, trạng thái: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		decoderFormat := p.decoderFormat(resp.Header.Get("Content-Type"))
		if decoderFormat != "wav" && decoderFormat != "pcm" && decoderFormat != "mp3" {
			log.Errorf("Piper TTS chỉ hỗ trợ wav/pcm/mp3, nhận format: %s", decoderFormat)
			close(outputChan)
			return
		}

		decoder, err := util.CreateAudioDecoderWithSampleRate(requestCtx, resp.Body, outputChan, requestFrameDuration, decoderFormat, requestSampleRate)
		if err != nil {
			log.Errorf("tạo Piper TTS audio decoder thất bại: %v", err)
			close(outputChan)
			return
		}
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("decode Piper TTS audio thất bại: %v", err)
			return
		}
		log.Infof("Piper TTS tổng hợp audio: %d ms", time.Now().UnixMilli()-startTs)
	}()

	return outputChan, nil
}

func (p *PiperTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
		return nil
	}
	if modelPath, ok := voiceConfig["model_path"].(string); ok && strings.TrimSpace(modelPath) != "" {
		p.ModelPath = strings.TrimSpace(modelPath)
		if modelConfigPath, ok := voiceConfig["model_config_path"].(string); ok {
			p.ModelConfigPath = strings.TrimSpace(modelConfigPath)
		}
		return nil
	}
	return fmt.Errorf("Piper TTS thiếu voice hoặc model_path")
}

func (p *PiperTTSProvider) Close() error {
	return nil
}

func (p *PiperTTSProvider) IsValid() bool {
	return p != nil && strings.TrimSpace(p.APIURL) != ""
}

func (p *PiperTTSProvider) decoderFormat(contentType string) string {
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch mimeType {
	case "audio/mpeg", "audio/mp3", "audio/mpeg3", "audio/x-mpeg-3":
		return "mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "audio/pcm", "audio/x-pcm", "audio/l16":
		return "pcm"
	}
	format := strings.ToLower(strings.TrimSpace(p.ResponseFormat))
	if format == "" {
		return "wav"
	}
	return format
}

func normalizeAPIURL(apiURL string) string {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return apiURL
	}
	trimmed := strings.TrimRight(apiURL, "/")
	if strings.HasSuffix(strings.ToLower(trimmed), "/tts") {
		return trimmed
	}
	return trimmed + "/tts"
}

func intFromConfig(config map[string]interface{}, key string, fallback int) int {
	switch value := config[key].(type) {
	case int:
		if value != 0 {
			return value
		}
	case int64:
		if value != 0 {
			return int(value)
		}
	case float64:
		if value != 0 {
			return int(value)
		}
	case float32:
		if value != 0 {
			return int(value)
		}
	}
	return fallback
}

func floatFromConfig(config map[string]interface{}, key string, fallback float64) float64 {
	switch value := config[key].(type) {
	case float64:
		if value != 0 {
			return value
		}
	case float32:
		if value != 0 {
			return float64(value)
		}
	case int:
		if value != 0 {
			return float64(value)
		}
	case int64:
		if value != 0 {
			return float64(value)
		}
	}
	return fallback
}
