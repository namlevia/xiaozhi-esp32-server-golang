package cosyvoice

import (
	"context"
	"os"
	"testing"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
)

func TestNewCosyVoiceTTSProviderDefaultsAndSetVoice(t *testing.T) {
	provider := NewCosyVoiceTTSProvider(map[string]interface{}{})

	if provider.APIURL != "https://tts.linkerai.cn/tts" {
		t.Fatalf("APIURL = %q", provider.APIURL)
	}
	if provider.SpeakerID == "" {
		t.Fatal("SpeakerID should use a default")
	}
	if provider.FrameDuration != audio.FrameDuration {
		t.Fatalf("FrameDuration = %d", provider.FrameDuration)
	}
	if provider.TargetSR != audio.SampleRate {
		t.Fatalf("TargetSR = %d", provider.TargetSR)
	}
	if provider.AudioFormat != "mp3" {
		t.Fatalf("AudioFormat = %q", provider.AudioFormat)
	}
	if !provider.IsValid() {
		t.Fatal("provider should be valid")
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}

	if err := provider.SetVoice(map[string]interface{}{"spk_id": "speaker-2"}); err != nil {
		t.Fatalf("SetVoice error = %v", err)
	}
	if provider.SpeakerID != "speaker-2" {
		t.Fatalf("SpeakerID = %q", provider.SpeakerID)
	}
	if err := provider.SetVoice(map[string]interface{}{}); err == nil {
		t.Fatal("expected missing spk_id to fail")
	}
}

func TestCosyVoiceTTS(t *testing.T) {
	// Bỏ qua networkrequesttest，trừ khi đã đặt biến môi trường
	if os.Getenv("RUN_COSYVOICE_TEST") != "1" {
		t.Skip("Bỏ quaCosyVoice APItest，đặt biến môi trườngRUN_COSYVOICE_TEST=1để bật")
	}

	config := map[string]interface{}{
		"api_url":        "https://cosyvoice.com/tts",
		"spk_id":         "OUeAo1mhq6IBExi",
		"frame_duration": float64(60),
		"target_sr":      float64(16000),
		"audio_format":   "mp3",
		"instruct_text":  "Xin chào",
	}

	provider := NewCosyVoiceTTSProvider(config)

	// testtext-to-speech
	t.Run("TestTextToSpeech", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		frames, err := provider.TextToSpeech(ctx, "Bạn có nói được tiếng Việt không", 16000, 1, 60)
		if err != nil {
			t.Fatalf("TextToSpeechthất bại: %v", err)
		}

		if len(frames) == 0 {
			t.Error("chưatrả vềbất kỳaudioframe")
		}
	})

	// teststreamingtext-to-speech
	t.Run("TestTextToSpeechStream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		outputChan, err := provider.TextToSpeechStream(ctx, "Bạn có nói được tiếng Việt không", 16000, 1, 60)
		if err != nil {
			t.Fatalf("TextToSpeechStreamthất bại: %v", err)
		}

		// Nhận tất cảframe
		var receivedFrames [][]byte
		timeout := time.After(10 * time.Second)

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
	})
}
