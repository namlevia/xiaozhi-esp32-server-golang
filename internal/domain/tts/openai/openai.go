package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

// HTTP client global, triển khai connection pool
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// Lấy HTTP client đã cấu hình connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second, // OpenAI TTS noi_dungthời gian
		}
	})
	return httpClient
}

// OpenAITTSProvider OpenAI provider TTS
type OpenAITTSProvider struct {
	APIKey         string
	APIURL         string
	Model          string
	Voice          string
	ResponseFormat string
	Speed          float64
	Stream         bool
	FrameDuration  int
}

// requeststruct
type openAIRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
	Stream         bool    `json:"stream,omitempty"`
}

// NewOpenAITTSProvider Tạo mớiOpenAI provider TTS
func NewOpenAITTSProvider(config map[string]interface{}) *OpenAITTSProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	responseFormat, _ := config["response_format"].(string)
	speed, _ := config["speed"].(float64)
	stream, _ := config["stream"].(bool)
	frameDuration, _ := config["frame_duration"].(float64)

	// Thiết lập giá trị mặc định
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/audio/speech"
	}
	if model == "" {
		model = "tts-1" // tts-1 hoặc tts-1-hd
	}
	if voice == "" {
		voice = "alloy" // alloy, echo, fable, onyx, nova, shimmer
	}
	if responseFormat == "" {
		responseFormat = "mp3" // mp3, opus, aac, flac, wav, pcm
	}
	if speed == 0 {
		speed = 1.0 // 0.25 tới 4.0
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &OpenAITTSProvider{
		APIKey:         apiKey,
		APIURL:         apiURL,
		Model:          model,
		Voice:          voice,
		ResponseFormat: responseFormat,
		Stream:         stream,
		Speed:          speed,
		FrameDuration:  int(frameDuration),
	}
}

// TextToSpeech Chuyển text thành giọng nói, trả về dữ liệu frame audiovàlỗi
func (p *OpenAITTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	streamChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	audioFrames := make([][]byte, 0, 32)
	for frame := range streamChan {
		audioFrames = append(audioFrames, frame)
	}
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("OpenAI TTS trả vềaudiolàrỗng")
	}
	return audioFrames, nil
}

// TextToSpeechStream Tổng hợp giọng nói streamingtriển khai
func (p *OpenAITTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	// tạorequestnoi_dung
	reqBody := openAIRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: p.ResponseFormat,
		Speed:          p.Speed,
		Stream:         p.Stream,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("sequencenoi_dungrequest thất bại: %v", err)
	}

	//log.Debugf("OpenAI TTSrequest: %s", string(jsonData))

	// tạoHTTPrequest
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("tạorequest thất bại: %v", err)
	}

	// noi_dungrequestnoi_dung
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	}

	// dùngkết nốinoi_dungtạoclient
	client := getHTTPClient()

	// tạooutputchannel
	outputChan = make(chan []byte, 100)

	// Khởi độnggoroutinexử lýstreamingresponse
	go func() {
		// gửirequest
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("gửiOpenAIrequest thất bại: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		// kiểm traresponsetrạng tháinoi_dung
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("OpenAI APIrequest thất bại，trạng tháinoi_dung: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		// kiểm traresponsenội dungđộ dài
		contentLength := resp.ContentLength
		log.Debugf("NhậnOpenAI TTSresponse，Content-Length: %d", contentLength)

		// Kiểm traContent-Lengthcó hợp lý không
		if contentLength == 0 {
			log.Errorf("OpenAI APItrả vềrỗngresponse，Content-Lengthlà0")
			close(outputChan)
			return
		}

		responseFormat := strings.ToLower(strings.TrimSpace(p.ResponseFormat))
		decoderFormat := responseFormat
		if responseFormat == "opus" {
			decoderFormat = "ogg_opus"
			contentTypeFormat := util.GetAudioFormatByMimeType(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type"))))
			if contentTypeFormat == "ogg_opus" || contentTypeFormat == "opus" {
				decoderFormat = contentTypeFormat
			}
		}

		if decoderFormat != "mp3" && decoderFormat != "wav" && decoderFormat != "pcm" && decoderFormat != "opus" && decoderFormat != "ogg_opus" {
			log.Errorf("hiện tạinoi_dunghỗ trợ mp3/wav/pcm/opus/ogg_opus formatnoi_dungstreamingtổng hợp")
			close(outputChan)
			return
		}

		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, resp.Body, outputChan, frameDuration, decoderFormat, sampleRate)
		if err != nil {
			log.Errorf("tạoOpenAIaudiodecoderthất bại: %v", err)
			close(outputChan)
			return
		}
		if decoderFormat == "opus" {
			sourceChannels := channels
			if sourceChannels < 1 {
				sourceChannels = 1
			}
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(util.NormalizeOpusSampleRate(sampleRate)),
				NumChannels: sourceChannels,
			})
		}

		if err := decoder.Run(startTs); err != nil {
			log.Errorf("OpenAIDecode audio thất bại: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			log.Debugf("OpenAI TTSstreamingtổng hợphủy, text: %s", text)
			return
		default:
			log.Infof("OpenAI TTSnoi_dung: noi_dunginputnoi_dunglấyaudiodatanoi_dung: %d ms", time.Now().UnixMilli()-startTs)
		}
	}()

	return outputChan, nil
}

// SetVoice Thiết lập tham số voice
func (p *OpenAITTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("noi_dungvoiceconfig: noi_dung voice")
}

// Close đóngtài nguyên（noi_dungtrạng thái Provider，noi_dungđóng）
func (p *OpenAITTSProvider) Close() error {
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *OpenAITTSProvider) IsValid() bool {
	return p != nil
}
