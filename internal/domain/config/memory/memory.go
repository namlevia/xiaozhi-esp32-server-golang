package memory

import (
	"context"
	"fmt"
	"sync"

	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// MemoryUserConfigProvider là provider cấu hình người dùng trong memory.
// Triển khai interface UserConfigProvider, lưu config trong memory.
// Lưu ý: dữ liệu sẽ mất sau khi restart, phù hợp cho test hoặc lưu tạm.
type MemoryUserConfigProvider struct {
	mu         sync.RWMutex
	configs    map[string]types.UConfig
	maxEntries int
}

// MemoryConfig là cấu trúc config memory.
type MemoryConfig struct {
	MaxEntries int `json:"max_entries"` // Số entry lưu trữ tối đa
}

// NewMemoryUserConfigProvider tạo provider cấu hình người dùng memory.
// config: map tham số config, gồm max_entries và các field khác.
func NewMemoryUserConfigProvider(config map[string]interface{}) (*MemoryUserConfigProvider, error) {
	// Parse tham số config
	memoryConfig := &MemoryConfig{
		MaxEntries: 1000, // Mặc định tối đa 1000 config
	}

	if maxEntries, ok := config["max_entries"].(int); ok && maxEntries > 0 {
		memoryConfig.MaxEntries = maxEntries
	} else if maxEntriesFloat, ok := config["max_entries"].(float64); ok && maxEntriesFloat > 0 {
		memoryConfig.MaxEntries = int(maxEntriesFloat)
	}

	provider := &MemoryUserConfigProvider{
		configs:    make(map[string]types.UConfig),
		maxEntries: memoryConfig.MaxEntries,
	}

	log.Log().Infof("Khởi tạo provider cấu hình người dùng memory thành công, số entry tối đa: %d", memoryConfig.MaxEntries)
	return provider, nil
}

// GetUserConfig lấy cấu hình người dùng.
func (m *MemoryUserConfigProvider) GetUserConfig(ctx context.Context, userID string) (types.UConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[userID]
	if !exists {
		log.Log().Debugf("Cấu hình người dùng %s không tồn tại, trả về config rỗng", userID)
		return types.UConfig{}, nil
	}

	return config, nil
}

// SetUserConfig thiết lập cấu hình người dùng.
func (m *MemoryUserConfigProvider) SetUserConfig(ctx context.Context, userID string, config types.UConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Kiểm tra có vượt quá số entry tối đa hay không
	if len(m.configs) >= m.maxEntries && !m.configExists(userID) {
		return fmt.Errorf("Đã đạt số entry lưu trữ tối đa %d, không thể thêm config mới", m.maxEntries)
	}

	m.configs[userID] = config
	log.Log().Infof("Thiết lập cấu hình người dùng %s thành công (lưu memory)", userID)
	return nil
}

// DeleteUserConfig xóa cấu hình người dùng.
func (m *MemoryUserConfigProvider) DeleteUserConfig(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.configs[userID]; !exists {
		log.Log().Warnf("Cấu hình người dùng %s không tồn tại, không cần xóa", userID)
		return nil
	}

	delete(m.configs, userID)
	log.Log().Infof("Xóa cấu hình người dùng %s thành công (lưu memory)", userID)
	return nil
}

// Close đóng provider; provider memory không cần cleanup đặc biệt.
func (m *MemoryUserConfigProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Xóa toàn bộ config
	m.configs = make(map[string]types.UConfig)
	log.Log().Info("Provider cấu hình người dùng memory đã đóng, toàn bộ config đã được xóa")
	return nil
}

// configExists kiểm tra config có tồn tại hay không; method nội bộ, cần giữ lock khi gọi.
func (m *MemoryUserConfigProvider) configExists(userID string) bool {
	_, exists := m.configs[userID]
	return exists
}

// GetStats lấy thông tin thống kê lưu trữ, method tiện ích bổ sung.
func (m *MemoryUserConfigProvider) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_configs": len(m.configs),
		"max_entries":   m.maxEntries,
		"usage_percent": float64(len(m.configs)) / float64(m.maxEntries) * 100,
	}
}

// ListUserIDs liệt kê toàn bộ user ID, method tiện ích bổ sung.
func (m *MemoryUserConfigProvider) ListUserIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userIDs := make([]string, 0, len(m.configs))
	for userID := range m.configs {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

// GetSystemConfig lấy config hệ thống.
func (m *MemoryUserConfigProvider) GetSystemConfig(ctx context.Context) (string, error) {
	// Provider cấu hình memory không cung cấp system config
	return "", nil
}

// Init khởi tạo provider cấu hình Memory.
func Init(ctx context.Context) error {
	log.Log().Info("Memory config provider initialized successfully")
	return nil
}

// Close đóng provider cấu hình Memory và cleanup tài nguyên.
func Close() error {
	log.Log().Info("Memory config provider closed")
	return nil
}

// IsConnected kiểm tra provider cấu hình Memory đã kết nối hay chưa.
func IsConnected() bool {
	// Provider cấu hình memory luôn ở trạng thái "kết nối"
	return true
}
