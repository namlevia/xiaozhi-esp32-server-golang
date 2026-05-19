package websocket

import (
	"net/http"
	"strings"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/golang-jwt/jwt/v4"
)

// MCPClaims là cấu trúc JWT claims
type MCPClaims struct {
	UserID     uint   `json:"userId"`
	AgentID    string `json:"agentId"`
	EndpointID string `json:"endpointId"`
	Purpose    string `json:"purpose"`
	jwt.RegisteredClaims
}

// handleMCPWebSocket xử lý kết nối MCP WebSocket
func (s *WebSocketServer) handleMCPWebSocket(w http.ResponseWriter, r *http.Request) {
	var agentId string

	// Trước tiên thử lấy token từ tham số URL
	token := r.URL.Query().Get("token")
	if token != "" {
		// Parse device ID từ token
		claims, err := s.parseMCPToken(token)
		if err != nil {
			log.Warnf("Parse token thất bại: %v", err)
			http.Error(w, "Token không hợp lệ", http.StatusUnauthorized)
			return
		}
		log.Infof("Parse token thành công: %v", claims)

		agentId = claims.AgentID
	} else {
		log.Errorf("Thiếu token")
		return
	}

	log.Infof("Nhận yêu cầu kết nối WebSocket từ MCP server, Agent ID: %s", agentId)

	// Upgrade kết nối WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Upgrade kết nối WebSocket thất bại: %v", err)
		return
	}

	mcpClientSession := mcp.GetDeviceMcpClient(agentId)
	if mcpClientSession == nil {
		mcpClientSession = mcp.NewDeviceMCPSession(agentId)
		mcp.AddDeviceMcpClient(agentId, mcpClientSession)
	}

	// Tạo MCP client
	mcpClient := mcp.NewWsEndPointMcpClient(mcpClientSession.Ctx, agentId, conn)
	if mcpClient == nil {
		log.Errorf("Tạo MCP client thất bại")
		conn.Close()
		return
	}
	mcpClientSession.AddWsEndPointMcp(mcpClient)

	// Khi MCP server ngắt kết nối, dọn ws endpoint MCP client
	go func() {
		<-mcpClient.Ctx.Done()
		log.Infof("Kết nối MCP của server %s đã ngắt", mcpClient.GetServerName())
	}()

	log.Infof("Kết nối MCP của server %s đã thiết lập", mcpClient.GetServerName()) // todo
}

// parseMCPToken parse MCP JWT token
func (s *WebSocketServer) parseMCPToken(tokenString string) (*MCPClaims, error) {
	// Xóa prefix "Bearer "
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Dùng cùng secret với lúc tạo token
	jwtSecret := []byte(util.GetManagerEndpointAuthToken())

	token, err := jwt.ParseWithClaims(tokenString, &MCPClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MCPClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// handleMCPAPI xử lý request MCP REST API
func (s *WebSocketServer) handleMCPAPI(w http.ResponseWriter, r *http.Request) {
	// Lấy deviceId từ URL path
	// Định dạng URL: /xiaozhi/api/mcp/tools/{deviceId}
	path := strings.TrimPrefix(r.URL.Path, "/xiaozhi/api/mcp/tools/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Thiếu tham số ID thiết bị", http.StatusBadRequest)
		return
	}

	deviceID := strings.TrimSuffix(path, "/")
	if deviceID == "" {
		http.Error(w, "ID thiết bị không được để trống", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetDeviceTools(w, r, deviceID)
	default:
		http.Error(w, "Phương thức HTTP không được hỗ trợ", http.StatusMethodNotAllowed)
	}
}
