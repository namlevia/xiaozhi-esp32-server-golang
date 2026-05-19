package piper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPiperProviderPostsExpectedRequest(t *testing.T) {
	var got piperRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tts" {
			t.Fatalf("path = %q, want /tts", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write(make([]byte, 640))
	}))
	defer server.Close()

	provider := NewPiperTTSProvider(map[string]interface{}{
		"api_url":           server.URL,
		"voice":             "banmai",
		"model_path":        "tts_server/tts-model/banmai.onnx",
		"model_config_path": "tts_server/tts-model/banmai.onnx.json",
		"response_format":   "pcm",
		"sample_rate":       float64(16000),
		"frame_duration":    float64(20),
	})

	frames, err := provider.TextToSpeech(context.Background(), "xin chào", 16000, 1, 20)
	if err != nil {
		t.Fatalf("TextToSpeech() error = %v", err)
	}
	if len(frames) == 0 {
		t.Fatal("TextToSpeech() returned no frames")
	}
	if got.Text != "xin chào" {
		t.Fatalf("text = %q, want xin chào", got.Text)
	}
	if got.Voice != "banmai" {
		t.Fatalf("voice = %q, want banmai", got.Voice)
	}
	if got.ModelPath != "tts_server/tts-model/banmai.onnx" {
		t.Fatalf("model_path = %q", got.ModelPath)
	}
	if got.ModelConfigPath != "tts_server/tts-model/banmai.onnx.json" {
		t.Fatalf("model_config_path = %q", got.ModelConfigPath)
	}
	if got.ResponseFormat != "pcm" {
		t.Fatalf("response_format = %q, want pcm", got.ResponseFormat)
	}
	if got.SampleRate != 16000 {
		t.Fatalf("sample_rate = %d, want 16000", got.SampleRate)
	}
}

func TestPiperProviderNormalizesAPIURL(t *testing.T) {
	provider := NewPiperTTSProvider(map[string]interface{}{"api_url": "http://piper-tts:9002"})
	if provider.APIURL != "http://piper-tts:9002/tts" {
		t.Fatalf("APIURL = %q", provider.APIURL)
	}

	provider = NewPiperTTSProvider(map[string]interface{}{"api_url": "http://piper-tts:9002/tts"})
	if provider.APIURL != "http://piper-tts:9002/tts" {
		t.Fatalf("APIURL = %q", provider.APIURL)
	}
}

func TestPiperSetVoiceSupportsModelPath(t *testing.T) {
	provider := NewPiperTTSProvider(map[string]interface{}{})
	if err := provider.SetVoice(map[string]interface{}{
		"model_path":        "tts_server/tts-model/john.onnx",
		"model_config_path": "tts_server/tts-model/john.onnx.json",
	}); err != nil {
		t.Fatalf("SetVoice() error = %v", err)
	}
	if provider.ModelPath != "tts_server/tts-model/john.onnx" {
		t.Fatalf("ModelPath = %q", provider.ModelPath)
	}
	if provider.ModelConfigPath != "tts_server/tts-model/john.onnx.json" {
		t.Fatalf("ModelConfigPath = %q", provider.ModelConfigPath)
	}
}
