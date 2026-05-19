# Quickstart tiếng Việt cho repo sau khi fork

Tài liệu này hướng dẫn cách chạy nhanh `xiaozhi-esp32-server-golang` sau khi fork, ưu tiên đường đi ngắn nhất để lên được console, cấu hình được backend chính, và xác minh được luồng WebSocket hoặc MQTT+UDP.

Xem thêm thuật ngữ thống nhất tại [docs/vi_vn_glossary.md](./vi_vn_glossary.md).

## 1. Mục tiêu của quickstart này

Sau khi hoàn thành tài liệu này, bạn sẽ có thể:

- chạy stack cơ bản bằng Docker Compose
- biết caveat khi local build
- biết các cổng chính cần mở
- vào được console quản trị
- chỉnh đúng các cấu hình nền như `manager.backend_url`, `udp.external_host`, OTA URL, API key ASR/LLM/TTS
- kích hoạt thiết bị
- chạy smoke test cho WebSocket và MQTT+UDP

## 2. Trước khi bắt đầu

### Yêu cầu tối thiểu

- Git
- Docker và Docker Compose plugin
- một máy Linux/macOS/Windows có thể mở các cổng cần thiết
- ít nhất một API key hoặc endpoint hoạt động cho ASR / LLM / TTS nếu bạn muốn test full voice pipeline

Kiểm tra nhanh:

```bash
docker --version
docker compose version
```

## 3. Chọn cách chạy

Repo này có nhiều cách chạy, nhưng sau khi fork thì thực tế nên đi theo thứ tự sau:

1. **Docker Compose**: dễ lên nhanh nhất, có đủ MySQL, main server, backend, frontend
2. **Local build**: phù hợp khi bạn cần debug hoặc sửa code sâu

Nếu mục tiêu là lên hệ thống để bắt đầu cấu hình và smoke test, nên dùng Docker Compose trước.

## 4. Các cổng chính cần biết

Theo docs hiện có và `config/config.yaml`, các cổng quan trọng là:

| Thành phần | Cổng | Mục đích |
|---|---:|---|
| Frontend console | `8080` hoặc mapping tương đương | Giao diện quản trị |
| Backend API | `8081` hoặc `8080` nội bộ | API cho console |
| WebSocket | `8989` | Thiết bị kết nối realtime |
| MQTT | `2883` | MQTT broker / MQTT server |
| UDP | `8990` hoặc mapping tương đương | Luồng audio/data qua UDP |
| MySQL | `3306` nội bộ, có thể map `23306` | Cơ sở dữ liệu |

Lưu ý: file `docker/docker-composer_build.yml` trong repo hiện map cổng khác với một số ví dụ trong `doc/docker_compose.md`:

- frontend: `18080:80`
- backend: `28080:8080`
- main WebSocket: `18989:8989`
- MQTT: `12883:2883`
- MySQL: `23306:3306`

Vì vậy sau khi fork, hãy **đọc đúng file compose mà bạn thực sự dùng**, không giả định mọi tài liệu đều cùng một mapping.

## 5. Chạy nhanh bằng Docker Compose

### 5.1. Clone repo

```bash
git clone <repo-fork-cua-ban>
cd xiaozhi-esp32-server-golang
```

### 5.2. Chuẩn bị file cấu hình

Repo đã có `config/config.yaml`. Với bước quickstart, bạn thường sẽ chỉnh tối thiểu các phần sau:

- `manager.backend_url`
- `websocket.port`
- `mqtt` / `mqtt_server`
- `udp.external_host`
- `ota.test.websocket.url`
- `ota.external.websocket.url`
- API key ở các provider ASR / TTS / LLM đang dùng

### 5.3. Biên dịch và chạy stack

Nếu dùng file compose build sẵn trong repo:

```bash
docker compose -f docker/docker-composer_build.yml up -d --build
```

Kiểm tra container:

```bash
docker compose -f docker/docker-composer_build.yml ps
```

## 6. Vào console quản trị

Tùy file compose bạn dùng, console thường ở một trong các địa chỉ sau:

- `http://<IP-hoac-domain>:8080`
- `http://<IP-hoac-domain>:18080`

Với `docker/docker-composer_build.yml` hiện tại của repo này, frontend map ra:

- `http://<IP-hoac-domain>:18080`

Backend API tương ứng map ra:

- `http://<IP-hoac-domain>:28080`

Nếu mở console không lên:

1. kiểm tra `docker compose ps`
2. kiểm tra cổng có đang bị chiếm không
3. kiểm tra log frontend/backend/main-server

Ví dụ:

```bash
docker compose -f docker/docker-composer_build.yml logs -f frontend
docker compose -f docker/docker-composer_build.yml logs -f backend
docker compose -f docker/docker-composer_build.yml logs -f main-server
```

## 7. Caveat khi local build

Local build **không phải đường ngắn nhất** vì repo có phụ thuộc native. Theo README và trạng thái repo hiện tại, bạn cần lưu ý:

- Go chỉ là một phần của dependency
- có thể cần `libopus`, `libopusfile`, ONNX Runtime, `libc++`, `libc++abi`
- một số module/test ở root có thể fail do phụ thuộc native hoặc mã thử nghiệm ngoài phạm vi quickstart

Nếu bạn chỉ muốn chạy được hệ thống sau khi fork, ưu tiên Docker Compose. Chỉ local build khi bạn thực sự cần debug source.

Ví dụ local build main server:

```bash
go build -o xiaozhi_server ./cmd/server/
./xiaozhi_server -c config/config.yaml
```

Nhưng trước khi chọn đường này, nên đọc thêm:

- `doc/config.md`
- `doc/compile_deploy.md`

## 8. Cấu hình bắt buộc nên chỉnh sau khi fork

### 8.1. `manager.backend_url`

Trong `config/config.yaml` hiện có:

```yaml
manager:
  backend_url: "http://127.0.0.1:8080"
```

Field này là địa chỉ backend quản trị nội bộ mà main server gọi tới.

#### Khi chạy local cùng máy

Có thể giữ kiểu:

```yaml
manager:
  backend_url: "http://127.0.0.1:8080"
```

#### Khi chạy bằng Docker Compose

Nếu main server gọi backend qua network nội bộ container, giá trị nên trỏ tới service backend, ví dụ như file compose build hiện có:

```yaml
BACKEND_URL=http://backend:8080
```

Nói ngắn gọn:

- **trong container**: thường dùng tên service như `http://backend:8080`
- **ngoài máy host / reverse proxy**: dùng URL public hoặc URL reverse proxy thực tế

## 8.2. WebSocket server

Trong `config/config.yaml`:

```yaml
websocket:
  host: "0.0.0.0"
  port: 8989
```

Thiết bị sẽ dùng URL OTA trả về để kết nối tới server này, nên ngoài việc port đúng, bạn còn phải cấu hình đúng phần OTA ở dưới.

## 8.3. MQTT / MQTT Server

Repo hỗ trợ hai kiểu:

### Kiểu A: dùng MQTT server tích hợp sẵn

```yaml
mqtt:
  enable: true
  broker: "127.0.0.1"
  port: 2883

mqtt_server:
  enable: true
  listen_port: 2883
```

Dùng khi bạn muốn một stack gọn, dễ test nhanh.

### Kiểu B: dùng broker ngoài như EMQX

- tắt hoặc không dùng `mqtt_server` nội bộ
- sửa `mqtt.broker`, `mqtt.port`, `username`, `password` theo broker ngoài
- vẫn phải để `udp.external_host` và `udp.external_port` đúng địa chỉ thiết bị truy cập được

## 8.4. `udp.external_host`

Trong `config/config.yaml`:

```yaml
udp:
  external_host: "127.0.0.1"
  external_port: 8990
  listen_host: "0.0.0.0"
  listen_port: 8990
```

Đây là phần rất hay cấu hình sai.

- `listen_host` / `listen_port`: địa chỉ server bind thực tế
- `external_host` / `external_port`: địa chỉ **trả về cho thiết bị** trong luồng hello

Nếu thiết bị nằm ngoài máy local, **không được để `127.0.0.1`**. Hãy đổi thành IP LAN, IP public hoặc domain mà thiết bị truy cập được.

Ví dụ:

```yaml
udp:
  external_host: "192.168.1.12"
  external_port: 8990
  listen_host: "0.0.0.0"
  listen_port: 8990
```

## 8.5. OTA URL và cấu hình OTA trả về

Thiết bị gọi OTA endpoint để nhận cấu hình kết nối WebSocket hoặc MQTT.

Theo docs gốc:

- OTA endpoint thường là: `http://<server>:8989/xiaozhi/ota/`

Trong `config/config.yaml`:

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

### Cách chỉnh nhanh

- `ota.test.websocket.url`: dùng cho mạng nội bộ / test
- `ota.external.websocket.url`: dùng cho môi trường public / production
- `ota.*.mqtt.enable`: bật nếu muốn thiết bị ưu tiên MQTT+UDP
- `ota.*.mqtt.endpoint`: broker address mà thiết bị sẽ nhận

Ví dụ nội bộ:

```yaml
ota:
  test:
    websocket:
      url: "ws://192.168.1.12:8989/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "192.168.1.12:2883"
```

Ví dụ public:

```yaml
ota:
  external:
    websocket:
      url: "wss://voice.example.com/xiaozhi/v1/"
    mqtt:
      enable: true
      endpoint: "mqtt.example.com:2883"
```

## 8.6. API key cho ASR / LLM / TTS

Bạn không cần điền mọi provider, chỉ cần điền provider mình thực sự dùng.

### ASR

Ví dụ một số field thường cần:

- `asr.provider`
- `asr.aliyun_funasr.api_key`
- `asr.doubao.appid`
- `asr.doubao.access_token`
- `asr.xunfei.api_key`
- `asr.xunfei.api_secret`

### LLM

Ví dụ:

- `llm.provider`
- `llm.<provider>.api_key`
- `llm.<provider>.base_url`
- `llm.<provider>.model_name`

### TTS

Ví dụ:

- `tts.provider`
- `tts.openai.api_key`
- `tts.doubao.access_token`
- `tts.doubao_ws.access_token`
- `tts.xunfei.api_key`
- `tts.xunfei.api_secret`

### Lưu ý bảo mật

Các giá trị mẫu trong repo chỉ để minh họa. Sau khi fork và trước khi dùng môi trường thật, hãy đổi:

- `manager.auth_token`
- `manager.endpoint_auth_token`
- `mqtt.password`
- `mqtt_server.password`
- `mqtt_server.signature_key`
- `ota.signature_key`
- mọi `api_key`, `api_secret`, `access_token`

## 9. Kích hoạt thiết bị

Từ các docs gốc, luồng chung là:

1. thiết bị biết OTA URL
2. thiết bị gọi `POST /xiaozhi/ota/`
3. server trả về cấu hình WebSocket hoặc MQTT
4. thiết bị kết nối lên server
5. nếu bật activation flow, bạn lấy mã kích hoạt trong console và bind thiết bị

Với ESP32, OTA URL thường cần nhập theo một trong hai cách:

### Cách 1: sửa qua giao diện Wi‑Fi provisioning

Điền URL dạng:

```text
http://<IP-cua-ban>:8989/xiaozhi/ota/
```

Ví dụ:

```text
http://192.168.1.12:8989/xiaozhi/ota/
```

### Cách 2: nhúng sẵn vào firmware

Theo docs gốc, có thể đặt kiểu:

```json
"CONFIG_OTA_URL": "http://<your-server>/xiaozhi/ota/"
```

### Sau đó làm gì trong console

- mở trang quản trị thiết bị
- tìm thiết bị mới online hoặc chờ mã kích hoạt xuất hiện
- thêm / bind thiết bị theo flow của console
- kiểm tra trạng thái online

Nếu không kích hoạt được:

- kiểm tra OTA URL có trỏ đúng host và port không
- kiểm tra thiết bị có vào được `websocket.url` hoặc `mqtt.endpoint` mà OTA trả về không
- kiểm tra chữ ký / token / signature key nếu bạn đã bật cơ chế xác thực liên quan

## 10. Smoke test WebSocket

Dùng khi bạn muốn xác minh đường kết nối đơn giản nhất trước.

### Mục tiêu

- OTA trả đúng `websocket.url`
- thiết bị hoặc client test mở được WebSocket
- console/backend thấy thiết bị online hoặc có log kết nối

### Checklist

1. `websocket.port` đang mở
2. `ota.test.websocket.url` hoặc `ota.external.websocket.url` trỏ đúng địa chỉ thật
3. log main server không báo lỗi auth/config
4. console hiển thị thiết bị online hoặc có bản ghi kết nối

### Test tối thiểu bằng HTTP OTA

Bạn có thể kiểm tra endpoint OTA có phản hồi không:

```bash
curl -i -X POST \
  -H 'Device-Id: test-device-001' \
  -H 'Client-Id: test-client-001' \
  http://127.0.0.1:8989/xiaozhi/ota/
```

Nếu chạy qua compose với port mapping khác, đổi lại host/port tương ứng, ví dụ `18989`.

Điều bạn cần thấy là server trả được payload OTA hợp lệ, trong đó có `websocket.url` đúng môi trường.

## 11. Smoke test MQTT + UDP

Dùng khi bạn muốn xác minh mode hiệu năng cao hơn.

### Mục tiêu

- OTA trả `mqtt.enable: true`
- thiết bị nhận đúng `mqtt.endpoint`
- MQTT broker nhận kết nối
- UDP trả về đúng `external_host` và `external_port`

### Checklist cấu hình

```yaml
mqtt:
  enable: true
  broker: "<broker-host>"
  port: 2883

mqtt_server:
  enable: true
  listen_port: 2883

udp:
  external_host: "<host-thiet-bi-truy-cap-duoc>"
  external_port: 8990
  listen_port: 8990

ota:
  test:
    mqtt:
      enable: true
      endpoint: "<broker-host>:2883"
```

### Cách kiểm tra nhanh

1. gọi OTA endpoint và xác nhận response có phần MQTT đúng
2. kiểm tra log main server / mqtt server có kết nối mới
3. xác nhận thiết bị gửi `hello`
4. xác nhận server trả về thông tin UDP và thiết bị bắt đầu dùng UDP để truyền dữ liệu

Nếu dùng broker ngoài, nhớ thêm phần topic mapping theo tài liệu `doc/mqtt_udp.md`.

## 12. Những lỗi cấu hình gặp nhiều nhất

### Console lên được nhưng thiết bị không vào được

Thường do một trong các lỗi sau:

- `ota.*.websocket.url` vẫn đang trỏ IP mẫu cũ
- `udp.external_host` để `127.0.0.1`
- port mở ở local nhưng chưa mở ở firewall / security group / NAT
- dùng compose port mapping nhưng lại điền port nội bộ vào OTA

### Main server gọi không tới backend

Kiểm tra lại:

- `manager.backend_url`
- nếu chạy Docker Compose, main server nên gọi backend qua service name nội bộ như `http://backend:8080`
- nếu chạy tách ngoài container, phải dùng URL thật mà main server truy cập được

### Có API key nhưng pipeline vẫn fail

Kiểm tra đúng provider đang bật:

- `asr.provider`
- `tts.provider`
- `llm.provider`

Rất nhiều trường hợp key được điền ở provider A nhưng hệ thống lại đang chọn provider B.

## 13. Trình tự khuyến nghị sau khi fork

Nếu muốn đi nhanh và ít lỗi nhất, nên làm theo thứ tự này:

1. fork và clone repo
2. chạy Docker Compose
3. vào console
4. sửa `manager.backend_url` nếu cần
5. sửa `udp.external_host`
6. sửa `ota.test.websocket.url` và `ota.external.websocket.url`
7. điền API key cho đúng provider ASR / LLM / TTS đang dùng
8. test OTA bằng `curl`
9. bind/kích hoạt thiết bị
10. smoke test WebSocket
11. nếu cần hiệu năng cao hơn, bật và smoke test MQTT+UDP

## 14. Tài liệu gốc nên đọc tiếp

Sau quickstart này, nên đọc tiếp:

- `doc/docker_compose.md`
- `doc/config.md`
- `doc/manager_console_guide.md`
- `doc/websocket_server.md`
- `doc/mqtt_udp.md`
- `doc/esp32_xiaozhi_backend_guide.md`

## 15. Ghi chú cuối

Tài liệu này không thay thế docs gốc, mà là bản tóm tắt tiếng Việt để giúp bạn chạy nhanh repo sau khi fork. Khi có khác biệt giữa tài liệu và cấu hình thực tế của nhánh bạn đang dùng, hãy ưu tiên:

1. file compose thực tế bạn đang chạy
2. `config/config.yaml`
3. log runtime của container / service
4. docs gốc trong thư mục `doc/`
