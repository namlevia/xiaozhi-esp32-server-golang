package cosyvoice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

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
			Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// CosyVoiceTTSProvider CosyVoice provider TTS
type CosyVoiceTTSProvider struct {
	APIURL        string
	SpeakerID     string
	FrameDuration int
	TargetSR      int
	AudioFormat   string
	InstructText  string
}

// responsestruct
type cosyVoiceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

// NewCosyVoiceTTSProvider Tạo mớiCosyVoice provider TTS
func NewCosyVoiceTTSProvider(config map[string]interface{}) *CosyVoiceTTSProvider {
	apiURL, _ := config["api_url"].(string)
	speakerID, _ := config["spk_id"].(string)
	frameDuration, _ := config["frame_duration"].(float64)
	targetSR, _ := config["target_sr"].(float64)
	audioFormat, _ := config["audio_format"].(string)
	instructText, _ := config["instruct_text"].(string)

	// Thiết lập giá trị mặc định
	if apiURL == "" {
		apiURL = "https://tts.linkerai.cn/tts"
	}
	if speakerID == "" {
		speakerID = "OUeAo1mhq6IBExi"
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}
	if targetSR == 0 {
		targetSR = audio.SampleRate
	}
	if audioFormat == "" {
		audioFormat = "mp3"
	}

	return &CosyVoiceTTSProvider{
		APIURL:        apiURL,
		SpeakerID:     speakerID,
		FrameDuration: int(frameDuration),
		TargetSR:      int(targetSR),
		AudioFormat:   audioFormat,
		InstructText:  instructText,
	}
}

// TextToSpeech Chuyển text thành giọng nói, trả về dữ liệu frame audiovàlỗi
func (p *CosyVoiceTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	// Dựngtham số query
	params := url.Values{}
	params.Add("tts_text", text)
	params.Add("spk_id", p.SpeakerID)
	params.Add("frame_durition", fmt.Sprintf("%d", p.FrameDuration))
	params.Add("stream", "true") // streamingrequest
	params.Add("target_sr", fmt.Sprintf("%d", p.TargetSR))
	params.Add("audio_format", p.AudioFormat)

	startTs := time.Now().UnixMilli()

	// DựngURL đầy đủ
	requestURL := fmt.Sprintf("%s?%s", p.APIURL, params.Encode())

	// tạoHTTPrequest
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tạorequest thất bại: %v", err)
	}

	req.Header.Set("Accept", "application/json")

	// dùngkết nốinoi_dunggửirequest
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gửirequest thất bại: %v", err)
	}
	defer resp.Body.Close()

	// đọcresponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Đọc response thất bại: %v", err)
	}

	// kiểm traresponsetrạng tháinoi_dung
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("APIrequest thất bại，trạng tháinoi_dung: %d, response: %s", resp.StatusCode, string(body))
	}

	// kiểm traresponsecontent typevànội dungđộ dài
	// contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	// Ghiresponseđộ dàitớilog
	log.Debugf("NhậnTTSresponse，Content-Length: %d", contentLength)

	// Kiểm traContent-Lengthcó hợp lý không
	if contentLength == 0 {
		log.Errorf("APItrả vềrỗngresponse，Content-Lengthlà0")
		return nil, fmt.Errorf("APItrả vềrỗngresponse，Content-Lengthlà0")
	}

	// MP3header filecần ít nhất100byteđể có thểparse
	// -1noi_dungkhông xác địnhđộ dài（ví dụchunked transfer）
	if contentLength > 0 && contentLength < 100 {
		log.Errorf("APItrả vềnoi_dungresponsequá nhỏ, không thểparselàMP3: %dbyte", contentLength)
		return nil, fmt.Errorf("APItrả vềnoi_dungresponsequá nhỏ, không thểparselàMP3: %dbyte", contentLength)
	}

	// chuyểnlàOpusframe
	if p.AudioFormat == "mp3" {
		// tạomộtpipe
		doneChan := make(chan struct{})
		outputChan := make(chan []byte, 1000)

		// tạoMP3decoder
		mp3Decoder, err := util.CreateAudioDecoder(ctx, io.NopCloser(bytes.NewReader(body)), outputChan, frameDuration, p.AudioFormat)
		if err != nil {
			close(doneChan)
			return nil, fmt.Errorf("tạoMP3decoderthất bại: %v", err)
		}
		// Khởi độngquá trình decode
		go func() {
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3noi_dungthất bại: %v", err)
			}
		}()

		// Thu thập toàn bộOpusframe
		var opusFrames [][]byte
		for frame := range outputChan {
			opusFrames = append(opusFrames, frame)
		}

		return opusFrames, nil
	}

	return nil, fmt.Errorf("Không hỗ trợaudioformat: %s", p.AudioFormat)
}

// TextToSpeechStream Tổng hợp giọng nói streamingtriển khai
func (p *CosyVoiceTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	// Dựngtham số query
	params := url.Values{}
	params.Add("tts_text", text)
	params.Add("spk_id", p.SpeakerID)
	params.Add("frame_durition", fmt.Sprintf("%d", frameDuration))
	params.Add("stream", "true") // streamingrequest
	params.Add("target_sr", fmt.Sprintf("%d", sampleRate))
	params.Add("audio_format", p.AudioFormat)

	startTs := time.Now().UnixMilli()

	// DựngURL đầy đủ
	requestURL := fmt.Sprintf("%s?%s", p.APIURL, params.Encode())

	// tạoHTTPrequest
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tạorequest thất bại: %v", err)
	}

	req.Header.Set("Accept", "application/json")

	// dùngkết nốinoi_dungtạoclient
	client := getHTTPClient()

	// tạooutputchannel
	outputChan = make(chan []byte, 100)
	// Khởi độnggoroutinexử lýstreamingresponse
	go func() {
		decoderStarted := false
		defer func() {
			if !decoderStarted {
				close(outputChan)
			}
		}()

		// gửirequest
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("gửirequest thất bại: %v", err)
			return
		}
		defer func() {
			resp.Body.Close()
		}()

		// kiểm traresponsetrạng tháinoi_dung
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("APIrequest thất bại，trạng tháinoi_dung: %d, response: %s", resp.StatusCode, string(body))
			return
		}

		// kiểm traresponsecontent typevànội dungđộ dài
		// contentType := resp.Header.Get("Content-Type")
		contentLength := resp.ContentLength

		// Ghiresponseđộ dàitớilog
		log.Debugf("NhậnTTSresponse，Content-Length: %d", contentLength)

		// Kiểm traContent-Lengthcó hợp lý không
		if contentLength == 0 {
			log.Errorf("APItrả vềrỗngresponse，Content-Lengthlà0")
			return
		}

		// MP3header filecần ít nhất100byteđể có thểparse
		// -1noi_dungkhông xác địnhđộ dài（ví dụchunked transfer）
		if contentLength > 0 && contentLength < 100 {
			log.Errorf("APItrả vềnoi_dungresponsequá nhỏ, không thểparselàMP3: %dbyte", contentLength)
			return
		}

		// noi_dungaudioformatxử lýstreamingresponse
		if p.AudioFormat == "mp3" {
			// tạo MP3 decoder，noi_dung context noi_dung done channel
			mp3Decoder, err := util.CreateAudioDecoder(ctx, resp.Body, outputChan, frameDuration, p.AudioFormat)
			if err != nil {
				log.Errorf("tạoMP3decoderthất bại: %v", err)
				return
			}

			// Khởi độngquá trình decode
			decoderStarted = true
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3noi_dungthất bại: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("TTSstreamingtổng hợphủy, text: %s", text)
				return
			default:
				log.Infof("ttsnoi_dung: noi_dunginputnoi_dunglấyMP3datanoi_dung: %d ms", time.Now().UnixMilli()-startTs)

			}
		} else {
			log.Errorf("hiện tạinoi_dunghỗ trợMP3formatnoi_dungstreamingtổng hợp")
		}
	}()

	return outputChan, nil
}

// SetVoice Thiết lập tham số voice
func (p *CosyVoiceTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if spkID, ok := voiceConfig["spk_id"].(string); ok && spkID != "" {
		p.SpeakerID = spkID
		return nil
	}
	return fmt.Errorf("noi_dungvoiceconfig: noi_dung spk_id")
}

// Close đóngtài nguyên（noi_dungtrạng thái Provider，noi_dungđóng）
func (p *CosyVoiceTTSProvider) Close() error {
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *CosyVoiceTTSProvider) IsValid() bool {
	return p != nil
}
