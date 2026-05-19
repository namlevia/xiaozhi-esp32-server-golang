package user_config

import (
	"fmt"

	"xiaozhi-esp32-server-golang/internal/domain/config/manager"
	userconfig_redis "xiaozhi-esp32-server-golang/internal/domain/config/redis"
	"xiaozhi-esp32-server-golang/internal/util"
)

// Config là cấu trúc config provider cấu hình người dùng.
type Config struct {
	Type       string                 `json:"type"`       // Loại lưu trữ: "redis", "memory", "file"
	Parameters map[string]interface{} `json:"parameters"` // Tham số config liên quan lưu trữ
}

func GetProvider(sType string) (UserConfigProvider, error) {
	config := make(map[string]interface{})
	if sType == "manager" {
		// Ưu tiên lấy địa chỉ backend từ biến môi trường; nếu không có thì lấy từ config.
		backendUrl := util.GetBackendURL()
		config = map[string]interface{}{
			"backend_url": backendUrl,
			"auth_token":  util.GetManagerAuthToken(),
		}
	}

	provider, err := GetUserConfigProvider(sType, config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// GetUserConfigProvider tạo provider cấu hình người dùng.
// Tạo instance provider tương ứng theo loại lưu trữ và tham số config truyền vào.
// providerType: loại provider, hỗ trợ "redis", "memory", "file".
// config: tham số config provider.
// Trả về interface UserConfigProvider, hỗ trợ đầy đủ thao tác CRUD.
func GetUserConfigProvider(providerType string, config map[string]interface{}) (UserConfigProvider, error) {
	if config == nil {
		config = make(map[string]interface{})
	}

	switch providerType {
	case "redis":
		// Tạo provider cấu hình người dùng Redis
		provider, err := userconfig_redis.NewRedisUserConfigProvider(config)
		if err != nil {
			return nil, fmt.Errorf("Tạo provider cấu hình người dùng Redis thất bại: %v", err)
		}
		return provider, nil
	case "manager":
		// Tạo provider cấu hình người dùng backend manager
		provider, err := manager.NewManagerUserConfigProvider(config)
		if err != nil {
			return nil, fmt.Errorf("Tạo provider cấu hình người dùng backend manager thất bại: %v", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("Provider cấu hình người dùng không được hỗ trợ: %s", providerType)
	}
}
