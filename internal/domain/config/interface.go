package user_config

import (
	"context"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
)

// UserConfigProvider là interface provider cấu hình người dùng.
// Đây là interface mở rộng, hỗ trợ nhiều thao tác hơn so với interface UserConfig cũ.
type UserConfigProvider interface {
	//auth
	// Lấy thông tin kích hoạt theo deviceId và clientId.
	IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error)
	GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int)
	VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error)

	//llm memory

	// GetUserConfig lấy cấu hình người dùng, tương thích interface cũ.
	GetUserConfig(ctx context.Context, userID string) (types.UConfig, error)

	// SwitchDeviceRoleByName chuyển role thiết bị theo tên role, hỗ trợ match mờ.
	SwitchDeviceRoleByName(ctx context.Context, deviceID string, roleName string) (string, error)

	// RestoreDeviceDefaultRole khôi phục role mặc định của thiết bị bằng cách xóa role đang bind.
	RestoreDeviceDefaultRole(ctx context.Context, deviceID string) error

	// Lấy config mqtt, mqtt_server, udp, ota, vision.
	GetSystemConfig(ctx context.Context) (string, error)

	// Đăng ký handler sự kiện upstream, ví dụ thiết bị online/offline.
	NotifyDeviceEvent(ctx context.Context, eventType string, eventData map[string]interface{})
	// Đăng ký handler sự kiện downstream, ví dụ inject message.
	RegisterMessageEventHandler(ctx context.Context, eventType string, eventHandler types.EventHandler)
}
