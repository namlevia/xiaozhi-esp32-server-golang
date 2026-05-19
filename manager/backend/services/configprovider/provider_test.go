package configprovider

import "testing"

func TestNormalizeProviderInfersKnownProviderInsteadOfConfigID(t *testing.T) {
	tests := []struct {
		name       string
		configType string
		configID   string
		data       map[string]interface{}
		want       string
	}{
		{
			name:       "llm openai compatible proxy id",
			configType: "llm",
			configID:   "proxy_qwen",
			data: map[string]interface{}{
				"type":     "openai",
				"base_url": "https://api.onethingai.com/v1",
			},
			want: "openai",
		},
		{
			name:       "asr funasr custom id",
			configType: "asr",
			configID:   "local_asr",
			data: map[string]interface{}{
				"host": "127.0.0.1",
				"port": 10096,
			},
			want: "funasr",
		},
		{
			name:       "tts aliyun qwen custom id",
			configType: "tts",
			configID:   "proxy_tts",
			data: map[string]interface{}{
				"api_url": "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
				"model":   "qwen-tts",
			},
			want: "aliyun_qwen",
		},
		{
			name:       "tts piper model path",
			configType: "tts",
			configID:   "offline_voice",
			data: map[string]interface{}{
				"model_path":        "tts_server/tts-model/banmai.onnx",
				"model_config_path": "tts_server/tts-model/banmai.onnx.json",
			},
			want: "piper",
		},
		{
			name:       "unknown vad falls back to managed default",
			configType: "vad",
			configID:   "custom_vad",
			data:       map[string]interface{}{},
			want:       "ten_vad",
		},
		{
			name:       "unknown asr falls back to Vietnamese ASR default",
			configType: "asr",
			configID:   "custom_asr",
			data:       map[string]interface{}{},
			want:       "wyoming_vietnamese_asr",
		},
		{
			name:       "llm gemini openai compatible endpoint",
			configType: "llm",
			configID:   "gemini_chat",
			data: map[string]interface{}{
				"type":     "openai",
				"base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
			},
			want: "gemini",
		},
		{
			name:       "llm openrouter endpoint",
			configType: "llm",
			configID:   "router_chat",
			data: map[string]interface{}{
				"type":     "openai",
				"base_url": "https://openrouter.ai/api/v1",
			},
			want: "openrouter",
		},
		{
			name:       "llm 9router endpoint",
			configType: "llm",
			configID:   "nine_router_chat",
			data: map[string]interface{}{
				"type":     "openai",
				"base_url": "https://api.9router.com/v1",
			},
			want: "9router",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeProvider(tt.configType, tt.configID, tt.data); got != tt.want {
				t.Fatalf("NormalizeProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExportDataAddsNormalizedProviderAndLLMType(t *testing.T) {
	got := ExportData("llm", "proxy_qwen", "proxy_qwen", map[string]interface{}{
		"model_name": "qwen-plus",
	})

	if got["provider"] != "openai" {
		t.Fatalf("provider = %q, want openai", got["provider"])
	}
	if got["type"] != "openai" {
		t.Fatalf("type = %q, want openai", got["type"])
	}
}
