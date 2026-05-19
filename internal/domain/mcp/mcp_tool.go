package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var callRemoteMCPTool = func(ctx context.Context, cli *client.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return cli.CallTool(ctx, request)
}

var reconnectGlobalMCPServer = func(serverName string) (*client.Client, error) {
	return GetGlobalMCPManager().reconnectServer(serverName)
}

// LocalToolHandler kiểu hàm xử lý tool local
type LocalToolHandler func(ctx context.Context, argumentsInJSON string) (string, error)

// mcpTool triển khai MCP tool, hỗ trợ tool remote và local
type McpTool struct {
	info       *schema.ToolInfo
	originName string
	serverName string
	client     *client.Client

	// Hỗ trợ tool local
	isLocal      bool
	localHandler LocalToolHandler
}

// Info Lấy thông tin tool, triển khai interface BaseTool
func (t *McpTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *McpTool) callName() string {
	if t.originName != "" {
		return t.originName
	}
	if t.info != nil {
		return t.info.Name
	}
	return ""
}

func mcpToolMatchesName(invokable tool.InvokableTool, name string) bool {
	mcpTool, ok := invokable.(*McpTool)
	if !ok || mcpTool == nil {
		return false
	}
	if mcpTool.info != nil && mcpTool.info.Name == name {
		return true
	}
	return mcpTool.originName != "" && mcpTool.originName == name
}

func findInvokableToolByName(tools map[string]tool.InvokableTool, name string) (tool.InvokableTool, bool) {
	if invokable, ok := tools[name]; ok {
		return invokable, true
	}
	for _, invokable := range tools {
		if mcpToolMatchesName(invokable, name) {
			return invokable, true
		}
	}
	return nil, false
}

func remoteCallNameForTool(invokable tool.InvokableTool, fallback string) string {
	if mcpTool, ok := invokable.(*McpTool); ok && mcpTool != nil {
		if name := mcpTool.callName(); name != "" {
			return name
		}
	}
	return fallback
}

func (t *McpTool) InvokeableLocalRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	toolInfo := t.info
	if t.localHandler == nil {
		return "", fmt.Errorf("Hàm xử lý của tool local %s chưa được định nghĩa", toolInfo.Name)
	}

	log.Infof("Thực thi tool local: %s, tham số: %s", toolInfo.Name, argumentsInJSON)

	resultStr, err := t.localHandler(ctx, argumentsInJSON)
	if err != nil {
		log.Errorf("Tool local %s thực thi thất bại: %v", toolInfo.Name, err)
		return "", fmt.Errorf("Thực thi tool local thất bại: %v", err)
	}
	if len(resultStr) > 2048 {
		log.Infof("Tool local %s thực thi thành công, độ dài kết quả: %d", toolInfo.Name, len(resultStr))
	} else {
		log.Infof("Tool local %s thực thi thành công, kết quả: %+s", toolInfo.Name, resultStr)
	}

	return resultStr, nil
}

// InvokableRun Gọi tool, triển khai interface InvokableTool
func (t *McpTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Nếu là tool local, gọi trực tiếp hàm xử lý local
	if t.isLocal {
		return t.InvokeableLocalRun(ctx, argumentsInJSON, opts...)
	}

	retContent := ""

	// Logic gọi tool MCP remote
	// Kiểm tra client có khả dụng không
	if t.client == nil {
		return retContent, fmt.Errorf("Gọi MCP tool thất bại: MCP client chưa được khởi tạo")
	}

	// Parse tham số
	var arguments map[string]interface{}
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &arguments); err != nil {
			return retContent, fmt.Errorf("Parse tham số tool thất bại: %v", err)
		}
	}

	// Chuẩn bị request gọi
	toolName := t.callName()
	callRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}

	result, err := callRemoteMCPTool(ctx, t.client, callRequest)
	if err != nil {
		if !isRetryableRemoteCallError(err) {
			return retContent, fmt.Errorf("Gọi tool thất bại: %v", err)
		}

		log.Warnf("Tool %s gọi thất bại, chuẩn bị reconnect server %s rồi thử lại: %v", t.info.Name, t.serverName, err)

		newClient, reconnectErr := reconnectGlobalMCPServer(t.serverName)
		if reconnectErr != nil {
			return retContent, fmt.Errorf("Gọi tool thất bại: %v, và reconnect server thất bại: %v", err, reconnectErr)
		}

		t.client = newClient
		result, err = callRemoteMCPTool(ctx, t.client, callRequest)
		if err != nil {
			return retContent, fmt.Errorf("Gọi lại sau reconnect vẫn thất bại: %v", err)
		}
	}

	if err != nil {
		return retContent, fmt.Errorf("Gọi tool thất bại: %v", err)
	}

	resultStr, err := result.MarshalJSON()
	if err != nil {
		return retContent, fmt.Errorf("Chuyển đổi nội dung trả về từ tool call thất bại: %v", err)
	}

	return string(resultStr), nil
}

func (t *McpTool) GetClient() *client.Client {
	return t.client
}

func (t *McpTool) GetServerName() string {
	return t.serverName
}
