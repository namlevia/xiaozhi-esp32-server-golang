# Tổng kết refactor transport MCP SSE

## Tổng quan

Mục tiêu refactor là dùng SSE client native của `mark3labs/mcp-go` thay cho `github.com/r3labs/sse/v2`, tận dụng implementation MCP protocol chính thức và giúp code dễ bảo trì hơn.

Bản cập nhật mới dùng kết hợp `client.NewClient` và `transport.NewSSE` để có abstraction transport linh hoạt hơn.

## Các giai đoạn

### Giai đoạn 1: thay thư viện SSE bên thứ ba

- Xóa dependency `github.com/r3labs/sse/v2`.
- Dùng client SSE từ `mcp-go`.

### Giai đoạn 2: dùng thiết kế transport module hóa

- Dùng `transport.NewSSE` để tạo transport layer.
- Dùng `client.NewClient` để tạo MCP client từ transport.
- Tách rõ trách nhiệm giữa client và transport.

## Lợi ích

- Chuẩn hóa theo implementation MCP chính thức.
- Giảm dependency không cần thiết.
- Dễ thay thế transport trong tương lai.
- Giữ nguyên behavior gọi `initialize`, `tools/list` và tool call.
