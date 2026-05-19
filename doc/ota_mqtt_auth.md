# Cấu hình xác thực MQTT cho interface OTA

## Tổng quan

Interface OTA hiện hỗ trợ cơ chế xác thực mật khẩu MQTT dựa trên chữ ký HMAC-SHA256, giúp cung cấp cách xác thực an toàn hơn. MQTT server cũng hỗ trợ logic xác thực tương ứng.

## Cấu trúc cấu hình

### File cấu hình `config/config.yaml`

```yaml
mqtt_server:
  signature_key: "your_ota_signature_key_here"
ota:
  signature_key: "your_ota_signature_key_here"
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

### Mô tả cấu hình

- `mqtt_server.signature_key`: khóa ký MQTT, dùng để tạo chữ ký mật khẩu MQTT.
- `ota.signature_key`: key dùng khi OTA cấp mật khẩu MQTT xuống thiết bị; cần tương ứng với `mqtt_server.signature_key`.
- `ota.test`: cấu hình môi trường test, dùng cho IP nội bộ.
- `ota.external`: cấu hình môi trường bên ngoài, dùng cho IP public.

### Tích hợp với xiaozhi-mqtt-gateway

Hệ thống này có thể phối hợp với dự án chính thức [xiaozhi-mqtt-gateway](https://github.com/78/xiaozhi-mqtt-gateway) để hoàn thiện luồng xác thực MQTT:

1. **Yêu cầu cấu hình nhất quán**: `ota.signature_key` phải hoàn toàn giống khóa ký trong dự án xiaozhi-mqtt-gateway.
2. **Luồng xác thực**:
   - xiaozhi-mqtt-gateway chịu trách nhiệm tạo credential kết nối MQTT.
   - Hệ thống này chịu trách nhiệm xác thực credential kết nối MQTT.
   - Hai bên dùng cùng thuật toán ký và cùng khóa để đảm bảo xác thực thành công.
3. **Khuyến nghị triển khai**: nên triển khai hai dự án trong cùng môi trường mạng và đảm bảo cập nhật cấu hình đồng bộ.

## Hàm tiện ích

### 1. Tạo chữ ký mật khẩu

```go
// Tạo chữ ký mật khẩu HMAC-SHA256.
password := util.GeneratePasswordSignature(data, key)
```

### 2. Tạo credential MQTT

```go
// Tạo credential kết nối MQTT đầy đủ.
credentials, err := util.GenerateMqttCredentials(deviceId, clientId, ip, signatureKey)
if err != nil {
    // Xử lý lỗi.
}
// credentials gồm: ClientId, Username, Password.
```

### 3. Xác thực credential MQTT

```go
// Xác thực credential kết nối MQTT.
credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
if err != nil {
    // Xác thực thất bại.
}
// credentialInfo gồm: GroupId, MacAddress, UUID, UserData.
```

## Logic xác thực MQTT

### 1. Định dạng Client ID

```
GID_test@@@{deviceId}@@@{clientId}
```

Ví dụ:

```
GID_test@@@02_4A_7D_E3_89_BF@@@e3b0c442-98fc-4e1a-8c3d-6a5b6a5b6a5b
```

### 2. Định dạng Username

JSON được encode Base64, chứa thông tin IP của client:

```yaml
ip: "1.202.193.194"
```

Sau khi encode Base64:

```
eyJpcCI6IjEuMjAyLjE5My4xOTQifQ==
```

### 3. Tạo Password

Dùng thuật toán HMAC-SHA256 để tạo chữ ký mật khẩu:

```go
signatureData := clientId + "|" + username
password := HMAC-SHA256(signatureData, signature_key)
```

### 4. Logic xác thực

Khi xác thực client cần:

1. Parse `clientId` để lấy `groupId`, `macAddress`, `uuid`.
2. Decode `username` để lấy thông tin IP.
3. Dùng cùng khóa ký và thuật toán để xác thực mật khẩu.

## Xác thực MQTT server

### Luồng xác thực

1. **Xác thực super admin**
   - Username: `admin`, có thể cấu hình.
   - Password: `shijingbo!@#`, có thể cấu hình.

2. **Xác thực người dùng thường**
   - Ưu tiên xác thực bằng chữ ký HMAC-SHA256.
   - Nếu chưa cấu hình khóa ký, fallback về cách xác thực AES.

### Triển khai hook xác thực

```go
func (h *AuthHook) OnConnectAuthenticate(cl *mqttServer.Client, pk packets.Packet) bool {
    username := string(pk.Connect.Username)
    password := string(pk.Connect.Password)
    clientId := string(pk.Connect.ClientIdentifier)

    // Kiểm tra super admin.
    if username == adminUsername && password == adminPassword {
        return true
    }

    // Kiểm tra người dùng thường bằng logic xác thực chữ ký mới.
    signatureKey := viper.GetString("mqtt_server.signature_key")
    if signatureKey != "" {
        credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
        if err != nil {
            return false
        }
        return true
    }

    // Fallback về logic xác thực AES.
    return h.validateWithAes(username, password)
}
```

## Tương thích

- Nếu chưa cấu hình `mqtt_server.signature_key`, hệ thống sẽ fallback về cách tạo mật khẩu SHA256/AES cũ.
- Giữ tương thích ngược và không ảnh hưởng chức năng hiện có.
- MQTT server hỗ trợ nhiều cách xác thực cùng tồn tại.

## Khuyến nghị bảo mật

1. Dùng chuỗi ngẫu nhiên mạnh làm khóa ký.
2. Xoay vòng khóa ký định kỳ.
3. Dùng HTTPS/WSS trong môi trường production.
4. Theo dõi các lần đăng nhập bất thường.
5. Bật ghi log để truy vết xác thực thành công/thất bại.
6. **Đảm bảo xiaozhi-mqtt-gateway và hệ thống này cập nhật khóa ký đồng bộ.**

## Cấu trúc dữ liệu

### MqttCredentials

```go
type MqttCredentials struct {
    ClientId string `json:"client_id"`
    Username string `json:"username"`
    Password string `json:"password"`
}
```

### MqttCredentialInfo

```go
type MqttCredentialInfo struct {
    GroupId    string                 `json:"groupId"`
    MacAddress string                 `json:"macAddress"`
    UUID       string                 `json:"uuid"`
    UserData   map[string]interface{} `json:"userData"`
}
```

# Hướng dẫn dùng xiaozhi-mqtt-gateway chính thức

Hệ thống này có thể phối hợp với dự án chính thức [xiaozhi-mqtt-gateway](https://github.com/78/xiaozhi-mqtt-gateway).

Chỉ cần username/password MQTT do interface OTA cấp xuống được xiaozhi-mqtt-gateway xác thực thành công. Để xác thực MQTT hoạt động bình thường, **`ota.signature_key` phải nhất quán với khóa ký trong xiaozhi-mqtt-gateway**.

Cấu hình như sau:

1. Không bật MQTT server nội bộ, dùng xiaozhi-mqtt-gateway.
2. `ota.signature_key` phải nhất quán với khóa ký trong xiaozhi-mqtt-gateway.
3. Cấu hình backend WebSocket của xiaozhi-mqtt-gateway trỏ tới địa chỉ của dự án này.

```yaml
mqtt_server:
  enable: false
ota:
  signature_key: "your_ota_signature_key_here"
  test:  # Response cho test mạng nội bộ.
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "192.168.208.214:1883"  # Địa chỉ MQTT server trong xiaozhi-mqtt-gateway.
  external:  # Response cho mạng bên ngoài.
    websocket:
      url: "wss://www.tb263.cn:55555/go_ws/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "mqtt.youdomain.com:1883"  # Địa chỉ MQTT server trong xiaozhi-mqtt-gateway.
```
