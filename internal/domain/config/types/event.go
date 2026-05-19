package types

import "context"

type EventHandler func(ctx context.Context, eventType string, eventData map[string]interface{}) (string, error)

// Sự kiện push upstream: chương trình chính => manager nội bộ.
const (
	EventDeviceOnline  = "/api/device/active"   // Thiết bị online
	EventDeviceOffline = "/api/device/inactive" // Thiết bị offline
)

// Sự kiện pull downstream: manager nội bộ => chương trình chính.
const (
	EventHandleMessageInject = "/api/device/inject_msg" // Xử lý inject message
)
