package user_config

import (
	"context"
	"testing"
)

func TestMemoryProvider(t *testing.T) {
	ctx := context.Background()

	// Tạo provider memory
	config := map[string]interface{}{
		"max_entries": 10,
	}

	provider, err := GetUserConfigProvider("memory", config)
	if err != nil {
		t.Fatalf("Tạo provider memory thất bại: %v", err)
	}
	// Lưu ý: interface không có method Close nên không cần gọi.

	userID := "test_user_123"

	// Vì interface không có method SetUserConfig nên chỉ test method GetUserConfig.
	// Test lấy config của user không tồn tại, kỳ vọng trả về config rỗng.
	retrievedConfig, err := provider.GetUserConfig(ctx, userID)
	if err != nil {
		t.Fatalf("Lấy cấu hình người dùng thất bại: %v", err)
	}

	// Xác minh giá trị trả về là config rỗng
	if retrievedConfig.Llm.Provider != "" {
		t.Errorf("Kỳ vọng config rỗng, nhưng nhận được LLM Provider: %s", retrievedConfig.Llm.Provider)
	}

	// Test lấy config hệ thống
	systemConfig, err := provider.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("Lấy config hệ thống thất bại: %v", err)
	}
	_ = systemConfig // Config hệ thống có thể rỗng, đây là bình thường
}

func TestProviderAdapter(t *testing.T) {
	ctx := context.Background()

	// Tạo provider memory
	provider, err := GetUserConfigProvider("memory", map[string]interface{}{
		"max_entries": 5,
	})
	if err != nil {
		t.Fatalf("Tạo provider memory thất bại: %v", err)
	}
	// Lưu ý: interface không có method Close nên không cần gọi.

	// Test adapter lấy config
	userID := "adapter_test_user"

	// Dùng adapter lấy config, có thể là config rỗng
	adapter := NewUserConfigAdapter(provider)
	retrievedConfig, err := adapter.GetUserConfig(ctx, userID)
	if err != nil {
		t.Fatalf("Lấy config qua adapter thất bại: %v", err)
	}

	// Xác minh adapter hoạt động bình thường, lấy được cấu trúc config
	if retrievedConfig.SystemPrompt == "" {
		t.Logf("Adapter lấy được system prompt rỗng, đây là bình thường")
	} else {
		t.Logf("Adapter lấy được system prompt: %s", retrievedConfig.SystemPrompt)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Test config mặc định Redis
	redisConfig := DefaultConfig("redis")
	if redisConfig["host"] != "localhost" {
		t.Errorf("Config host mặc định Redis sai, kỳ vọng: localhost, thực tế: %v", redisConfig["host"])
	}

	// Test config mặc định memory
	memoryConfig := DefaultConfig("memory")
	if memoryConfig["max_entries"] != 1000 {
		t.Errorf("Config max_entries mặc định memory sai, kỳ vọng: 1000, thực tế: %v", memoryConfig["max_entries"])
	}

	// Test loại không được hỗ trợ
	unknownConfig := DefaultConfig("unknown")
	if len(unknownConfig) != 0 {
		t.Errorf("Loại không xác định nên trả về config rỗng, thực tế: %v", unknownConfig)
	}
}

func TestValidateConfig(t *testing.T) {
	// Test config Redis hợp lệ
	validRedisConfig := map[string]interface{}{
		"host": "localhost",
		"port": 6379,
	}
	err := ValidateConfig("redis", validRedisConfig)
	if err != nil {
		t.Errorf("Validate config Redis hợp lệ thất bại: %v", err)
	}

	// Test config Redis không hợp lệ, thiếu host
	invalidRedisConfig := map[string]interface{}{
		"port": 6379,
	}
	err = ValidateConfig("redis", invalidRedisConfig)
	if err == nil {
		t.Error("Config Redis thiếu host phải validate thất bại")
	}

	// Test config Memory, không cần validate
	err = ValidateConfig("memory", map[string]interface{}{})
	if err != nil {
		t.Errorf("Validate config Memory thất bại: %v", err)
	}
}
