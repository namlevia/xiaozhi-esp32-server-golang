package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
)

func TestOpenAITTS(t *testing.T) {
	// Bỏ qua networkrequesttest，trừ khi đã đặt biến môi trường
	if os.Getenv("RUN_OPENAI_TEST") != "1" {
		t.Skip("Bỏ quaOpenAI APItest，đặt biến môi trườngRUN_OPENAI_TEST=1để bật")
	}

	// noi_dunglấyAPIkey
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Bỏ quaOpenAI APItest，noi_dungđặt biến môi trườngOPENAI_API_KEY")
	}

	config := map[string]interface{}{
		"api_key":         apiKey,
		"api_url":         "https://api.openai.com/v1/audio/speech",
		"model":           "tts-1",
		"voice":           "alloy",
		"response_format": "mp3",
		"speed":           1.0,
		"frame_duration":  float64(60),
	}

	provider := NewOpenAITTSProvider(config)

	// testtext-to-speech
	t.Run("TestTextToSpeech", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		frames, err := provider.TextToSpeech(ctx, "Hello, this is a test of OpenAI text to speech.", 16000, 1, 60)
		if err != nil {
			t.Fatalf("TextToSpeechthất bại: %v", err)
		}

		if len(frames) == 0 {
			t.Error("chưatrả vềbất kỳaudioframe")
		}

		t.Logf("thành côngnoi_dung %d noi_dungaudioframe", len(frames))
	})

	// teststreamingtext-to-speech
	t.Run("TestTextToSpeechStream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		outputChan, err := provider.TextToSpeechStream(ctx, "Hello, this is a test of OpenAI streaming text to speech.", 16000, 1, 60)
		if err != nil {
			t.Fatalf("TextToSpeechStreamthất bại: %v", err)
		}

		// Nhận tất cảframe
		var receivedFrames [][]byte
		timeout := time.After(20 * time.Second)

	receiveLoop:
		for {
			select {
			case frame, ok := <-outputChan:
				if !ok {
					break receiveLoop
				}
				receivedFrames = append(receivedFrames, frame)
			case <-timeout:
				t.Error("nhậnaudioframetimeout")
				break receiveLoop
			}
		}

		if len(receivedFrames) == 0 {
			t.Error("chưanoi_dungNhậnbất kỳaudioframe")
		}

		t.Logf("thành côngnhận %d noi_dungaudioframe", len(receivedFrames))
	})

	// testnoi_dungvoice
	t.Run("TestDifferentVoices", func(t *testing.T) {
		voices := []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}

		for _, voice := range voices {
			t.Run(voice, func(t *testing.T) {
				config := map[string]interface{}{
					"api_key":         apiKey,
					"model":           "tts-1",
					"voice":           voice,
					"response_format": "mp3",
					"speed":           1.0,
				}

				provider := NewOpenAITTSProvider(config)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				frames, err := provider.TextToSpeech(ctx, "Testing voice: "+voice, 16000, 1, 60)
				if err != nil {
					t.Errorf("dùngvoice %s thất bại: %v", voice, err)
					return
				}

				if len(frames) == 0 {
					t.Errorf("voice %s chưatrả vềbất kỳaudioframe", voice)
				}

				t.Logf("voice %s thành côngnoi_dung %d noi_dungaudioframe", voice, len(frames))
			})
		}
	})

	// testnoi_dung
	t.Run("TestDifferentSpeeds", func(t *testing.T) {
		speeds := []float64{0.5, 1.0, 1.5, 2.0}

		for _, speed := range speeds {
			t.Run(string(rune(speed)), func(t *testing.T) {
				config := map[string]interface{}{
					"api_key":         apiKey,
					"model":           "tts-1",
					"voice":           "alloy",
					"response_format": "mp3",
					"speed":           speed,
				}

				provider := NewOpenAITTSProvider(config)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				frames, err := provider.TextToSpeech(ctx, "Testing speed", 16000, 1, 60)
				if err != nil {
					t.Errorf("dùngnoi_dung %.1f thất bại: %v", speed, err)
					return
				}

				if len(frames) == 0 {
					t.Errorf("noi_dung %.1f chưatrả vềbất kỳaudioframe", speed)
				}

				t.Logf("noi_dung %.1f thành côngnoi_dung %d noi_dungaudioframe", speed, len(frames))
			})
		}
	})
}

// TestOpenAITTSProviderDefaults testmặc địnhnoi_dung
func TestOpenAITTSProviderDefaults(t *testing.T) {
	config := map[string]interface{}{
		"api_key": "test-key",
	}

	provider := NewOpenAITTSProvider(config)

	if provider.APIURL != "https://api.openai.com/v1/audio/speech" {
		t.Errorf("noi_dungmặc địnhAPI URLlà https://api.openai.com/v1/audio/speech，thực tếlà %s", provider.APIURL)
	}

	if provider.Model != "tts-1" {
		t.Errorf("noi_dungmặc địnhmodellà tts-1，thực tếlà %s", provider.Model)
	}

	if provider.Voice != "alloy" {
		t.Errorf("noi_dungmặc địnhvoicelà alloy，thực tếlà %s", provider.Voice)
	}

	if provider.ResponseFormat != "mp3" {
		t.Errorf("noi_dungmặc địnhresponseformatlà mp3，thực tếlà %s", provider.ResponseFormat)
	}

	if provider.Speed != 1.0 {
		t.Errorf("noi_dungmặc địnhnoi_dunglà 1.0，thực tếlà %.1f", provider.Speed)
	}
}

func TestOpenAITTSProviderSupportsOpusResponse(t *testing.T) {
	sampleRate := 16000
	pcm := make([]int16, sampleRate/2)
	for i := range pcm {
		if i%32 < 16 {
			pcm[i] = 2400
		} else {
			pcm[i] = -2400
		}
	}

	opusBytes, err := util.PCM16ToOggOpus(pcm, sampleRate, 1, 20)
	if err != nil {
		t.Fatalf("noi_dungtest Ogg Opus thất bại: %v", err)
	}

	requestErrCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			requestErrCh <- err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ResponseFormat != "opus" {
			requestErrCh <- fmt.Errorf("noi_dung response_format=opus，thực tếlà %s", req.ResponseFormat)
			http.Error(w, "unexpected response_format", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "audio/ogg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(opusBytes)
	}))
	defer server.Close()

	provider := NewOpenAITTSProvider(map[string]interface{}{
		"api_url":         server.URL,
		"model":           "tts-1",
		"voice":           "alloy",
		"response_format": "opus",
		"speed":           1.0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	outputChan, err := provider.TextToSpeechStream(ctx, "test opus output", sampleRate, 1, 60)
	if err != nil {
		t.Fatalf("TextToSpeechStream trả vềlỗi: %v", err)
	}

	frameCount := 0
	for frame := range outputChan {
		if len(frame) == 0 {
			t.Fatal("Nhậnrỗng Opus frame")
		}
		frameCount++
	}

	if frameCount == 0 {
		t.Fatal("chưaNhậnbất kỳ Opus frame")
	}

	select {
	case err := <-requestErrCh:
		t.Fatalf("mock server noi_dungthất bại: %v", err)
	default:
	}
}
