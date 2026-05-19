package zhipu

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gopxl/beep"
	sse "github.com/tmaxmax/go-sse"
)

// HTTP client global, triển khai connection pool
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

const (
	zhipuDefaultSampleRate = 24000
	zhipuLeadingFadeInMs   = 5
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
			Timeout:   60 * time.Second,
		}
	})
	return httpClient
}

// ZhipuTTSProvider noi_dung provider TTS
type ZhipuTTSProvider struct {
	APIKey         string
	APIURL         string
	Model          string
	Voice          string
	ResponseFormat string
	Speed          float64
	Volume         float64
	Stream         bool
	EncodeFormat   string // noi_dungstreamingnoi_dungdùng：base64 hoặc hex
	FrameDuration  int
}

// requeststruct（noi_dung API noi_dung）
type zhipuRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
	Volume         float64 `json:"volume,omitempty"`
	Stream         bool    `json:"stream,omitempty"`
	EncodeFormat   string  `json:"encode_format,omitempty"` // noi_dungstreamingnoi_dungdùng：base64 hoặc hex
}

// Event Stream responsestruct（noi_dung OpenAI format）
type zhipuEventStreamResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason,omitempty"`
		Delta        struct {
			Role             string `json:"role,omitempty"`
			Content          string `json:"content,omitempty"` // base64 noi_dungaudiodata
			ReturnSampleRate int    `json:"return_sample_rate,omitempty"`
			ReturnFormat     string `json:"return_format,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

// NewZhipuTTSProvider Tạo mớinoi_dung provider TTS
func NewZhipuTTSProvider(config map[string]interface{}) *ZhipuTTSProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	responseFormat, _ := config["response_format"].(string)
	speed, _ := config["speed"].(float64)
	volume, _ := config["volume"].(float64)
	stream, _ := config["stream"].(bool)
	encodeFormat, _ := config["encode_format"].(string)
	frameDuration, _ := config["frame_duration"].(float64)

	// Thiết lập giá trị mặc định
	if apiURL == "" {
		apiURL = "https://open.bigmodel.cn/api/paas/v4/audio/speech"
	}
	if model == "" {
		model = "glm-tts"
	}
	if voice == "" {
		voice = "tongtong" // mặc địnhvoice
	}
	if responseFormat == "" {
		responseFormat = "pcm" // noi_dungmặc định pcm，noi_dunghỗ trợ wav
	}
	if speed == 0 {
		speed = 1.0 // 0.5 tới 2.0
	}
	if volume == 0 {
		volume = 1.0 // 0 tới 10
	}
	if encodeFormat == "" {
		encodeFormat = "base64" // mặc định base64，noi_dunghỗ trợ hex
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &ZhipuTTSProvider{
		APIKey:         apiKey,
		APIURL:         apiURL,
		Model:          model,
		Voice:          voice,
		ResponseFormat: responseFormat,
		Stream:         stream,
		Speed:          speed,
		Volume:         volume,
		EncodeFormat:   encodeFormat,
		FrameDuration:  int(frameDuration),
	}
}

// TextToSpeech Chuyển text thành giọng nói, trả về dữ liệu frame audiovàlỗi
func (p *ZhipuTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	startTs := time.Now().UnixMilli()

	// noi_dungtextđộ dài（noi_dung API noi_dung 1024 noi_dung）
	if len(text) > 1024 {
		text = text[:1024]
		log.Warnf("textđộ dàinoi_dung1024noi_dung，đãnoi_dung")
	}

	// tạorequestnoi_dung
	reqBody := zhipuRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: p.ResponseFormat,
		Speed:          p.Speed,
		Volume:         p.Volume,
		Stream:         false, // noi_dungstreaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("sequencenoi_dungrequest thất bại: %v", err)
	}

	// tạoHTTPrequest
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("tạorequest thất bại: %v", err)
	}

	// noi_dungrequestnoi_dung
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	// dùngkết nốinoi_dunggửirequest
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gửirequest thất bại: %v", err)
	}
	defer resp.Body.Close()

	// kiểm traresponsetrạng tháinoi_dung
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("APIrequest thất bại，trạng tháinoi_dung: %d, response: %s", resp.StatusCode, string(body))
	}

	// kiểm traresponsenội dungđộ dài
	contentLength := resp.ContentLength
	log.Debugf("Nhậnnoi_dung TTSresponse，Content-Length: %d", contentLength)

	// Kiểm traContent-Lengthcó hợp lý không
	if contentLength == 0 {
		log.Errorf("APItrả vềrỗngresponse，Content-Lengthlà0")
		return nil, fmt.Errorf("APItrả vềrỗngresponse，Content-Lengthlà0")
	}

	// noi_dungaudioformatxử lýresponse（noi_dunghỗ trợ wav và pcm）
	if p.ResponseFormat == "wav" || p.ResponseFormat == "pcm" {
		audioReader := io.ReadCloser(resp.Body)
		if strings.EqualFold(p.ResponseFormat, "pcm") {
			pcmData, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("đọcnoi_dung PCM datathất bại: %v", err)
			}
			audioReader = io.NopCloser(bytes.NewReader(
				applyPCM16MonoLeadingFadeIn(pcmData, leadingFadeInSampleCount(zhipuDefaultSampleRate, zhipuLeadingFadeInMs)),
			))
		}

		// tạomộtchannelnoi_dungaudioframe
		outputChan := make(chan []byte, 1000)

		// tạoaudiodecoder
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, audioReader, outputChan, frameDuration, p.ResponseFormat, sampleRate)
		if err != nil {
			return nil, fmt.Errorf("Tạo audio decoder thất bại: %v", err)
		}
		if strings.EqualFold(p.ResponseFormat, "pcm") {
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(zhipuDefaultSampleRate),
				NumChannels: 1,
			})
		}

		// Khởi độngquá trình decode
		go func() {
			if err := decoder.Run(startTs); err != nil {
				log.Errorf("Decode audio thất bại: %v", err)
			}
		}()

		// Thu thập toàn bộaudioframe
		var audioFrames [][]byte
		for frame := range outputChan {
			audioFrames = append(audioFrames, frame)
		}

		log.Debugf("noi_dung TTShoàn tất，noi_dunginputtớilấyaudiodatanoi_dung: %d ms", time.Now().UnixMilli()-startTs)
		return audioFrames, nil
	}

	return nil, fmt.Errorf("Không hỗ trợaudioformat: %s，noi_dunghỗ trợ wav và pcm", p.ResponseFormat)
}

// TextToSpeechStream Tổng hợp giọng nói streamingtriển khai
func (p *ZhipuTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	// noi_dungtextđộ dài（noi_dung API noi_dung 1024 noi_dung）
	if len(text) > 1024 {
		text = text[:1024]
		log.Warnf("textđộ dàinoi_dung1024noi_dung，đãnoi_dung")
	}

	// streamingnoi_dunghỗ trợ pcmvàwav format
	responseFormat := p.ResponseFormat

	// tạorequestnoi_dung
	reqBody := zhipuRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: responseFormat,
		Speed:          p.Speed,
		Volume:         p.Volume,
		Stream:         true,           // streaming
		EncodeFormat:   p.EncodeFormat, // dùngconfignoi_dungformat
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("sequencenoi_dungrequest thất bại: %v", err)
	}

	// tạoHTTPrequest
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("tạorequest thất bại: %v", err)
	}

	// noi_dungrequestnoi_dung
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	// dùngkết nốinoi_dungtạoclient
	client := getHTTPClient()

	// tạooutputchannel
	outputChan = make(chan []byte, 100)

	// Khởi độnggoroutinexử lýstreamingresponse
	go func() {
		// gửirequest
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("gửinoi_dungrequest thất bại: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		// kiểm traresponsetrạng tháinoi_dung
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("noi_dung APIrequest thất bại，trạng tháinoi_dung: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		// kiểm tra Content-Type noi_dunglà Event Stream
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			log.Warnf("noi_dung APItrả vềnoi_dungContent-Typenoi_dungtext/event-stream: %s", contentType)
		}

		// streamingnoi_dunghỗ trợ pcm và wav format
		//log.Debugf("noi_dung TTS streaming responseFormat(request): %s", responseFormat)
		if responseFormat == "pcm" || responseFormat == "wav" {
			// tạopipe，dùng đểnoi_dungsaunoi_dungdatanoi_dungaudiodecoder
			pipeReader, pipeWriter := io.Pipe()

			// Khởi động goroutine parse Event Stream noi_dung
			go func() {
				defer func() {
					if err := pipeWriter.Close(); err != nil {
						log.Debugf("đóngpipeghinoi_dungthất bại: %v", err)
					}
				}()

				// noi_dungparsemethod
				if err := p.parseEventStream(ctx, resp.Body, pipeWriter, text); err != nil {
					log.Errorf("parse Event Stream thất bại: %v", err)
				}
			}()

			// tạoaudiodecoder，noi_dungpipeđọcnoi_dungsaunoi_dungdata
			decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, responseFormat, sampleRate)
			if err != nil {
				log.Errorf("tạonoi_dungaudiodecoderthất bại: %v", err)
				pipeReader.Close()
				close(outputChan)
				return
			}
			if strings.EqualFold(responseFormat, "pcm") {
				decoder.WithFormat(beep.Format{
					SampleRate:  beep.SampleRate(zhipuDefaultSampleRate),
					NumChannels: 1,
				})
			}

			// Khởi độngquá trình decode
			if err := decoder.Run(startTs); err != nil {
				log.Errorf("noi_dungDecode audio thất bại: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("noi_dung TTSstreamingtổng hợphủy, text: %s", text)
				return
			default:
				log.Debugf("noi_dung TTSnoi_dung: noi_dunginputnoi_dunglấyaudiodatanoi_dung: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("noi_dungstreamingoutputnoi_dunghỗ trợ pcm format")
			close(outputChan)
		}
	}()

	return outputChan, nil
}

// parseEventStream dùng go-sse parsenoi_dung Event Stream response，noi_dungdatanoi_dungghipipe
// ctx: noi_dung，dùng đểhủynoi_dung
// reader: responsenoi_dungđọcnoi_dung
// writer: pipeghinoi_dung，dùng đểoutputnoi_dungsaunoi_dungdata
// text: gốctext，dùng đểlogGhi
func (p *ZhipuTTSProvider) parseEventStream(ctx context.Context, reader io.Reader, writer *io.PipeWriter, text string) error {
	// config go-sse noi_dung ReadConfig，noi_dung MaxEventSize noi_dungxử lýnoi_dung token
	// noi_dung TTS trả vềnoi_dung base64 noi_dungaudiodatanoi_dungmặc địnhnoi_dung 64KB noi_dung
	readConfig := &sse.ReadConfig{
		MaxEventSize: 4 * 1024 * 1024, // 4MB，noi_dungxử lýnoi_dung base64 noi_dungaudiodata
	}
	fadeTotalSamples := 0
	fadeSamplesRemaining := -1

	for ev, evErr := range sse.Read(reader, readConfig) {
		if evErr != nil {
			return fmt.Errorf("đọcnoi_dung SSE eventthất bại: %w", evErr)
		}

		select {
		case <-ctx.Done():
			log.Debugf("noi_dung TTSstreamingtổng hợphủy, text: %s", text)
			return ctx.Err()
		default:
		}

		// Event Stream format：
		// data: {"id":"...","choices":[{"delta":{"content":"base64_data"}}]}
		// data: {"choices":[{"finish_reason":"stop"}]}

		dataValue := strings.TrimSpace(ev.Data)
		if dataValue == "" {
			continue
		}

		// parse JSON
		var eventResp zhipuEventStreamResponse
		if err := json.Unmarshal([]byte(dataValue), &eventResp); err != nil {
			log.Warnf("parsenoi_dung Event Stream JSON thất bại: %v, data: %s", err, previewString(dataValue, 200))
			continue
		}

		// kiểm tranoi_dung finish_reason，noi_dung
		for _, choice := range eventResp.Choices {
			if choice.FinishReason == "stop" {
				log.Debugf("Nhận finish_reason: stop，Event Stream noi_dung")
				return nil
			}
		}

		// noi_dunglấynoi_dung choice noi_dung content fieldnoi_dungxử lý
		for _, choice := range eventResp.Choices {
			if choice.Delta.Content != "" {
				decodedData, err := p.decodeAudioContent(choice.Delta.Content)
				if err != nil {
					return fmt.Errorf("xử lý content thất bại: %v", err)
				}

				returnFormat := strings.TrimSpace(choice.Delta.ReturnFormat)
				if returnFormat == "" {
					returnFormat = p.ResponseFormat
				}
				if strings.EqualFold(returnFormat, "pcm") {
					if fadeSamplesRemaining < 0 {
						sampleRate := choice.Delta.ReturnSampleRate
						if sampleRate < 1 {
							sampleRate = zhipuDefaultSampleRate
						}
						fadeTotalSamples = leadingFadeInSampleCount(sampleRate, zhipuLeadingFadeInMs)
						fadeSamplesRemaining = fadeTotalSamples
					}
					applyPCM16MonoLeadingFadeInInPlace(decodedData, fadeTotalSamples, &fadeSamplesRemaining)
				}

				if len(decodedData) > 0 {
					if _, err := writer.Write(decodedData); err != nil {
						return fmt.Errorf("ghipipethất bại: %v", err)
					}
				}
			}
		}
	}

	return nil
}

// previewString trả vềnoi_dungtrước n noi_dungdùng đểlog
func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// decodeAudioContent noi_dung content field
// content: base64 hoặc hex noi_dungaudiodatanoi_dung
func (p *ZhipuTTSProvider) decodeAudioContent(content string) ([]byte, error) {
	if content == "" {
		return nil, nil
	}

	// noi_dung encode_format noi_dung
	var decodedData []byte
	var decodeErr error

	switch p.EncodeFormat {
	case "base64":
		decodedData, decodeErr = base64.StdEncoding.DecodeString(content)
	case "hex":
		decodedData, decodeErr = hex.DecodeString(content)
	default:
		log.Warnf("không xác địnhnoi_dungformat: %s，dùng base64", p.EncodeFormat)
		decodedData, decodeErr = base64.StdEncoding.DecodeString(content)
	}

	if decodeErr != nil {
		return nil, fmt.Errorf("noi_dungaudiodatathất bại: %v, datađộ dài: %d", decodeErr, len(content))
	}

	return decodedData, nil
}

func leadingFadeInSampleCount(sampleRate int, fadeMs int) int {
	if sampleRate < 1 {
		sampleRate = zhipuDefaultSampleRate
	}
	if fadeMs < 1 {
		return 0
	}
	samples := sampleRate * fadeMs / 1000
	if samples < 1 {
		return 1
	}
	return samples
}

func applyPCM16MonoLeadingFadeIn(data []byte, remainingSamples int) []byte {
	if len(data) == 0 || remainingSamples <= 0 {
		return data
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	applyPCM16MonoLeadingFadeInInPlace(cloned, remainingSamples, &remainingSamples)
	return cloned
}

func applyPCM16MonoLeadingFadeInInPlace(data []byte, totalSamples int, remainingSamples *int) {
	if len(data) < 2 || totalSamples <= 0 || remainingSamples == nil || *remainingSamples <= 0 {
		return
	}

	samplePairs := len(data) / 2
	for i := 0; i < samplePairs && *remainingSamples > 0; i++ {
		offset := i * 2
		sample := int16(uint16(data[offset]) | uint16(data[offset+1])<<8)
		appliedIndex := totalSamples - *remainingSamples
		scaled := int32(sample) * int32(appliedIndex) / int32(totalSamples)
		binarySample := uint16(int16(scaled))
		data[offset] = byte(binarySample)
		data[offset+1] = byte(binarySample >> 8)
		*remainingSamples = *remainingSamples - 1
	}
}

// SetVoice Thiết lập tham số voice
func (p *ZhipuTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("noi_dungvoiceconfig: noi_dung voice")
}

// Close đóngtài nguyên（noi_dungtrạng thái Provider，noi_dungđóng）
func (p *ZhipuTTSProvider) Close() error {
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *ZhipuTTSProvider) IsValid() bool {
	return p != nil
}
