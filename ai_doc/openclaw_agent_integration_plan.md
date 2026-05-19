# Phương án triển khai tích hợp OpenClaw theo chiều Agent

## 1. Mục tiêu

Dựa trên `XIAOZHI_OPENCLAW_PROTOCOL.md`, triển khai tích hợp OpenClaw trong `xiaozhi-esp32-server-golang` với các yêu cầu sau:

1. Console không thêm trang cấu hình OpenClaw độc lập.
2. Cách tạo OpenClaw endpoint nhất quán với MCP endpoint: tạo theo trợ lý, token chứa `user_id`, `agent_id`.
3. Chương trình chính quản lý kết nối OpenClaw WebSocket theo `agent_id`.
4. Cấu hình thiết bị gửi xuống OpenClaw bằng struct: cờ cho phép + keyword vào/thoát.
5. Sau ASR hỗ trợ vào/thoát chế độ OpenClaw; trong chế độ này message bỏ qua LLM, đi thẳng qua OpenClaw rồi tới TTS.
6. Khi phản hồi OpenClaw đến trễ và thiết bị offline, đưa vào hàng đợi offline trong bộ nhớ; gửi bù khi thiết bị online lần sau.
7. Chính sách hàng đợi offline: tối đa 20 message mỗi thiết bị, TTL 24 giờ.

## 2. Sửa console (manager backend)

### 2.1 Mở rộng field trợ lý

Mở rộng `models.Agent`:

- `OpenclawEnabled bool`
- `OpenclawEnterKeywords string` (chuỗi JSON array)
- `OpenclawExitKeywords string` (chuỗi JSON array)

Ghi chú:

- Không thêm bảng cấu hình OpenClaw endpoint độc lập.
- Logic lấy endpoint align với MCP: tạo động và trả qua API trợ lý.

### 2.2 API Endpoint

Thêm API cho người dùng và admin:

- `GET /api/user/agents/:id/openclaw-endpoint`
- `GET /api/admin/agents/:id/openclaw-endpoint`

Hành vi:

1. Xác minh agent tồn tại và quyền sở hữu hợp lệ.
2. Đọc OTA public WebSocket URL.
3. Sinh JWT token ổn định, hiệu lực dài hạn, không set `exp/iat`.
4. Trả `ws(s)://host/ws/openclaw?token=<token>`.

### 2.3 Token Claims

Thêm claims cho OpenClaw:

- `user_id`
- `agent_id`
- `endpoint_id` (`agent_<agentID>`)
- `purpose=openclaw-endpoint`

## 3. Gửi xuống và parse cấu hình thiết bị

### 3.1 Struct

Thêm vào `UConfig` của chương trình chính:

```go
type OpenClawConfig struct {
    Allowed       bool     `json:"allowed"`
    EnterKeywords []string `json:"enter_keywords"`
    ExitKeywords  []string `json:"exit_keywords"`
}
```

Và thêm vào `UConfig`:

```go
OpenClaw OpenClawConfig `json:"openclaw"`
```

### 3.2 Phản hồi /api/configs

Thêm vào `GetDeviceConfigs` phía manager:

```json
"openclaw": {
  "allowed": true,
  "enter_keywords": ["vào chế độ OpenClaw", "openclaw"],
  "exit_keywords": ["thoát chế độ OpenClaw", "thoát openclaw"]
}
```

Quy tắc fill:

1. `allowed = agent.openclaw_enabled`
2. Parse `enter_keywords/exit_keywords` từ field của agent dưới dạng JSON array
3. Nếu field rỗng hoặc parse thất bại thì fallback về array rỗng

### 3.3 Chương trình chính kéo cấu hình

`ConfigManager.GetUserConfig` parse object `openclaw` và ghi vào `types.UConfig.OpenClaw`.

## 4. OpenClaw WebSocket server trong chương trình chính

### 4.1 Route

Thêm route:

- `/ws/openclaw`

### 4.2 Chiều kết nối

Quản lý connection pool theo `agent_id`:

- key: `agentID`
- value: OpenClaw session (một kết nối)

Khi kết nối mới được thiết lập, thay thế kết nối cũ để đảm bảo mỗi agent chỉ có một kết nối OpenClaw WS đang hoạt động.

### 4.3 Xử lý protocol

1. Sau khi kết nối, gửi `handshake_ack` trước
2. Nhận `ping` thì trả `pong`
3. Khi nhận `response`:
   - Tìm route device theo `correlation_id`
   - Nếu thiết bị online thì đẩy TTS
   - Nếu thiết bị offline thì ghi vào hàng đợi offline
4. Nhận `error/close` thì ghi log và cleanup theo session

## 5. Sửa luồng Chat

### 5.1 Trạng thái session

Thêm runtime state OpenClaw vào `ClientState`:

- `OpenClawMode bool`

### 5.2 Phân luồng sau ASR

Thêm quy trình trong `ChatSession.actionDoChat`:

1. Nhận diện keyword thoát, ưu tiên cao nhất
2. Nhận diện keyword vào
3. Nếu hiện đang ở chế độ OpenClaw:
   - Gửi text trực tiếp tới OpenClaw
   - Không gọi LLM
4. Nếu không ở chế độ OpenClaw:
   - Giữ nguyên luồng LLM hiện tại

Cách match keyword: chuẩn hóa text trước (trim khoảng trắng đầu/cuối, bỏ dấu câu phổ biến), rồi match bằng `contains`.

## 6. Hàng đợi message offline

Thêm manager hàng đợi offline trong bộ nhớ:

- key: `deviceID`
- value: `[]OfflineMessage`
- Field mỗi bản ghi: `Text`, `CreatedAt`, `CorrelationID`

Chính sách:

1. Tối đa 20 message mỗi thiết bị, vượt thì bỏ message cũ nhất
2. TTL 24h, cleanup cả khi ghi và khi đọc
3. Khi thiết bị online, tự replay và xóa các message gửi thành công

Điểm kích hoạt online:

- Sau khi thiết bị online trong `App.OnNewConnection`, kích hoạt replay message offline của thiết bị đó

## 7. Điểm sửa code chính

1. `manager/backend/models/models.go`
2. `manager/backend/controllers/admin.go`
3. `manager/backend/controllers/user.go`
4. `manager/backend/router/router.go`
5. `internal/domain/config/types/types.go`
6. `internal/domain/config/manager/manager.go`
7. `internal/app/server/websocket/websocket_server.go`
8. `internal/app/server/websocket/openclaw.go` (thêm mới)
9. `internal/domain/openclaw/*` (thêm mới: connection pool, message model, hàng đợi offline)
10. `internal/data/client/client.go`
11. `internal/app/server/chat/session.go`
12. `internal/app/server/app.go`

## 8. Bước cài đặt (gồm cấu hình channel)

1. Cài plugin xiaozhi OpenClaw:
   `openclaw plugins install @xiaozhi_openclaw/xiaozhi`
2. Trong console, bật cấu hình OpenClaw của trợ lý và copy OpenClaw endpoint của trợ lý đó (`ws(s)://.../ws/openclaw?token=...`).
3. Trong phiên OpenClaw, thực hiện “cấu hình channel”:
   - Gửi trực tiếp endpoint ở bước trước cho OpenClaw
   - Nói rõ: `cấu hình plugin kênh xiaozhi`
4. Sau khi cấu hình xong, gửi một message trong phiên test để xác nhận nhận được phản hồi OpenClaw.

## 9. Checklist xác minh

1. Console có thể lấy OpenClaw endpoint, logic tạo nhất quán với MCP.
2. `/api/configs` trả field `openclaw` có cấu trúc.
3. OpenClaw client có thể kết nối `/ws/openclaw` và handshake.
4. Keyword vào có thể chuyển sang chế độ OpenClaw, keyword thoát có thể thoát chế độ.
5. Trong chế độ OpenClaw, message không đi qua LLM, phản hồi có thể chuyển sang TTS.
6. Khi thiết bị offline, phản hồi vào hàng đợi offline; khi online sẽ gửi bù.
7. Hàng đợi offline đáp ứng giới hạn 20 message và TTL 24 giờ.
