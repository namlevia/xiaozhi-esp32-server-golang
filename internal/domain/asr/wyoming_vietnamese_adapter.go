package asr

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
)

type WyomingVietnameseAdapter struct {
	baseURL    string
	timeout    time.Duration
	sampleRate int
	client     *http.Client
}

type wyomingVietnameseResponse struct {
	Text          string `json:"text"`
	Transcription string `json:"transcription"`
	Result        string `json:"result"`
	Language      string `json:"language"`
	Error         string `json:"error"`
	Detail        any    `json:"detail"`
}

func NewWyomingVietnameseAdapter(config map[string]interface{}) (AsrProvider, error) {
	adapter := &WyomingVietnameseAdapter{
		baseURL:    "http://127.0.0.1:8082",
		timeout:    30 * time.Second,
		sampleRate: audio.SampleRate,
	}

	if baseURL := stringConfig(config, "base_url", "api_url", "url"); baseURL != "" {
		adapter.baseURL = strings.TrimRight(baseURL, "/")
	}
	if host := stringConfig(config, "host"); host != "" {
		port := stringConfig(config, "port")
		if port == "" {
			port = "9000"
		}
		adapter.baseURL = "http://" + host + ":" + port
	}
	if timeoutMs := intConfig(config, "timeout_ms"); timeoutMs > 0 {
		adapter.timeout = time.Duration(timeoutMs) * time.Millisecond
	} else if timeoutSeconds := intConfig(config, "timeout"); timeoutSeconds > 0 {
		adapter.timeout = time.Duration(timeoutSeconds) * time.Second
	}
	if sampleRate := intConfig(config, "sample_rate"); sampleRate > 0 {
		adapter.sampleRate = sampleRate
	}
	if _, err := url.ParseRequestURI(adapter.baseURL); err != nil {
		return nil, fmt.Errorf("base_url Vietnamese ASR Go không hợp lệ: %w", err)
	}
	adapter.client = &http.Client{Timeout: adapter.timeout}
	return adapter, nil
}

func (a *WyomingVietnameseAdapter) Process(pcmData []float32) (string, error) {
	if len(pcmData) == 0 {
		return "", nil
	}
	wavData := encodePCMFloat32ToWAV(pcmData, a.sampleRate)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("tạo form audio thất bại: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("ghi form audio thất bại: %w", err)
	}
	if err := writer.WriteField("language", "vi"); err != nil {
		return "", fmt.Errorf("ghi field language thất bại: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("đóng multipart writer thất bại: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/transcribe", body)
	if err != nil {
		return "", fmt.Errorf("tạo yêu cầu Vietnamese ASR Go thất bại: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gọi Vietnamese ASR Go thất bại: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("đọc phản hồi Vietnamese ASR Go thất bại: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Vietnamese ASR Go trả về HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result wyomingVietnameseResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		text := strings.TrimSpace(string(respBody))
		if text != "" {
			return text, nil
		}
		return "", fmt.Errorf("phân tích phản hồi Vietnamese ASR Go thất bại: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("Vietnamese ASR Go trả về lỗi: %s", result.Error)
	}
	text := strings.TrimSpace(firstNonEmpty(result.Text, result.Transcription, result.Result))
	if text == "" && result.Detail != nil {
		return "", fmt.Errorf("Vietnamese ASR Go không trả về text: %v", result.Detail)
	}
	return text, nil
}

func (a *WyomingVietnameseAdapter) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	resultChan := make(chan types.StreamingResult, 1)
	go func() {
		defer close(resultChan)
		var pcmData []float32
		for {
			select {
			case <-ctx.Done():
				resultChan <- types.StreamingResult{Error: ctx.Err(), AsrType: "wyoming_vietnamese_asr", IsFinal: true}
				return
			case chunk, ok := <-audioStream:
				if !ok {
					text, err := a.Process(pcmData)
					resultChan <- types.StreamingResult{Text: text, IsFinal: true, Error: err, AsrType: "wyoming_vietnamese_asr", Mode: "offline"}
					return
				}
				pcmData = append(pcmData, chunk...)
			}
		}
	}()
	return resultChan, nil
}

func (a *WyomingVietnameseAdapter) Close() error {
	return nil
}

func (a *WyomingVietnameseAdapter) IsValid() bool {
	return a != nil && a.client != nil && a.baseURL != ""
}

func encodePCMFloat32ToWAV(samples []float32, sampleRate int) []byte {
	pcmBytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		if sample > 1 {
			sample = 1
		} else if sample < -1 {
			sample = -1
		}
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(int16(sample*32767)))
	}

	dataSize := uint32(len(pcmBytes))
	byteRate := uint32(sampleRate * 2)
	blockAlign := uint16(2)
	buffer := &bytes.Buffer{}
	buffer.WriteString("RIFF")
	_ = binary.Write(buffer, binary.LittleEndian, uint32(36)+dataSize)
	buffer.WriteString("WAVE")
	buffer.WriteString("fmt ")
	_ = binary.Write(buffer, binary.LittleEndian, uint32(16))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(buffer, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buffer, binary.LittleEndian, byteRate)
	_ = binary.Write(buffer, binary.LittleEndian, blockAlign)
	_ = binary.Write(buffer, binary.LittleEndian, uint16(16))
	buffer.WriteString("data")
	_ = binary.Write(buffer, binary.LittleEndian, dataSize)
	buffer.Write(pcmBytes)
	return buffer.Bytes()
}

func stringConfig(config map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if config == nil {
			return ""
		}
		switch value := config[key].(type) {
		case string:
			return strings.TrimSpace(value)
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		case float64:
			return strconv.Itoa(int(value))
		}
	}
	return ""
}

func intConfig(config map[string]interface{}, key string) int {
	if config == nil {
		return 0
	}
	switch value := config[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(value))
		return parsed
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
