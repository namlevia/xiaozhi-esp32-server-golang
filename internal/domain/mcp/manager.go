package mcp

import (
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"
)

// MCPManager MCP manager thống nhất, chịu trách nhiệm điều phối các sub-manager
type MCPManager struct {
	localManager  *LocalMCPManager
	globalManager *GlobalMCPManager
	// deviceManager sau này có thể quản lý pool device manager tại đây

	mu      sync.RWMutex
	started bool
}

var (
	mcpManager *MCPManager
	mcpOnce    sync.Once
)

// GetMCPManager Lấy singleton MCP manager thống nhất
func GetMCPManager() *MCPManager {
	mcpOnce.Do(func() {
		mcpManager = &MCPManager{
			localManager:  GetLocalMCPManager(),
			globalManager: GetGlobalMCPManager(),
			started:       false,
		}
	})
	return mcpManager
}

// Start Khởi động tất cả MCP manager
func (m *MCPManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		log.Warn("MCP manager đã khởi động")
		return nil
	}

	log.Info("=== Khởi động cụm MCP manager ===")

	// 1. Khởi động local manager trước
	log.Info("Đang khởi động Local MCP manager...")
	if err := m.localManager.Start(); err != nil {
		log.Errorf("Khởi động Local MCP manager thất bại: %v", err)
		return fmt.Errorf("Khởi động Local MCP manager thất bại: %v", err)
	}

	// 2. Sau đó khởi động Global manager
	log.Info("Đang khởi động Global MCP manager...")
	if err := m.globalManager.Start(); err != nil {
		log.Errorf("Khởi động Global MCP manager thất bại: %v", err)
		return fmt.Errorf("Khởi động Global MCP manager thất bại: %v", err)
	}

	// 3. Device manager được tạo động khi kết nối, không cần khởi động tại đây
	log.Info("Device MCP manager sẽ được tạo động theo kết nối")

	m.started = true
	log.Info("=== Cụm MCP manager khởi động hoàn tất ===")

	// Xuất thống kê trạng thái khởi động
	m.printStartupStats()

	return nil
}

// Stop Dừng tất cả MCP manager
func (m *MCPManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		log.Info("MCP manager chưa khởi động, không cần dừng")
		return nil
	}

	log.Info("=== Dừng cụm MCP manager ===")

	// Dừng manager theo thứ tự ngược lại
	// 1. Dừng Global manager
	log.Info("Đang dừng Global MCP manager...")
	if err := m.globalManager.Stop(); err != nil {
		log.Errorf("Dừng Global MCP manager thất bại: %v", err)
	}

	// 2. Dừng local manager
	log.Info("Đang dừng Local MCP manager...")
	if err := m.localManager.Stop(); err != nil {
		log.Errorf("Dừng Local MCP manager thất bại: %v", err)
	}

	// 3. Device manager tự dọn khi kết nối ngắt
	log.Info("Kết nối Device MCP sẽ tự dọn")

	m.started = false
	log.Info("=== Cụm MCP manager đã dừng ===")
	return nil
}

// IsStarted Kiểm tra manager đã khởi động chưa
func (m *MCPManager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// GetLocalManager Lấy local manager
func (m *MCPManager) GetLocalManager() *LocalMCPManager {
	return m.localManager
}

// GetGlobalManager Lấy global manager
func (m *MCPManager) GetGlobalManager() *GlobalMCPManager {
	return m.globalManager
}

// printStartupStats Xuất thống kê trạng thái khởi động
func (m *MCPManager) printStartupStats() {
	localToolCount := m.localManager.GetToolCount()
	globalToolCount := len(m.globalManager.GetAllTools())

	log.Infof("Thống kê khởi động MCP manager:")
	log.Infof("  - Số tool local: %d", localToolCount)
	log.Infof("  - Số tool global: %d", globalToolCount)
	log.Infof("  - Device manager: quản lý động")
	log.Infof("  - Tổng số tool: %d", localToolCount+globalToolCount)
}

// GetAllManagersStatus Lấy thông tin trạng thái của tất cả manager
func (m *MCPManager) GetAllManagersStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"mcp_manager": map[string]interface{}{
			"started": m.started,
		},
		"local_manager": map[string]interface{}{
			"tool_count": m.localManager.GetToolCount(),
			"tool_names": m.localManager.GetToolNames(),
		},
		"global_manager": map[string]interface{}{
			"tool_count": len(m.globalManager.GetAllTools()),
		},
		"device_manager": map[string]interface{}{
			"active_devices": mcpClientPool.device2McpClient.Count(),
		},
	}

	return status
}

// RestartManager Khởi động lại manager được chỉ định
func (m *MCPManager) RestartManager(managerType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("Cụm MCP manager chưa khởi động")
	}

	switch managerType {
	case "local":
		log.Info("Đang khởi động lại Local MCP manager...")
		if err := m.localManager.Stop(); err != nil {
			log.Errorf("Dừng local manager thất bại: %v", err)
		}
		if err := m.localManager.Start(); err != nil {
			return fmt.Errorf("Khởi động lại local manager thất bại: %v", err)
		}
		log.Info("Local MCP manager khởi động lại hoàn tất")

	case "global":
		log.Info("Đang khởi động lại Global MCP manager...")
		if err := m.globalManager.Stop(); err != nil {
			log.Errorf("Dừng global manager thất bại: %v", err)
		}
		if err := m.globalManager.Start(); err != nil {
			return fmt.Errorf("Khởi động lại global manager thất bại: %v", err)
		}
		log.Info("Global MCP manager khởi động lại hoàn tất")

	default:
		return fmt.Errorf("Không hỗ trợ loại manager: %s", managerType)
	}

	return nil
}

// Cung cấp hàm tiện ích để tương thích ngược

// StartMCPManagers Khởi động tất cả MCP manager(hàm tiện ích)
func StartMCPManagers() error {
	return GetMCPManager().Start()
}

// StopMCPManagers Dừng tất cả MCP manager(hàm tiện ích)
func StopMCPManagers() error {
	return GetMCPManager().Stop()
}

// GetMCPManagerStatus lấy trạng thái MCP manager (hàm tiện ích)
func GetMCPManagerStatus() map[string]interface{} {
	return GetMCPManager().GetAllManagersStatus()
}
