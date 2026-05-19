# Hướng dẫn sử dụng Config Manager

## Tổng quan

Package này cung cấp hai manager chính:

1. **ConfigManager** - manager cấu hình, cung cấp chức năng quản lý config cấp cao.
2. **AuthManager** - manager xác thực, chuyên xử lý kích hoạt thiết bị và xác thực liên quan.

## Tính năng chính

### ConfigManager

- Cơ chế cache config để cải thiện hiệu năng truy cập.
- Validate config.
- Quản lý global theo singleton.
- Hỗ trợ load config hệ thống và config thiết bị từ backend Manager.

### AuthManager

- Kiểm tra trạng thái kích hoạt thiết bị.
- Lấy thông tin kích hoạt thiết bị.
- Xác minh challenge/HMAC.
- Hỗ trợ luồng kích hoạt thiết bị.

## Cách dùng cơ bản

```go
manager, err := NewConfigManager(map[string]interface{}{
    "backend_url": "http://localhost:8080",
})
if err != nil {
    return err
}

config, err := manager.GetSystemConfig(ctx)
if err != nil {
    return err
}
_ = config
```

## Ghi chú

- Giữ nguyên các key cấu hình như `backend_url`, `device_id`, `agent_id`.
- Các request tới backend Manager dùng HTTP/WebSocket tùy chức năng.
- Với WebSocket, client có cơ chế heartbeat và reconnect.
