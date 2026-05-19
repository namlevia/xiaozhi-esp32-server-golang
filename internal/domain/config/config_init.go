package user_config

import (
	"context"
	"fmt"
	log "xiaozhi-esp32-server-golang/logger"

	"xiaozhi-esp32-server-golang/internal/domain/config/manager"
	"xiaozhi-esp32-server-golang/internal/domain/config/memory"
	redis_config "xiaozhi-esp32-server-golang/internal/domain/config/redis"

	"github.com/spf13/viper"
)

var (
	// managerSystemConfigHandlers là danh sách callback khi nhận push system_config qua WebSocket; chương trình chính có thể đăng ký nhiều lần, ví dụ merge vào viper hoặc hot reload service.
	managerSystemConfigHandlers []func(map[string]interface{})
)

// RegisterManagerSystemConfigHandler đăng ký callback push system config ở mode manager; nên gọi trước InitConfigSystem và có thể gọi nhiều lần để thêm callback.
func RegisterManagerSystemConfigHandler(fn func(map[string]interface{})) {
	managerSystemConfigHandlers = append(managerSystemConfigHandlers, fn)
}

// InitConfigSystem khởi tạo hệ thống config.
// Gọi Init của package config tương ứng theo giá trị config_provider.type.
func InitConfigSystem(ctx context.Context) error {
	// Lấy loại provider config
	providerType := viper.GetString("config_provider.type")
	if providerType == "" {
		providerType = "redis" // Mặc định dùng redis
		log.Infof("config_provider.type not set, using default: redis")
	}

	log.Infof("Initializing config system with provider: %s", providerType)

	// Gọi Init tương ứng theo loại provider config
	switch providerType {
	case "manager":
		manager.SetSystemConfigPushHandler(func(data map[string]interface{}) {
			for _, h := range managerSystemConfigHandlers {
				h(data)
			}
		})
		return manager.Init(ctx)
	case "redis":
		return redis_config.Init(ctx)
	case "memory":
		return memory.Init(ctx)
	default:
		return fmt.Errorf("unsupported config provider type: %s", providerType)
	}
}
