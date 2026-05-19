# Mô tả luồng kết nối WebSocket

## Tổng quan

Tài liệu này mô tả luồng kết nối và giao tiếp WebSocket giữa `internal/domain/config/manager/websocket_client.go` và `websocket.go`.

## Thiết kế kiến trúc

### Định nghĩa vai trò

1. **`internal/domain/config/manager/websocket_client.go`** - WebSocket client của server chính
   - Đóng vai trò client kết nối tới Manager Backend
   - Có thể gửi request và nhận response
   - Hỗ trợ giao tiếp hai chiều

2. **`websocket.go`** - WebSocket server của Manager Backend
   - Đóng vai trò server nhận kết nối WebSocket từ server chính
   - Xử lý request do server chính gửi lên
   - **Chỉ giữ kết nối hợp lệ cuối cùng**; kết nối mới sẽ ngắt kết nối cũ
   - Hỗ trợ chủ động đẩy message

### Luồng kết nối

```text
Server chính (internal/domain/config/manager/websocket_client.go)  →  Manager Backend (websocket.go)
        client                                      server (một kết nối)
```

## Luồng chi tiết

### 1. Thiết lập kết nối

#### Manager Backend khởi động WebSocket server

```go
// Trong websocket.go
controller := NewWebSocketController(db)
// Đăng ký trong router
router.GET("/ws", controller.HandleWebSocket)
```

#### Server chính kết nối tới Manager Backend

```go
// Trong internal/domain/config/manager/websocket_client.go
client := manager.NewWebSocketClient()
err := client.Connect(ctx)
```

Định dạng URL kết nối:

- Nếu cấu hình là `http://localhost:8080`
- Kết nối thực tế sẽ là `ws://localhost:8080/ws`

**Quan trọng**: nếu có request kết nối mới, Manager Backend sẽ tự động ngắt kết nối hiện tại và chỉ giữ kết nối mới nhất.

### 2. Luồng request danh sách tool

#### Server chính request danh sách MCP tool

```go
// Trong internal/domain/config/manager/websocket_client.go
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "some_agent_id",
})
```

#### Manager Backend xử lý request

```go
// Trong websocket.go
func (client *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
    agentID := request.Body["agent_id"].(string)
    
    // Logic lấy danh sách tool
    response := map[string]interface{}{
        "agent_id": agentID,
        "tools":    []string{"tool1", "tool2", "tool3"},
        "count":    3,
    }
    
    client.sendResponse(request.ID, 200, response, "")
}
```

### 3. Hỗ trợ giao tiếp hai chiều

### Client → Server (chức năng hiện có)

#### Server chính request danh sách MCP tool

```go
// Trong internal/domain/config/manager/websocket_client.go
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "some_agent_id",
})
```

#### Manager Backend xử lý request

```go
// Trong websocket.go
func (client *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
    agentID := request.Body["agent_id"].(string)
    
    // Logic lấy danh sách tool
    response := map[string]interface{}{
        "agent_id": agentID,
        "tools":    []string{"tool1", "tool2", "tool3"},
        "count":    3,
    }
    
    client.sendResponse(request.ID, 200, response, "")
}
```

### Server → Client (chức năng mới)

#### Manager Backend chủ động request client

```go
// Trong websocket.go
func (ctrl *WebSocketController) RequestMcpToolsFromClient(ctx context.Context, agentID string) (*WebSocketResponse, error) {
    body := map[string]interface{}{
        "agent_id": agentID,
    }
    return ctrl.SendRequestToClient(ctx, "GET", "/api/mcp/tools", body)
}

// Request thông tin server từ client
func (ctrl *WebSocketController) RequestServerInfoFromClient(ctx context.Context) (*WebSocketResponse, error) {
    return ctrl.SendRequestToClient(ctx, "GET", "/api/server/info", nil)
}

// Request ping từ client
func (ctrl *WebSocketController) RequestPingFromClient(ctx context.Context) (*WebSocketResponse, error) {
    return ctrl.SendRequestToClient(ctx, "GET", "/api/server/ping", nil)
}
```

#### Client xử lý request từ server

```go
// Trong internal/domain/config/manager/websocket_client.go
client.SetRequestHandler(func(request *WebSocketRequest) {
    // Xử lý request nhận được
    switch request.Path {
    case "/api/mcp/tools":
        // Xử lý request danh sách MCP tool
        c.handleMcpToolListRequest(request)
    case "/api/server/info":
        // Xử lý request thông tin server
        c.handleServerInfoRequest(request)
    case "/api/server/ping":
        // Xử lý request ping
        c.handlePingRequest(request)
    }
})
```

### Ví dụ giao tiếp hai chiều hoàn chỉnh

```go
// 1. Client kết nối tới server
client := manager.NewWebSocketClient()
err := client.Connect(ctx)

// 2. Client thiết lập request handler
client.SetRequestHandler(func(request *WebSocketRequest) {
    // Xử lý request từ server
    // và gửi response
})

// 3. Client chủ động request server
response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
    "agent_id": "agent_123",
})

// 4. Server chủ động request client
serverResponse, err := websocketController.RequestMcpToolsFromClient(ctx, "agent_456")

// 5. Hoàn tất giao tiếp hai chiều
```

## Định dạng message

### Request message (WebSocketRequest)

```json
{
    "id": "uuid-string",
    "method": "GET",
    "path": "/api/mcp/tools",
    "body": {
        "agent_id": "agent_123"
    }
}
```

### Response message (WebSocketResponse)

```json
{
    "id": "uuid-string",
    "status": 200,
    "body": {
        "agent_id": "agent_123",
        "tools": ["tool1", "tool2", "tool3"],
        "count": 3
    },
    "error": ""
}
```

### Ping/Pong message

```json
// Ping
{"ping": 1640995200}

// Pong
{"pong": 1640995200}
```

## Quản lý kết nối

### Chiến lược một kết nối

- **Chỉ giữ kết nối hợp lệ cuối cùng**
- Kết nối mới sẽ tự động ngắt kết nối hiện tại
- Đơn giản hóa logic quản lý kết nối
- Phù hợp với kịch bản giao tiếp một-một

### Giám sát trạng thái kết nối

```go
// Kiểm tra có client đang kết nối hay không
func (ctrl *WebSocketController) HasConnectedClient() bool

// Lấy client đang kết nối hiện tại
func (ctrl *WebSocketController) GetCurrentClient() *WebSocketClient
```

### Logic chuyển đổi kết nối

```go
// Trong HandleWebSocket
if ctrl.currentClient != nil && ctrl.currentClient.isConnected {
    log.Printf("Ngắt kết nối hiện tại: %s", ctrl.currentClient.ID)
    ctrl.currentClient.conn.Close()
    ctrl.currentClient.isConnected = false
}

// Đặt kết nối mới làm client hiện tại
ctrl.currentClient = client
```

## Xử lý lỗi

### Lỗi kết nối

- Tự động kiểm tra heartbeat
- Tự động ngắt khi kết nối timeout
- Tự động dọn dẹp khi kết nối bất thường
- Kết nối mới tự động thay thế kết nối cũ

### Lỗi message

- Kiểm tra định dạng message
- Trả về response lỗi
- Ghi log

## Yêu cầu cấu hình

### Cấu hình server chính

```yaml
manager:
  backend_url: "http://localhost:8080"
```

### Cấu hình Manager Backend

```go
// Đăng ký endpoint WebSocket trong router
router.GET("/ws", websocketController.HandleWebSocket)
```

## Gợi ý kiểm thử

1. **Kiểm thử kết nối**
   - Xác minh WebSocket kết nối thành công
   - Kiểm thử kết nối mới ngắt kết nối cũ
   - Kiểm thử ngắt kết nối và reconnect

2. **Kiểm thử chức năng**
   - Kiểm thử request danh sách MCP tool
   - Xác minh giao tiếp hai chiều
   - Kiểm thử đẩy message

3. **Kiểm thử lỗi**
   - Reconnect sau khi mất mạng
   - Xử lý message không hợp lệ
   - Xử lý timeout
   - Heartbeat timeout
   - Chuyển đổi kết nối

## Lưu ý

1. **Giới hạn một kết nối**
   - Chỉ có một kết nối active tại cùng thời điểm
   - Kết nối mới sẽ buộc kết nối cũ ngắt
   - Phù hợp với kiến trúc master-slave, không phù hợp với kịch bản nhiều client

2. **An toàn đồng thời**
   - Dùng read-write lock để bảo vệ tham chiếu client hiện tại
   - Chuyển đổi client an toàn
   - Gửi message an toàn theo luồng

3. **Quản lý tài nguyên**
   - Dọn dẹp kịp thời các kết nối đã ngắt
   - Đóng WebSocket connection đúng cách
   - Tránh rò rỉ bộ nhớ

4. **Cơ chế heartbeat**
   - Gửi ping mỗi 30 giây
   - Tự động ngắt nếu không có response trong 60 giây
   - Hỗ trợ message ping/pong

5. **Ghi log**
   - Ghi lại thay đổi trạng thái kết nối
   - Ghi lại quá trình chuyển đổi kết nối
   - Ghi lại thông tin request và response
   - Ghi lại lỗi và tình huống bất thường

## Ví dụ sử dụng hoàn chỉnh

### Code kiểm thử giao tiếp hai chiều

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "xiaozhi-esp32-server-golang/internal/domain/config/manager"
)

func main() {
    ctx := context.Background()
    
    // 1. Tạo client và kết nối
    client := manager.NewWebSocketClient()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("Kết nối thất bại: %v", err)
    }
    defer client.Disconnect()
    
    // 2. Thiết lập request handler để xử lý request từ server
    client.SetRequestHandler(func(request *manager.WebSocketRequest) {
        log.Printf("Nhận request từ server: %s %s", request.Method, request.Path)
        
        switch request.Path {
        case "/api/mcp/tools":
            // Xử lý request danh sách MCP tool
            agentID := ""
            if request.Body != nil {
                if id, ok := request.Body["agent_id"].(string); ok {
                    agentID = id
                }
            }
            
            response := map[string]interface{}{
                "agent_id": agentID,
                "tools":    []string{"client_tool_1", "client_tool_2"},
                "count":    2,
            }
            
            client.SendResponse(request.ID, 200, response, "")
            
        case "/api/server/info":
            response := map[string]interface{}{
                "server_name": "xiaozhi-client",
                "version":     "1.0.0",
                "uptime":      time.Now().Format(time.RFC3339),
            }
            client.SendResponse(request.ID, 200, response, "")
            
        case "/api/server/ping":
            response := map[string]interface{}{
                "message": "pong from client",
                "time":    time.Now().Format(time.RFC3339),
            }
            client.SendResponse(request.ID, 200, response, "")
        }
    })
    
    // 3. Client chủ động request server
    fmt.Println("=== Client request server ===")
    response, err := client.SendRequest(ctx, "GET", "/api/mcp/tools", map[string]interface{}{
        "agent_id": "client_agent_123",
    })
    if err != nil {
        log.Printf("Client request thất bại: %v", err)
    } else {
        fmt.Printf("Response từ server: %+v\n", response)
    }
    
    // 4. Đợi một khoảng thời gian để server có cơ hội gửi request
    fmt.Println("Đang đợi request từ server...")
    time.Sleep(5 * time.Second)
    
    fmt.Println("Kiểm thử giao tiếp hai chiều hoàn tất")
}
```

### Code kiểm thử phía server

```go
// Trong Manager Backend
func testBidirectionalCommunication() {
    ctx := context.Background()
    
    // 1. Kiểm tra trạng thái kết nối client
    status := websocketController.GetClientConnectionStatus()
    fmt.Printf("Trạng thái client: %+v\n", status)
    
    // 2. Server chủ động request client
    fmt.Println("=== Server request client ===")
    
    // Request danh sách MCP tool
    response, err := websocketController.RequestMcpToolsFromClient(ctx, "server_agent_456")
    if err != nil {
        log.Printf("Request danh sách MCP tool thất bại: %v", err)
    } else {
        fmt.Printf("Response MCP tool từ client: %+v\n", response)
    }
    
    // Request thông tin server
    infoResponse, err := websocketController.RequestServerInfoFromClient(ctx)
    if err != nil {
        log.Printf("Request thông tin server thất bại: %v", err)
    } else {
        fmt.Printf("Thông tin server từ client: %+v\n", infoResponse)
    }
    
    // Request ping
    pingResponse, err := websocketController.RequestPingFromClient(ctx)
    if err != nil {
        log.Printf("Request ping thất bại: %v", err)
    } else {
        fmt.Printf("Response ping từ client: %+v\n", pingResponse)
    }
}
```

## Lưu ý bổ sung

1. **Yêu cầu giao tiếp hai chiều**
   - Client phải thiết lập request handler
   - Server và client đều phải triển khai các phương thức xử lý request tương ứng
   - Request ID phải khớp để response được định tuyến chính xác

2. **Xử lý lỗi**
   - Giao tiếp hai chiều sẽ thất bại khi mất kết nối mạng
   - Xử lý timeout rất quan trọng
   - Bắt buộc kiểm tra trạng thái kết nối

3. **Cân nhắc hiệu năng**
   - Tránh request hai chiều quá thường xuyên
   - Thiết lập timeout hợp lý
   - Giám sát trạng thái kết nối
