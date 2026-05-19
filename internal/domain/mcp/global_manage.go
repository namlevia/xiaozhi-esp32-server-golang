package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"

	log "xiaozhi-esp32-server-golang/logger"
)

const (
	globalMCPPingInterval                 = 60 * time.Second
	globalMCPPingTimeout                  = 5 * time.Second
	globalMCPPeriodicToolsRefreshInterval = 2 * time.Minute
)

// MCPServerConfig cấu hình MCP server
type MCPServerConfig struct {
	Name         string            `json:"name" mapstructure:"name"`
	Type         string            `json:"type" mapstructure:"type"`
	Url          string            `json:"url" mapstructure:"url"`
	SSEUrl       string            `json:"sse_url" mapstructure:"sse_url"` // Tương thích ngược với field sse_url
	Enabled      bool              `json:"enabled" mapstructure:"enabled"`
	Provider     string            `json:"provider,omitempty" mapstructure:"provider"`
	ServiceID    string            `json:"service_id,omitempty" mapstructure:"service_id"`
	AuthRef      string            `json:"auth_ref,omitempty" mapstructure:"auth_ref"`
	Headers      map[string]string `json:"headers,omitempty" mapstructure:"headers"`
	AllowedTools []string          `json:"allowed_tools,omitempty" mapstructure:"allowed_tools"`
}

// GlobalMCPManager Global MCP manager
type GlobalMCPManager struct {
	servers       map[string]*MCPServerConnection
	tools         map[string]tool.InvokableTool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	reconnectConf ReconnectConfig
	httpClient    *http.Client
}

// ReconnectConfig Cấu hình reconnect
type ReconnectConfig struct {
	Interval    time.Duration
	MaxAttempts int
}

// MCPServerConnection Kết nối MCP server
type MCPServerConnection struct {
	config        MCPServerConfig
	client        *client.Client
	tools         map[string]tool.InvokableTool
	connected     bool
	refreshing    bool
	refreshQueued bool
	mu            sync.RWMutex
	lastError     error
	retryCount    int
	lastPing      time.Time
	reconnecting  bool
	reconnectWait chan struct{}
}

var (
	globalManager *GlobalMCPManager
	once          sync.Once
)

var buildGlobalMCPTransport = buildMCPTransport

// GetGlobalMCPManager lấy singleton Global MCP manager
func GetGlobalMCPManager() *GlobalMCPManager {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalManager = &GlobalMCPManager{
			servers: make(map[string]*MCPServerConnection),
			tools:   make(map[string]tool.InvokableTool),
			ctx:     ctx,
			cancel:  cancel,
			reconnectConf: ReconnectConfig{
				Interval:    time.Duration(viper.GetInt("mcp.global.reconnect_interval")) * time.Second,
				MaxAttempts: viper.GetInt("mcp.global.max_reconnect_attempts"),
			},
			httpClient: &http.Client{
				Timeout: 600 * time.Second,
			},
		}
	})
	return globalManager
}

// Start khởi động Global MCP manager
func (g *GlobalMCPManager) Start() error {
	// Trong kịch bản hot reload: sau Stop thì ctx đã bị hủy, cần tạo lại để monitor và reconnect hoạt động bình thường sau khi khởi động lại
	if g.ctx != nil && g.ctx.Err() != nil {
		g.ctx, g.cancel = context.WithCancel(context.Background())
		g.reconnectConf = ReconnectConfig{
			Interval:    time.Duration(viper.GetInt("mcp.global.reconnect_interval")) * time.Second,
			MaxAttempts: viper.GetInt("mcp.global.max_reconnect_attempts"),
		}
	}

	// Kiểm tra cấu hình trước
	CheckMCPConfig()

	if !viper.GetBool("mcp.global.enabled") {
		log.Info("Global MCP manager đã bị tắt")
		return nil
	}

	var serverConfigs []MCPServerConfig
	if err := viper.UnmarshalKey("mcp.global.servers", &serverConfigs); err != nil {
		log.Errorf("Parse cấu hình MCP server thất bại: %v", err)
		return fmt.Errorf("Parse cấu hình MCP server thất bại: %v", err)
	}

	log.Infof("Đọc được %d cấu hình MCP server từ config", len(serverConfigs))

	// Ghi log chi tiết từng cấu hình server
	for i, config := range serverConfigs {
		log.Infof("MCP server[%d]: Type=%s, Name=%s, Url=%s, SSEUrl=%s, Enabled=%v",
			i+1, config.Type, config.Name, config.Url, config.SSEUrl, config.Enabled)
	}

	// Kết nối các server đã bật
	connectedCount := 0
	for _, config := range serverConfigs {
		if config.Enabled {
			if err := g.connectToServer(config); err != nil {
				log.Errorf("Kết nối tới MCP server %s thất bại: %v", config.Name, err)
			} else {
				connectedCount++
			}
		} else {
			log.Infof("MCP server %s đã bị tắt, bỏ qua kết nối", config.Name)
		}
	}

	log.Infof("Đã kết nối thành công %d MCP server", connectedCount)

	// Khởi động goroutine monitor
	go g.monitorConnections()

	log.Info("Global MCP manager đã khởi động")
	return nil
}

// Stop dừng Global MCP manager
func (g *GlobalMCPManager) Stop() error {
	g.cancel()

	g.mu.Lock()
	type serverEntry struct {
		name string
		conn *MCPServerConnection
	}
	servers := make([]serverEntry, 0, len(g.servers))
	for name, conn := range g.servers {
		if conn != nil {
			servers = append(servers, serverEntry{name: name, conn: conn})
		}
	}
	g.servers = make(map[string]*MCPServerConnection)
	g.tools = make(map[string]tool.InvokableTool)
	g.mu.Unlock()

	for _, server := range servers {
		if err := server.conn.disconnect(); err != nil {
			log.Errorf("Ngắt kết nối MCP server %s thất bại: %v", server.name, err)
		}
	}

	log.Info("Global MCP manager đã dừng")
	return nil
}

// createFailedConnection Tạo object kết nối thất bại để reconnect sau
func (g *GlobalMCPManager) createFailedConnection(config MCPServerConfig) {
	conn := &MCPServerConnection{
		config:     config,
		tools:      make(map[string]tool.InvokableTool),
		connected:  false,
		lastError:  fmt.Errorf("Khởi tạo kết nối thất bại"),
		retryCount: 0,
	}

	g.mu.Lock()
	g.servers[config.Name] = conn
	g.mu.Unlock()

	log.Infof("Đã tạo object kết nối cho MCP server thất bại: %s", config.Name)
}

// connectToServer Kết nối tới MCP server
func (g *GlobalMCPManager) connectToServer(config MCPServerConfig) error {
	// Xác thực cấu hình
	if config.Name == "" {
		return fmt.Errorf("Tên MCP server không được rỗng")
	}

	if !config.Enabled {
		log.Infof("MCP server %s đã bị tắt, bỏ qua kết nối", config.Name)
		return nil
	}

	_, endpoint, endpointErr := endpointForConfig(config)
	if endpointErr != nil {
		return endpointErr
	}
	log.Infof("Đang kết nối MCP server: %s (URL: %s)", config.Name, endpoint)

	conn := &MCPServerConnection{
		config: config,
		tools:  make(map[string]tool.InvokableTool),
	}

	g.mu.Lock()
	g.servers[config.Name] = conn
	g.mu.Unlock()

	// Kết nối tới server
	if err := conn.connect(); err != nil {
		return fmt.Errorf("Kết nối MCP server thất bại: %v", err)
	}

	log.Infof("Đã kết nối tới MCP server: %s", config.Name)
	return nil
}

// connect Kết nối tới MCP server
func (conn *MCPServerConnection) connect() (retErr error) {
	// Dùng background context, không đặt timeout để giữ kết nối SSE lâu dài
	ctx := context.Background()

	transportInstance, endpoint, err := buildGlobalMCPTransport(conn.config)
	if err != nil {
		return err
	}

	// Dùng client.NewClient để tạo MCP client
	mcpClient := client.NewClient(transportInstance)
	serverName := conn.config.Name
	defer func() {
		if retErr == nil {
			return
		}

		conn.mu.Lock()
		conn.client = nil
		conn.connected = false
		conn.refreshing = false
		conn.refreshQueued = false
		conn.tools = make(map[string]tool.InvokableTool)
		conn.lastError = retErr
		conn.mu.Unlock()

		if globalManager != nil {
			globalManager.removeGlobalTools(serverName)
		}

		if closeErr := mcpClient.Close(); closeErr != nil {
			log.Errorf("Đóng MCP client thất bại: %v", closeErr)
		}
	}()

	mcpClient.OnNotification(conn.handleJSONRPCNotification)
	conn.mu.Lock()
	conn.client = mcpClient
	conn.mu.Unlock()

	log.Infof("Bắt đầu kết nối MCP server: %s, %s URL: %s", conn.config.Name, conn.config.Type, endpoint)

	// Khởi động client
	if err := mcpClient.Start(ctx); err != nil {
		log.Errorf("Khởi động MCP client thất bại, server: %s, lỗi: %v", conn.config.Name, err)
		retErr = fmt.Errorf("Khởi động client thất bại: %v", err)
		return retErr
	}

	log.Infof("MCP client khởi động thành công: %s", conn.config.Name)

	// Khởi tạo client
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "xiaozhi-esp32-server",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{
				Experimental: make(map[string]any),
			},
		},
	}

	log.Infof("Đang khởi tạo MCP server: %s", conn.config.Name)
	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Errorf("Khởi tạo MCP server thất bại, server: %s, lỗi: %v", conn.config.Name, err)
		retErr = fmt.Errorf("Khởi tạo thất bại: %v", err)
		return retErr
	}

	log.Infof("MCP server khởi tạo thành công: %s, kết quả: %+v", conn.config.Name, initResult)

	// Lấy danh sách tool
	if refreshErr := conn.refreshTools(ctx); refreshErr != nil {
		log.Errorf("Lấy danh sách tool thất bại: %v", refreshErr)
		retErr = fmt.Errorf("Lấy danh sách tool thất bại: %v", refreshErr)
		return retErr
	}

	conn.mu.Lock()
	conn.connected = true
	conn.lastError = nil
	conn.retryCount = 0
	conn.mu.Unlock()

	log.Infof("Kết nối MCP server đã thiết lập xong: %s", conn.config.Name)
	return nil
}

func (conn *MCPServerConnection) handleJSONRPCNotification(notification mcp.JSONRPCNotification) {
	switch notification.Method {
	case mcp.MethodNotificationToolsListChanged, "notifications/tools/updated":
		log.Infof("MCP server %s nhận notification cập nhật danh sách tool, chuẩn bị refresh danh sách tool", conn.config.Name)
		conn.scheduleToolsRefresh()
	}
}

func (conn *MCPServerConnection) scheduleToolsRefresh() {
	conn.scheduleToolsRefreshWithReason("theo notification")
}

func (conn *MCPServerConnection) schedulePeriodicToolsRefresh() {
	conn.scheduleToolsRefreshWithReason("định kỳ")
}

func (conn *MCPServerConnection) scheduleToolsRefreshWithReason(reason string) {
	conn.mu.Lock()
	if conn.refreshing {
		conn.refreshQueued = true
		conn.mu.Unlock()
		return
	}
	conn.refreshing = true
	conn.mu.Unlock()

	go conn.runScheduledToolsRefresh(reason)
}

func (conn *MCPServerConnection) runScheduledToolsRefresh(reason string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := conn.refreshTools(ctx)
		cancel()
		if err != nil {
			log.Warnf("MCP server %s refresh danh sách tool %s thất bại: %v", conn.config.Name, reason, err)
		}

		conn.mu.Lock()
		if err != nil {
			conn.lastError = err
		} else {
			conn.lastError = nil
		}

		if conn.refreshQueued {
			conn.refreshQueued = false
			conn.mu.Unlock()
			continue
		}

		conn.refreshing = false
		conn.mu.Unlock()
		return
	}
}

func normalizeMCPTransportType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "sse":
		return "sse"
	case "streamable_http", "streamable-http", "http":
		return "streamablehttp"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

func endpointForConfig(config MCPServerConfig) (string, string, error) {
	transportType := normalizeMCPTransportType(config.Type)
	if transportType == "" {
		if strings.TrimSpace(config.SSEUrl) != "" {
			transportType = "sse"
		} else if strings.TrimSpace(config.Url) != "" {
			transportType = "streamablehttp"
		}
	}

	switch transportType {
	case "sse":
		if strings.TrimSpace(config.SSEUrl) != "" {
			return transportType, strings.TrimSpace(config.SSEUrl), nil
		}
		if strings.TrimSpace(config.Url) != "" {
			return transportType, strings.TrimSpace(config.Url), nil
		}
		return "", "", fmt.Errorf("MCP server %s thiếu SSE URL", config.Name)
	case "streamablehttp":
		if strings.TrimSpace(config.Url) != "" {
			return transportType, strings.TrimSpace(config.Url), nil
		}
		if strings.TrimSpace(config.SSEUrl) != "" {
			return transportType, strings.TrimSpace(config.SSEUrl), nil
		}
		return "", "", fmt.Errorf("MCP server %s thiếu StreamableHTTP URL", config.Name)
	default:
		return "", "", fmt.Errorf("MCP server %s không hỗ trợ type: %s", config.Name, config.Type)
	}
}

func buildMCPTransport(config MCPServerConfig) (transport.Interface, string, error) {
	transportType, endpoint, err := endpointForConfig(config)
	if err != nil {
		return nil, "", err
	}

	headers := make(map[string]string)
	for k, v := range config.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		headers[strings.TrimSpace(k)] = v
	}

	switch transportType {
	case "sse":
		opts := make([]transport.ClientOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHeaders(headers))
		}
		sseTransport, err := transport.NewSSE(endpoint, opts...)
		if err != nil {
			return nil, "", fmt.Errorf("Tạo tầng transport SSE thất bại: %v", err)
		}
		return sseTransport, endpoint, nil
	case "streamablehttp":
		opts := make([]transport.StreamableHTTPCOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(headers))
		}
		httpTransport, err := transport.NewStreamableHTTP(endpoint, opts...)
		if err != nil {
			return nil, "", fmt.Errorf("Tạo tầng transport StreamableHTTP thất bại: %v", err)
		}
		return httpTransport, endpoint, nil
	default:
		return nil, "", fmt.Errorf("Không hỗ trợ loại transport MCP: %s", transportType)
	}
}

func buildAllowedToolSet(allowedTools []string) map[string]struct{} {
	if len(allowedTools) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(allowedTools))
	for _, toolName := range allowedTools {
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			continue
		}
		set[toolName] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func filterMCPToolsByAllowList(tools []mcp.Tool, allowedTools []string) []mcp.Tool {
	allowedSet := buildAllowedToolSet(allowedTools)
	if len(allowedSet) == 0 {
		return tools
	}

	filtered := make([]mcp.Tool, 0, len(tools))
	for _, item := range tools {
		if _, ok := allowedSet[strings.TrimSpace(item.Name)]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// refreshTools Refresh danh sách tool
func (conn *MCPServerConnection) refreshTools(ctx context.Context) error {
	conn.mu.RLock()
	serverName := conn.config.Name
	allowedTools := append([]string(nil), conn.config.AllowedTools...)
	mcpClient := conn.client
	conn.mu.RUnlock()
	if mcpClient == nil {
		return fmt.Errorf("MCP client chưa được khởi tạo")
	}

	// Lấy danh sách tool
	listRequest := mcp.ListToolsRequest{}
	toolsResult, err := mcpClient.ListTools(ctx, listRequest)
	if err != nil {
		return fmt.Errorf("Lấy danh sách tool thất bại: %v", err)
	}

	tools := filterMCPToolsByAllowList(toolsResult.Tools, allowedTools)
	convertedTools := ConvertMcpToolListToInvokableToolList(tools, serverName, mcpClient)

	conn.mu.Lock()
	conn.tools = convertedTools
	conn.mu.Unlock()

	// Cập nhật bảng tool global bên ngoài conn.mu để tránh đảo thứ tự lock với g.mu.
	globalManager.updateGlobalTools(serverName, convertedTools)

	log.Infof("Danh sách tool của MCP server %s đã cập nhật, tổng %d tool", serverName, len(convertedTools))
	return nil
}

func ConvertMcpToolListToInvokableToolList(tools []mcp.Tool, serverName string, client *client.Client) map[string]tool.InvokableTool {
	invokeTools := make(map[string]tool.InvokableTool)
	usedNames := make(map[string]string, len(tools))
	for _, tool := range tools {
		originName := tool.Name
		if strings.TrimSpace(originName) == "" {
			log.Warnf("Bỏ qua MCP tool có tên rỗng, server=%s", serverName)
			continue
		}
		llmName := uniqueLLMToolName(sanitizeLLMToolName(originName), originName, usedNames)
		if llmName != originName {
			log.Debugf("Tên MCP tool %q không đúng chuẩn tên tool OpenAI, đã chuyển thành %q, server=%s", originName, llmName, serverName)
		}

		marshaledInputSchema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			log.Errorf("convert mcp tool to invokeable tool err: %+v", err)
			continue
		}
		inputSchema := &openapi3.Schema{}
		err = json.Unmarshal(marshaledInputSchema, inputSchema)
		if err != nil {
			log.Errorf("convert mcp tool to invokeable tool err: %+v", err)
			continue
		}

		mcpToolInstance := &McpTool{
			info: &schema.ToolInfo{
				Name:        llmName,
				Desc:        tool.Description,
				ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(inputSchema),
			},
			originName: originName,
			serverName: serverName,
			client:     client,
		}
		invokeTools[llmName] = mcpToolInstance
	}
	return invokeTools
}

// disconnect Ngắt kết nối
func (conn *MCPServerConnection) disconnect() error {
	conn.mu.Lock()
	serverName := conn.config.Name
	mcpClient := conn.client
	conn.client = nil
	conn.connected = false
	conn.tools = make(map[string]tool.InvokableTool)
	conn.mu.Unlock()

	if globalManager != nil {
		globalManager.removeGlobalTools(serverName)
	}

	if mcpClient != nil {
		// Đóng client bên ngoài lock để tránh khóa fast path.
		if err := mcpClient.Close(); err != nil {
			log.Errorf("Đóng MCP client thất bại: %v", err)
		}
	}

	return nil
}

func (g *GlobalMCPManager) removeGlobalTools(serverName string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for name, mcpToolInterface := range g.tools {
		if mt, ok := mcpToolInterface.(*McpTool); ok && mt.serverName == serverName {
			delete(g.tools, name)
		}
	}
}

// updateGlobalTools Cập nhật danh sách tool global
func (g *GlobalMCPManager) updateGlobalTools(serverName string, tools map[string]tool.InvokableTool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Xóa tool cũ của server này
	for name, mcpToolInterface := range g.tools {
		if mt, ok := mcpToolInterface.(*McpTool); ok && mt.serverName == serverName {
			delete(g.tools, name)
		}
	}

	// Thêm tool mới
	for name, mcpToolInterface := range tools {
		g.tools[fmt.Sprintf("%s_%s", serverName, name)] = mcpToolInterface
	}
}

// GetAllTools Lấy tất cả tool khả dụng
func (g *GlobalMCPManager) GetAllTools() map[string]tool.InvokableTool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]tool.InvokableTool)
	for name, mcpToolInterface := range g.tools {
		result[name] = mcpToolInterface
	}
	return result
}

// GetToolByName Lấy tool theo tên
func (g *GlobalMCPManager) GetToolByName(name string) (tool.InvokableTool, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if invokable, exists := g.tools[name]; exists {
		return invokable, true
	}

	var matched tool.InvokableTool
	matchCount := 0
	for _, invokable := range g.tools {
		if !mcpToolMatchesName(invokable, name) {
			continue
		}
		matchCount++
		if matchCount == 1 {
			matched = invokable
			continue
		}

		log.Warnf("Tên tool MCP global %s có nhiều provider trùng tên, vui lòng chỉ định rõ tên server", name)
		return nil, false
	}
	return matched, matchCount == 1
}

func GetServerClientByName(serverName string) *client.Client {
	return GetGlobalMCPManager().GetServerClientByName(serverName)
}

func (g *GlobalMCPManager) GetServerClientByName(serverName string) *client.Client {
	g.mu.RLock()
	conn, ok := g.servers[serverName]
	g.mu.RUnlock()
	if !ok || conn == nil {
		return nil
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.client
}

func GetServerEndpointSnapshotByName(serverName string) string {
	return GetGlobalMCPManager().GetServerEndpointSnapshotByName(serverName)
}

func (g *GlobalMCPManager) GetServerEndpointSnapshotByName(serverName string) string {
	g.mu.RLock()
	conn, ok := g.servers[serverName]
	g.mu.RUnlock()
	if !ok || conn == nil {
		return ""
	}

	conn.mu.RLock()
	config := conn.config
	conn.mu.RUnlock()

	_, endpoint, err := endpointForConfig(config)
	if err != nil {
		if strings.TrimSpace(config.Url) != "" {
			return strings.TrimSpace(config.Url)
		}
		return strings.TrimSpace(config.SSEUrl)
	}
	return endpoint
}

func ReconnectServerByName(serverName string) (*client.Client, error) {
	return GetGlobalMCPManager().reconnectServer(serverName)
}

// isSessionClosedError Kiểm tra có phải lỗi session closed không
func isSessionClosedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "session closed")
}

func isRetryableRemoteCallError(err error) bool {
	if err == nil {
		return false
	}
	if isSessionClosedError(err) {
		return true
	}

	message := strings.ToLower(err.Error())
	retryableIndicators := []string{
		"unexpected end of json input",
		"invalid character",
		"eof",
		"broken pipe",
		"connection reset",
		"connection refused",
		"connection aborted",
		"timeout",
		"bad gateway",
		"502",
		"temporarily unavailable",
	}
	for _, indicator := range retryableIndicators {
		if strings.Contains(message, indicator) {
			return true
		}
	}
	return false
}

func (g *GlobalMCPManager) schedulePeriodicToolsRefresh() {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, conn := range g.servers {
		if conn == nil {
			continue
		}

		conn.mu.RLock()
		connected := conn.connected
		hasClient := conn.client != nil
		conn.mu.RUnlock()
		if !connected || !hasClient {
			continue
		}

		conn.schedulePeriodicToolsRefresh()
	}
}

// monitorConnections Monitor trạng thái kết nối
func (g *GlobalMCPManager) monitorConnections() {
	pingTicker := time.NewTicker(globalMCPPingInterval) // ping mỗi 60 giây
	defer pingTicker.Stop()
	toolsRefreshTicker := time.NewTicker(globalMCPPeriodicToolsRefreshInterval)
	defer toolsRefreshTicker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-pingTicker.C:
			// Thực hiện kiểm tra ping
			g.mu.RLock()
			for name, conn := range g.servers {
				go func(name string, conn *MCPServerConnection) {
					ctx, cancel := context.WithTimeout(context.Background(), globalMCPPingTimeout)
					defer cancel()

					if err := conn.ping(ctx); err != nil {
						log.Warnf("MCP server %s ping thất bại, bắt đầu reconnect: %v", name, err)
						// Khi ping thất bại, đánh dấu disconnected và kích hoạt reconnect ngay
						conn.mu.Lock()
						conn.connected = false
						conn.lastError = err
						conn.mu.Unlock()

						// Kích hoạt reconnect trực tiếp
						go g.reconnectServer(name)
					} else {
						//log.Debugf("MCP server %s ping thành công", name)
					}
				}(name, conn)
			}
			g.mu.RUnlock()
		case <-toolsRefreshTicker.C:
			g.schedulePeriodicToolsRefresh()
		}
	}
}

// reconnectServer Reconnect server và trả về client mới
func (g *GlobalMCPManager) reconnectServer(serverName string) (*client.Client, error) {
	g.mu.RLock()
	var conn *MCPServerConnection
	for _, c := range g.servers {
		if c.config.Name == serverName {
			conn = c
			break
		}
	}
	g.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("Không tìm thấy kết nối server: %s", serverName)
	}

	conn.mu.Lock()
	if conn.reconnecting {
		wait := conn.reconnectWait
		conn.mu.Unlock()
		if wait != nil {
			<-wait
		}

		conn.mu.RLock()
		mcpClient := conn.client
		connected := conn.connected
		lastErr := conn.lastError
		conn.mu.RUnlock()
		if mcpClient != nil && connected {
			return mcpClient, nil
		}
		if lastErr != nil {
			return nil, fmt.Errorf("Reconnect thất bại: %v", lastErr)
		}
		return nil, fmt.Errorf("Reconnect thất bại: client chưa sẵn sàng")
	}
	wait := make(chan struct{})
	conn.reconnecting = true
	conn.reconnectWait = wait
	conn.mu.Unlock()

	defer func() {
		conn.mu.Lock()
		conn.reconnecting = false
		if conn.reconnectWait == wait {
			close(wait)
			conn.reconnectWait = nil
		}
		conn.mu.Unlock()
	}()

	// Ngắt kết nối
	if err := conn.disconnect(); err != nil {
		log.Errorf("Ngắt kết nối thất bại: %v", err)
	}

	// Chờ một lúc ngắn để đảm bảo tài nguyên được giải phóng
	time.Sleep(time.Second)

	// Kết nối lại
	if err := conn.connect(); err != nil {
		conn.mu.Lock()
		conn.lastError = err
		conn.mu.Unlock()
		return nil, fmt.Errorf("Reconnect thất bại: %v", err)
	}

	conn.mu.RLock()
	mcpClient := conn.client
	conn.mu.RUnlock()
	return mcpClient, nil
}

// ping Gửi request ping để kiểm tra trạng thái kết nối
func (conn *MCPServerConnection) ping(ctx context.Context) error {
	conn.mu.RLock()
	mcpClient := conn.client
	conn.mu.RUnlock()
	if mcpClient == nil {
		return fmt.Errorf("client chưa được khởi tạo")
	}

	// Dùng request Ping rỗng làm ping
	err := mcpClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping thất bại: %v", err)
	}

	conn.mu.Lock()
	conn.lastPing = time.Now()
	conn.mu.Unlock()

	return nil
}
