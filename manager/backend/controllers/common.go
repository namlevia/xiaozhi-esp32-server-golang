package controllers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WebSocketControllerInterface định nghĩa interface của WebSocket controller.
type WebSocketControllerInterface interface {
	RequestMcpToolDetailsFromClient(ctx context.Context, agentID string) ([]MCPTool, error)
}

// GetAgentMcpToolsCommon lấy danh sách công cụ MCP của trợ lý dùng chung cho quản trị viên và người dùng thường.
func GetAgentMcpToolsCommon(
	c *gin.Context,
	agentID string,
	webSocketController WebSocketControllerInterface,
	agentValidator func(agentID string) error,
) {
	log.Printf("GetAgentMcpToolsCommon bắt đầu xử lý, agentID: %s", agentID)

	if agentID == "" {
		log.Printf("Lỗi: tham số agent_id trống")
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}

	if err := agentValidator(agentID); err != nil {
		log.Printf("Xác thực trợ lý thất bại: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Xác thực trợ lý thành công, bắt đầu kiểm tra WebSocket controller")

	if webSocketController == nil {
		log.Printf("WebSocket controller chưa khởi tạo, trả về danh sách công cụ trống")
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	log.Printf("WebSocket controller tồn tại, bắt đầu yêu cầu danh sách công cụ MCP")

	ctx := context.Background()

	tools, err := webSocketController.RequestMcpToolDetailsFromClient(ctx, agentID)
	if err != nil {
		log.Printf("Lấy danh sách công cụ MCP thất bại: %v", err)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	log.Printf("Lấy danh sách công cụ MCP thành công: count=%d", len(tools))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": tools}})
}
