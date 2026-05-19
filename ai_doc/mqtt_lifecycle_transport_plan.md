# Phương án tạo trước Transport theo vòng đời MQTT

## Mục tiêu

Khi thiết bị kết nối/ngắt khỏi `mqtt_server`, `mqtt_server` sẽ publish message vòng đời qua callback tới MQTT topic mà chương trình chính đang lắng nghe. Sau khi chương trình chính nhận được:

1. Tạo trước `mqtt udp transport` khi thiết bị online
2. Pre-warm MCP theo best effort khi thiết bị online
3. Mapping trạng thái thiết bị offline ngay khi thiết bị offline
4. Giữ transport thêm một khoảng thời gian sau khi thiết bị offline, tránh tạo/hủy liên tục khi reconnect ngắn hạn
5. Không thay đổi ngữ nghĩa signaling hiện tại của `hello` / `listen` / `abort` / `goodbye`

## Thiết kế Topic

Không thêm root prefix mới, tái sử dụng prefix hiện có `"/p2p/device_public/"`.

Thêm lifecycle topic:

`/p2p/device_public/_server/lifecycle`

Đề xuất hằng số code tương ứng:

- `MDeviceLifecycleTopic = MDevicePubTopicPrefix + "_server/lifecycle"`

## Format message vòng đời

Body message dùng JSON:

```json
{
  "type": "mqtt_lifecycle",
  "device_id": "ba:8f:17:de:94:94",
  "state": "online",
  "client_id": "GID_test@@@ba_8f_17_de_94_94@@@uuid",
  "ts": 1710000000000
}
```

Giải thích trường:

- `type`: cố định là `mqtt_lifecycle`
- `device_id`: ID thiết bị đã chuẩn hóa, thống nhất dùng format có dấu hai chấm
- `state`: `online` / `offline`
- `client_id`: MQTT client id gốc, tiện debug
- `ts`: timestamp sự kiện, đơn vị millisecond

## Luồng end-to-end

### 1. mqtt_server publish message vòng đời

Trong `DeviceHook`:

- `OnSessionEstablished`
- `OnDisconnect`

Dùng callback để publish sự kiện vòng đời tới `/p2p/device_public/_server/lifecycle`.

Về implementation, `mqtt_server` vẫn chịu trách nhiệm publish; chỉ gom hành động publish thành callback được gọi trong hook để tránh logic ghép topic rải rác nhiều nơi.

### 2. Chương trình chính tái sử dụng subscription hiện có

`MqttUdpAdapter` tiếp tục chỉ subscribe topic hiện có:

`/p2p/device_public/#`

Sau khi nhận message, kiểm tra topic trước:

- Nếu là `/p2p/device_public/_server/lifecycle`, đi vào nhánh xử lý vòng đời
- Ngược lại tiếp tục đi theo nhánh message nghiệp vụ thiết bị hiện có

Cách này không ảnh hưởng việc parse signaling bình thường như `hello` / `listen`.

### 3. Tạo trước transport khi thiết bị online

Sau khi nhận lifecycle message `online`:

1. Debounce vòng đời trước
2. Nếu transport chưa tồn tại, tạo ngay `MqttUdpConn + UdpSession`
3. Kích hoạt `onNewConnection` để chương trình chính tạo `ChatManager`
4. Đánh dấu broker online
5. Kích hoạt một lần pre-warm MCP theo best effort
6. Mapping trạng thái thiết bị online

Lưu ý:

- Ở đây tạo `transport` và `ChatManager`
- `ChatSession` vẫn giữ cơ chế tạo lười sau `hello`

### 4. Thu hồi transport trễ khi thiết bị offline

Sau khi nhận lifecycle message `offline`:

1. Đánh dấu broker offline trước
2. Mapping trạng thái thiết bị offline ngay lập tức
3. Khởi động cleanup timer trễ
4. Giữ `transport + udp session` trong grace period
5. Nếu nhận lại `online` trong grace period, hủy cleanup timer và tái sử dụng transport cũ

Thời gian giữ mặc định đề xuất là `2m`, sau này có thể cấu hình hóa.

## Ngữ nghĩa trạng thái online

Trạng thái online của thiết bị MQTT-UDP đổi sang được điều khiển bởi vòng đời MQTT, thay vì do việc tạo/hủy `ChatManager` điều khiển.

Tức là:

- MQTT `online` -> thiết bị online
- MQTT `offline` -> thiết bị offline

Để tránh thông báo lặp:

- `App.OnNewConnection()` giữ logic cũ với `websocket`
- `DeviceOnline / DeviceOffline` của `mqtt udp` chuyển sang do callback vòng đời của `MqttUdpAdapter` kích hoạt

## Quan hệ với hello / listen

Không đổi logic signaling chat hiện có:

- Transport có thể tồn tại sớm sau khi kết nối MQTT được thiết lập
- `ChatManager` có thể tồn tại sớm
- `ChatSession` vẫn tạo sau khi `hello` thành công
- `listen` vẫn yêu cầu `hello` đã hoàn tất

Như vậy có thể “tạo trước transport” mà không thay đổi ngữ nghĩa tầng session.

## Chiến lược pre-warm MCP

Sau khi lifecycle `online` đến, kích hoạt một lần pre-warm MCP theo best effort.

Đồng thời giữ logic khởi tạo MCP fallback hiện có trong `hello`.

Khi hai đường cùng tồn tại, dựa vào năng lực idempotent và state machine MCP hiện có trên nhánh hiện tại để tránh khởi tạo lặp:

- Ưu tiên pre-warm khi online để tăng khả năng hiển thị tool trên console
- Tiếp tục fallback ở `hello` để tránh việc thiếu pre-warm ảnh hưởng nghiệp vụ

## Đồng thời cao và debounce

Duy trì trạng thái vòng đời theo chiều thiết bị:

- `brokerOnline`
- `lastEventTs`
- `cleanupTimer`
- `cleanupVersion`

Quy tắc debounce:

- Bỏ qua trực tiếp sự kiện có timestamp cũ
- `online` lặp lại không thông báo online lặp
- `offline` lặp lại chỉ refresh cleanup timer, không thông báo offline lặp
- Khi callback timer chạy, kiểm tra `cleanupVersion` để tránh timer cũ xóa nhầm kết nối mới

## Điểm cần sửa kèm

Vì transport sẽ được giữ ngắn hạn sau khi offline, nên việc resolve “transport online hiện tại” không thể chỉ dựa vào `ChatManager` có tồn tại hay không.

Cần để `MqttUdpConn` expose trạng thái broker online, đồng thời để `ChatManager.GetTransportType()` trả chuỗi rỗng khi MQTT transport đã offline. Như vậy MCP query/call theo chiều thiết bị vẫn phụ thuộc nghiêm ngặt vào “transport online hiện tại”.

## File liên quan

- `internal/data/msg/message_types.go`
- `internal/app/mqtt_server/device_hook.go`
- `internal/app/mqtt_server/mqtt_server.go`
- `internal/app/server/mqtt_udp/mqtt_udp_adapter.go`
- `internal/app/server/mqtt_udp/mqtt_udp_conn.go`
- `internal/app/server/app.go`
- `internal/app/server/chat/chat.go`
