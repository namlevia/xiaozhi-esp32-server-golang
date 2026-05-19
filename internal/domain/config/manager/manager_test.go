package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestConfigManager_GetSystemConfig(t *testing.T) {
	// Tạo manager cấu hình
	config := map[string]interface{}{
		"backend_url": "http://192.168.208.214:8080", // Điều chỉnh theo địa chỉ backend thực tế
	}

	manager, err := NewManagerUserConfigProvider(config)
	if err != nil {
		t.Fatalf("Tạo manager cấu hình thất bại: %v", err)
	}

	// Tạo context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Lấy system config
	configJSON, err := manager.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("Lấy system config thất bại: %v", err)
	}

	// Xác minh định dạng JSON trả về
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
		t.Fatalf("Parse JSON config thất bại: %v", err)
	}

	// Kiểm tra có chứa các config item kỳ vọng hay không
	expectedKeys := []string{"mqtt", "mqtt_server", "udp", "ota"}
	for _, key := range expectedKeys {
		if _, exists := configMap[key]; !exists {
			t.Errorf("Config thiếu key kỳ vọng: %s", key)
		}
	}

	fmt.Printf("System config nhận được: %s\n", configJSON)
	t.Logf("Kích thước config: %d byte", len(configJSON))
}
