# Phương án refactor tạo lười và tái sử dụng dài hạn ChatSession

## Mục tiêu

- Chuyển `ChatSession` từ “tạo ngay khi kết nối được thiết lập” sang “tạo lười sau khi `hello` thành công, và tái sử dụng lâu dài trong suốt vòng đời kết nối”.
- `ChatSession.Close()` chỉ giải phóng tài nguyên miền chat, không đóng `serverTransport` và cũng không đóng IoT-over-MCP phía thiết bị.
- Tách `hello` thành hai phần theo trách nhiệm:
  - Cấp transport: trả thông tin bắt tay như `transport/udp(server, port, key, nonce)`.
  - Cấp chat: ghi `audio_params`, khởi tạo hoặc tái sử dụng `SessionID`, refresh cấu hình thiết bị, kích hoạt tạo phiên.
- Tách `mcp/iot/goodbye` khỏi luồng chạy chính của `ChatSession`, giao cho `ChatManager` xử lý.

## Ranh giới thiết kế

### ChatManager

- Owner cấp kết nối.
- Giữ `transport`, `serverTransport`, `clientState`, `mcpTransport`, `hookHub`, `transformRegistry`.
- Chịu trách nhiệm khởi động và giữ command loop, audio loop.
- Chịu trách nhiệm xử lý:
  - `hello`
  - `mcp`
  - `iot`
  - `goodbye`
- Route `listen/abort`, khi cần thì gọi `ensureSession()`.

### ChatSession

- Chỉ chịu trách nhiệm miền chat:
  - `listen`
  - `abort`
  - ASR/VAD
  - LLM/TTS
  - Phát media cấp phiên
- `Start()` không còn khởi động `CmdMessageLoop/AudioMessageLoop` cấp kết nối.
- `Start()` khởi động các phần sau khi format audio đầu vào đã sẵn sàng:
  - Vòng nền VAD/ASR
  - `processChatText`
  - `llmManager.Start`
  - `ttsManager.Start`

## Quy ước vòng đời

- `hello` đầu tiên:
  - Ghi `clientState.InputAudioFormat`
  - Tạo `SessionID`
  - Tùy chọn khởi tạo MCP phía thiết bị
  - `ensureSession()`
  - Trả `hello`
- `hello` lặp lại:
  - Cập nhật `audio_params`
  - Refresh cấu hình thiết bị
  - Nếu hiện không có `ChatSession` hoạt động, gọi lại `ensureSession()`
  - Tùy chọn kích hoạt lại khởi tạo MCP phía thiết bị
- `mqtt_udp`:
  - Khi chat kết thúc bình thường, không đóng transport
  - Thoát rõ ràng/lỗi nghiêm trọng chỉ hủy `ChatSession`
  - Sau đó vẫn có thể tiếp tục tái sử dụng kết nối và dựng lại `ChatSession`
- `websocket`:
  - Sau khi thoát rõ ràng/lỗi nghiêm trọng, `ChatManager` đóng transport sau khi cleanup phiên hoàn tất

## Điểm sửa code

- `internal/app/server/chat/chat.go`
  - `ChatManager` giữ tài nguyên cấp kết nối và route message
  - Thêm `ensureSession()`, `HandleHelloMessage()`, loop `cmd/audio` cấp kết nối
- `internal/app/server/chat/session.go`
  - `Start()` chỉ giữ tác vụ nền miền chat
  - `Close()` đổi thành giải phóng thuần tài nguyên chat
  - Thêm callback đóng phiên để `ChatManager` xử lý khác nhau theo protocol
- `internal/app/server/chat/server_transport.go`
  - Thêm đường đóng không đóng transport nền, dùng cho trường hợp remote đã ngắt
- `internal/app/server/event_handle.go`
  - Sự kiện thoát chat chuyển sang do `ChatManager` thực thi, thay vì lấy trực tiếp `ChatSession`

## Điểm xác minh

- Sau khi kết nối mới được thiết lập, không tạo `ChatSession` ngay.
- Sau `hello` đầu tiên, tạo `ChatSession` và vẫn tiếp tục được `listen/start`.
- Trong `mqtt_udp`, sau `ChatSession.Close()` transport vẫn có thể tiếp tục nhận/gửi lệnh.
- Trong `websocket`, sau khi thoát rõ ràng thì kết nối được đóng.
- Tìm kiếm MCP tool tiếp tục giữ transport-aware, không fallback về chiều không có transport.
