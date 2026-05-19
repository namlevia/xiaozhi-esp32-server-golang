package llm

import (
	"context"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
)

// ConvertMCPToolsToEinoTools chuyển công cụ MCP sang định dạng Eino ToolInfo.
func ConvertMCPToolsToEinoTools(ctx context.Context, mcpTools map[string]interface{}) ([]*schema.ToolInfo, error) {
	var einoTools []*schema.ToolInfo

	for toolName, mcpTool := range mcpTools {
		// Thử lấy thông tin công cụ
		if invokableTool, ok := mcpTool.(interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}); ok {
			toolInfo, err := invokableTool.Info(ctx)
			if err != nil {
				log.Errorf("Lấy thông tin công cụ %s thất bại: %v", toolName, err)
				continue
			}
			einoTools = append(einoTools, toolInfo)
		} else {
			log.Warnf("Công cụ %s không hỗ trợ interface Info, bỏ qua chuyển đổi", toolName)
		}
	}

	log.Infof("Đã chuyển đổi thành công %d công cụ MCP thành công cụ Eino", len(einoTools))
	return einoTools, nil
}
