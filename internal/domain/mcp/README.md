# MCP Host implementation

Package này triển khai MCP (Model Context Protocol) Host dựa trên Eino, hỗ trợ quản lý tool ở cấp global và cấp thiết bị.

## Tính năng

### Quản lý MCP tool global

- Kết nối tới nhiều MCP server qua SSE.
- Tự động discover và đăng ký tool.
- Theo dõi trạng thái kết nối và tự động reconnect.
- Proxy lời gọi tool.

### Quản lý MCP theo thiết bị

- Mỗi thiết bị có kết nối MCP riêng.
- Hỗ trợ lấy danh sách tool theo thiết bị hoặc agent.
- Hỗ trợ transport WebSocket và IotOverMcp.

## Ghi chú

- Giữ nguyên các protocol name như MCP, SSE, WebSocket, JSON-RPC.
- Các path/API/tool name không được dịch.
