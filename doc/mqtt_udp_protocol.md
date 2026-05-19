# 🚦 Luồng dữ liệu

1. **Gọi interface OTA**
   - Lấy địa chỉ **MQTT** và **WebSocket**

2. **Kết nối MQTT**
   - `mqtt_server` tích hợp sẵn sẽ publish một sự kiện vòng đời tới `/p2p/device_public/_server/lifecycle`
   - Chương trình chính tạo hoặc tái sử dụng MQTT transport theo `device_id`, đồng thời cố gắng preheat MCP phía thiết bị theo best-effort

3. **Gửi message `hello`**
   - Lấy:
     - 🎵 `audio_params`
     - 🌐 địa chỉ UDP server
     - 🔑 `aes_key`
     - 🧩 `nonce`

4. **Kết nối UDP server**
   - Thực hiện gửi và nhận dữ liệu giọng nói

5. **Gửi các tín hiệu tiếp theo như `listen`, `abort`**
   - Ngữ nghĩa tín hiệu giữ nguyên, vẫn dựa trên khởi tạo cấp chat sau khi hoàn tất `hello`

---

# 🧭 Topic vòng đời

- **Topic**: `/p2p/device_public/_server/lifecycle`
- **Mục đích**: chỉ dùng nội bộ phía server để truyền sự kiện online/offline MQTT của thiết bị
- **Ví dụ message body**:
  ```json
  {
    "type": "mqtt_lifecycle",
    "device_id": "11:22:33:44:55:66",
    "state": "online",
    "client_id": "GID_test@@@11_22_33_44_55_66@@@uuid",
    "ts": 1710000000000
  }
  ```

- **Định nghĩa trạng thái**
  - `online`: thiết bị vừa kết nối vào `mqtt_server`, chương trình chính có thể chuẩn bị trước transport và MCP
  - `offline`: thiết bị đã ngắt khỏi `mqtt_server`, chương trình chính lập tức ánh xạ trạng thái offline, nhưng transport sẽ được giữ lại một thời gian để tái sử dụng khi reconnect ngắn hạn

- **Ranh giới**
  - Sự kiện vòng đời không thay thế `hello`
  - Sự kiện vòng đời chỉ duy trì tài nguyên cấp kết nối, không mang thông tin cấp chat như `audio_params`, thương lượng UDP, v.v.

---

# 🛠️ Luồng phía server

| Bước | Mô tả |
| :--- | :--- |
| 1. Lắng nghe vòng đời MQTT | Khi nhận sự kiện `online`, tạo hoặc tái sử dụng transport, đồng thời cố gắng preheat MCP phía thiết bị theo best-effort |
| 2. Xử lý `hello` | Trả về `audio_params`, địa chỉ UDP, khóa và `nonce`, đồng thời chuẩn bị trạng thái session cấp chat |
| 3. Lắng nghe message MQTT | Khi nhận `type: listen, state: start`, khởi tạo cấu trúc `clientState`, trạng thái là `start` |
| 4. Dịch vụ UDP | Sau khi nhận packet, parse `nonce`, tìm `clientState` tương ứng, điền địa chỉ remote, trạng thái là `recv` |
| 5. Dừng nhận | Khi nhận `type: listen, state: stop` hoặc tự động phát hiện không có giọng nói, dừng nhận |
| 6. Vòng đời MQTT offline | Khi nhận sự kiện `offline`, lập tức ánh xạ trạng thái offline và thu hồi transport sau thời gian giữ lại |

---

# 🔗 Quan hệ liên kết

- OTA xác thực **địa chỉ MAC** và **clientId**, rồi liên kết tới **uid**
- **Địa chỉ MQTT** và **mqtt_clientId** do OTA cấp xuống liên kết **địa chỉ MAC** và **clientId**
- Có thể liên kết trước **địa chỉ MAC**, `device_id`, `client_id` thông qua **message vòng đời kết nối MQTT**
- Có thể liên kết tới `audio_params`, `aes_key`, `nonce` thông qua **message MQTT `hello`**
- Có thể liên kết tới `nonce` thông qua **message âm thanh UDP**

---

> **Ghi chú:**
> - Cấu trúc `clientState` dùng để duy trì trạng thái session cấp chat và tài nguyên của từng client.
> - Transport và MCP có thể được chuẩn bị trước ở giai đoạn MQTT online, nhưng thương lượng cấp chat thật sự vẫn lấy `hello` làm chuẩn.
> - `nonce` là định danh duy nhất giữa client và server, dùng cho liên kết bảo mật và định tuyến dữ liệu.
