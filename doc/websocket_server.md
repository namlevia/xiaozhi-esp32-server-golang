# Hướng dẫn quy trình cấu hình WebSocket server và OTA

Tài liệu này dành cho người dùng mới, mô tả chi tiết cách cấu hình WebSocket server và các tham số liên quan đến OTA (nâng cấp firmware).

---

## 1. Vị trí file cấu hình

Toàn bộ cấu hình chính nằm ở:

- `config/config.yaml`

Nếu không tìm thấy file này, cũng có thể tham khảo `config/config.json.git`.

---

## 2. Cấu hình WebSocket server

### 2.1 Tác dụng

WebSocket server dùng cho giao tiếp realtime giữa thiết bị và server.

### 2.2 Mục cấu hình quan trọng

Tìm nội dung sau trong file `config/config.yaml`:

```yaml
websocket:
  host: "0.0.0.0"
  port: 8989
```

- `host`: địa chỉ lắng nghe, thường giữ `0.0.0.0` là được.
- `port`: cổng lắng nghe, mặc định `8989`, có thể sửa theo nhu cầu.

### 2.3 Cách sửa

Nếu cần đổi cổng thành 9000:

```yaml
websocket:
  host: "0.0.0.0"
  port: 9000
```

---

## 3. Cấu hình OTA (nâng cấp firmware)

### 3.1 Tác dụng

OTA dùng để thiết bị tự động lấy tham số kết nối WebSocket/MQTT do server cấp xuống và thông tin nâng cấp firmware.

### 3.2 Mục cấu hình quan trọng

Tìm phần `ota` trong file `config/config.yaml`:

```yaml
ota:
  test:
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "192.168.208.214"
  external:
    websocket:
      url: "wss://www.tb263.cn:55555/go_ws/xiaozhi/v1/"
    mqtt:
      enable: false
      endpoint: "www.youdomain.cn"
```

- `test`: tham số thiết bị lấy trong môi trường nội bộ; trong chương trình, điều kiện nhận diện thường là IP bắt đầu bằng `192.168` hoặc `127.0`.
- `external`: tham số thiết bị lấy trong môi trường bên ngoài.
- `websocket.url`: địa chỉ WebSocket server mà thiết bị cần kết nối.
- `mqtt.enable`: nếu bật, interface OTA sẽ trả về địa chỉ MQTT đã cấu hình; thiết bị sẽ ưu tiên dùng phương thức MQTT+UDP.
- `mqtt.endpoint`: địa chỉ MQTT server. Phía thiết bị mặc định dùng cổng `8883` (kết nối TLS); nếu kèm cổng khác `8883` thì sẽ dùng kết nối TCP không mã hóa.

### 3.3 Ví dụ sửa thường gặp

- Sửa địa chỉ WebSocket nội bộ:
  ```yaml
  ota:
    test:
      websocket:
        url: "ws://192.168.1.100:8989/xiaozhi/v1/"
  ```

- Sửa địa chỉ WebSocket bên ngoài:
  ```yaml
  ota:
    external:
      websocket:
        url: "wss://yourdomain.com:55555/go_ws/xiaozhi/v1/"
  ```

---

## 4. Mô tả interface OTA (thiết bị lấy cấu hình như thế nào)

1. Thiết bị gửi HTTP POST tới `http://địa_chỉ_server:cổng/xiaozhi/ota/`.
2. Request header cần chứa:
   - `Device-Id`: ID duy nhất của thiết bị (như địa chỉ MAC)
   - `Client-Id`: ID duy nhất của client
3. Server sẽ tự động chọn cấu hình `test` hoặc `external` theo IP thiết bị và trả về tham số như WebSocket/MQTT.
4. Thiết bị parse nội dung trả về rồi kết nối WebSocket server theo `websocket.url`.

---

## 5. Câu hỏi thường gặp

- **Cổng bị chiếm?**
  - Sửa `websocket.port`, rồi khởi động lại service.
- **Thiết bị không kết nối được server?**
  - Kiểm tra `websocket.url` trong cấu hình `ota` có đúng không, cổng server đã mở chưa.
- **Cần MQTT?**
  - Đặt `mqtt.enable` thành `true`, rồi cấu hình `endpoint`.

---

Nếu có thắc mắc, nên kiểm tra mục cấu hình trong `config/config.yaml` trước, sau đó tham khảo tài liệu này.
