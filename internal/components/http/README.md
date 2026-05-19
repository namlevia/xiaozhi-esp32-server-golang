# Component HTTP

Component HTTP thống nhất, dùng để quản lý mọi lời gọi HTTP tới backend Manager.

## Cấu trúc thư mục

```text
http/
├── client.go          # HTTP client dùng chung (hỗ trợ retry, auth...)
├── manager_client.go  # Client chuyên dụng cho backend Manager
├── types.go           # Định nghĩa kiểu dữ liệu
└── README.md          # Tài liệu này
```

## Thiết kế

### Client (HTTP client dùng chung)

Cung cấp chức năng HTTP request cơ bản:

- Hỗ trợ retry (dùng exponential backoff)
- Hỗ trợ auth token (Bearer Token)
- Hỗ trợ timeout tùy chỉnh
- Xử lý lỗi thống nhất
- Tự động serialize/deserialize JSON

### ManagerClient (client chuyên dụng cho backend Manager)

Bọc trên client dùng chung, chuyên dùng để gọi API backend Manager.

## Ví dụ sử dụng

### Tạo Manager client

```go
client := http.NewManagerClient(http.ManagerClientConfig{
    BaseURL:   "http://localhost:8000",
    AuthToken: "your-token",  // tùy chọn
    Timeout:   30 * time.Second,
})
```

### Gửi GET request

```go
var response SomeResponse
err := client.DoRequest(ctx, http.RequestOptions{
    Method:   "GET",
    Path:     "/api/example",
    Response: &response,
})
```

### Gửi POST request

```go
request := SomeRequest{Name: "example"}
var response SomeResponse
err := client.DoRequest(ctx, http.RequestOptions{
    Method:   "POST",
    Path:     "/api/example",
    Body:     request,
    Response: &response,
})
```

### Lấy response raw

```go
body, err := client.DoRequestRaw(ctx, http.RequestOptions{
    Method: "GET",
    Path:   "/api/example",
})
```

## Ghi chú refactor

### Trước refactor

- `HistoryClient` và `ConfigManager` tự triển khai logic gọi HTTP riêng.
- Code bị lặp, chi phí bảo trì cao.
- Logic retry, auth... bị phân tán.

### Sau refactor

- Component HTTP thống nhất, quản lý tập trung.
- Tái sử dụng code, dễ bảo trì.
- Xử lý lỗi và retry thống nhất.

## Module đã refactor

1. **internal/data/history/client.go** - client lịch sử chat
2. **internal/domain/config/manager/manager.go** - config manager
3. **internal/domain/config/manager/auth.go** - API liên quan auth

## Lưu ý

- Mọi lời gọi HTTP tới backend Manager nên dùng `ManagerClient`.
- Nếu cần gọi backend service khác, có thể tạo client chuyên dụng mới dựa trên `Client`.
- Cơ chế retry mặc định tối đa 3 lần, có thể điều chỉnh qua config.
