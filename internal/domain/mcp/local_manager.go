package mcp

import (
	"encoding/json"
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"

	mcp_protocol "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// LocalMCPManager Local MCP tool manager
type LocalMCPManager struct {
	tools map[string]*McpTool // tên tool -> định nghĩa tool
	mu    sync.RWMutex        // RW lock bảo vệ truy cập đồng thời
}

var (
	localManager *LocalMCPManager
	localOnce    sync.Once
)

// GetLocalMCPManager Lấy singleton Local MCP manager
func GetLocalMCPManager() *LocalMCPManager {
	localOnce.Do(func() {
		localManager = &LocalMCPManager{
			tools: make(map[string]*McpTool),
		}
		// Khởi tạo tool local mặc định
		localManager.initDefaultTools()
	})
	return localManager
}

// initDefaultTools Khởi tạo tool local mặc định
func (l *LocalMCPManager) initDefaultTools() {

	log.Info("Khởi tạo tool mặc định của Local MCP manager hoàn tất")
}

// RegisterTool Đăng ký tool local
func (l *LocalMCPManager) RegisterTool(tool *McpTool) error {
	if tool == nil {
		return fmt.Errorf("Tool không được rỗng")
	}

	if tool.info.Name == "" {
		return fmt.Errorf("Tên tool không được rỗng")
	}

	if !tool.isLocal || tool.localHandler == nil {
		return fmt.Errorf("Hàm xử lý tool không được rỗng")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Kiểm tra tool đã tồn tại chưa
	if _, exists := l.tools[tool.info.Name]; exists {
		log.Warnf("Tool local %s đã tồn tại, sẽ bị ghi đè", tool.info.Name)
	}

	l.tools[tool.info.Name] = tool
	log.Infof("Đăng ký tool local thành công: %s - %s", tool.info.Name, tool.info.Desc)
	return nil
}

func (l *LocalMCPManager) convertStructToOpenaipi3Schema(inputParams any) (*openapi3.Schema, error) {
	//Dùng github.com/ThinkInAIXYZ/go-mcp tạo tool từ struct, rồi chuyển thành openapi3.Schema
	toolInstance, err := mcp_protocol.NewTool("get_system_info", "Lấy thông tin cơ bản của hệ thống", inputParams)
	if err != nil {
		return nil, err
	}

	marshaledInputSchema, err := json.Marshal(toolInstance.InputSchema)
	if err != nil {
		return nil, err
	}

	inputSchema := &openapi3.Schema{}
	err = json.Unmarshal(marshaledInputSchema, inputSchema)
	if err != nil {
		return nil, err
	}
	return inputSchema, nil
}

// RegisterToolFunc Đăng ký hàm tool (bản đơn giản)
func (l *LocalMCPManager) RegisterToolFunc(name, description string, inputParams any, handler LocalToolHandler) error {
	inputSchema, err := l.convertStructToOpenaipi3Schema(inputParams)
	if err != nil {
		log.Errorf("Failed to convert struct to openapi3 schema: %v", err)
		return err
	}
	tool := &McpTool{
		info: &schema.ToolInfo{
			Name:        name,
			Desc:        description,
			ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(inputSchema),
		},
		isLocal:      true,
		localHandler: handler,
	}
	return l.RegisterTool(tool)
}

// UnregisterTool Hủy đăng ký tool
func (l *LocalMCPManager) UnregisterTool(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.tools[name]; !exists {
		return fmt.Errorf("Tool %s không tồn tại", name)
	}

	delete(l.tools, name)
	log.Infof("Hủy đăng ký tool local thành công: %s", name)
	return nil
}

// GetAllTools Lấy tất cả tool local, trả về định dạng interface tool Eino
func (l *LocalMCPManager) GetAllTools() map[string]tool.InvokableTool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]tool.InvokableTool)
	for name, mcpTool := range l.tools {
		result[name] = mcpTool
	}
	return result
}

// GetToolByName Lấy tool theo tên
func (l *LocalMCPManager) GetToolByName(name string) (tool.InvokableTool, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mcpTool, exists := l.tools[name]
	if !exists {
		return nil, false
	}

	return mcpTool, true
}

// GetToolNames Lấy danh sách tên tất cả tool
func (l *LocalMCPManager) GetToolNames() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.tools))
	for name := range l.tools {
		names = append(names, name)
	}
	return names
}

// GetToolCount Lấy số lượng tool
func (l *LocalMCPManager) GetToolCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.tools)
}

// Start Khởi động local manager (interface dự phòng)
func (l *LocalMCPManager) Start() error {
	log.Info("Local MCP manager đã khởi động")
	return nil
}

// Stop Dừng local manager (interface dự phòng)
func (l *LocalMCPManager) Stop() error {
	// Lưu ý: không xóa tool vì tool của local manager nên khả dụng trong suốt vòng đời ứng dụng
	// Nếu cần xóa tool, hãy gọi rõ method UnregisterTool
	log.Info("Local MCP manager đã dừng")
	return nil
}
