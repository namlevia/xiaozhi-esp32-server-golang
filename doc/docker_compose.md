# Hướng dẫn triển khai Docker Compose

## Tổng quan

Dự án này dùng Docker Compose để triển khai container hóa, bao gồm các service cốt lõi sau:

- **Service database MySQL**: lưu trữ dữ liệu
- **Service chương trình chính**: logic nghiệp vụ cốt lõi
- **Service backend quản trị**: service API interface
- **Service frontend quản trị**: giao diện Web quản trị

## Chỉ dẫn nhanh (bổ sung)

Phần này bổ sung cho `doc/docker.md`, giúp chọn nhanh và triển khai đúng cách.

### Local Docker stack khuyến nghị

Từ root repo:

```bash
docker compose -f docker/docker-composer/docker-compose.local.yml build
docker compose -f docker/docker-composer/docker-compose.local.yml up -d
docker compose -f docker/docker-composer/docker-compose.local.yml ps
```

URL và port host mặc định của stack local:

| Thành phần | URL/cổng host | Hostname nội bộ trong Docker |
| --- | --- | --- |
| Frontend console | `http://localhost:18080` | `frontend:80` |
| Backend API | `http://localhost:28080` | `backend:8080` |
| Main-server WebSocket/OTA | `localhost:18989` | `main-server:8989` |
| Embedded TTS / Piper | `http://localhost:19001` | `main-server:9001` |
| Voice/ASR service | `localhost:18082` | `voice-server:9000` |
| MQTT | `localhost:12883` | `main-server:2883` |
| MySQL | `localhost:23306` | `mysql:3306` |

Sau khi vào frontend, Dashboard của admin có thẻ Health Check để kiểm tra nhanh backend, database, main-server/TTS, Piper voices, ASR và readiness cấu hình. Có thể smoke check trực tiếp:

```bash
curl http://localhost:19001/healthz
curl http://localhost:19001/piper/voices
```

Lưu ý phân biệt hostname: cấu hình chạy trong container nên dùng `main-server`, `backend`, `voice-server`, `mysql`; URL trình duyệt/người dùng mới dùng `localhost` kèm port host.

### 1. Chọn cách triển khai

- Khuyến nghị: Docker Compose (bao gồm backend quản trị và service đầy đủ)
- Đơn giản: Docker một container (không có console hoặc chế độ rút gọn)

### 2. Đường dẫn nhanh Docker Compose

1. Kéo code hoặc chuẩn bị `docker-compose.yml`
2. Tham khảo các phần “chuẩn bị file cấu hình” và “khởi động service” bên dưới để hoàn tất cấu hình
3. Khởi động:

```bash
docker compose up -d
```

4. Địa chỉ backend quản trị mặc định: `http://<IP hoặc domain server>:8080/`

### 3. Docker một container (bổ sung)

Sau khi build hoặc pull image theo `doc/docker.md`, chạy container. Khuyến nghị thường gặp:

- Map các thư mục `config/`, `logs/`, `storage/` thành volume dữ liệu
- Expose các cổng WebSocket / MQTT / UDP ra ngoài
- Khi cần backend quản trị, hãy bật tham số tương ứng hoặc dùng Compose

### 4. Wizard cấu hình và kiểm thử

Sau khi khởi động, có thể dùng wizard cấu hình trong backend quản trị để hoàn tất cấu hình engine, rồi dùng công cụ test để kiểm tra tính khả dụng và độ trễ của VAD/ASR/LLM/TTS, cũng như xác minh toàn bộ luồng OTA.

### 5. Câu hỏi thường gặp

- Xung đột cổng: kiểm tra tình trạng chiếm dụng 8080/8989/2883/8990
- Cấu hình chưa có hiệu lực: xác nhận đường dẫn volume mount đúng, khởi động lại container để có hiệu lực
- Vấn đề quyền: trên Linux chú ý quyền thư mục mount và giới hạn SELinux

## Kiến trúc service

### 1. Service database MySQL (xiaozhi-mysql)

**Thông tin cấu hình:**

- Image: `docker.jsdelivr.fyi/mysql:8.0`
- Ánh xạ cổng: `23306:3306`
- Tên database: `xiaozhi_admin`
- Tên người dùng: `root`
- Mật khẩu: `password`

**Đặc điểm:**

- Dùng MySQL 8.0
- Có cấu hình health check
- Dữ liệu được persist

### 2. Service chương trình chính (xiaozhi-main-server)

**Thông tin cấu hình:**

- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_server:0.5`
- Ánh xạ cổng:
  - `8989:8989` - WebSocket service
  - `2882:2883` - MQTT service
  - `8888:8888/udp` - UDP service

**Quan hệ dependency:**

- Phụ thuộc trạng thái healthy của MySQL service
- Phụ thuộc backend service đã khởi động xong

**Hỗ trợ file cấu hình:**

- Import file cấu hình tùy chỉnh bằng volume mount
- Đường dẫn file cấu hình: `../../config:/workspace/config`

**Hỗ trợ ten_vad:**

- Docker image đã bao gồm thư viện ten_vad (`/workspace/lib/ten-vad/`)
- Đường dẫn runtime library đã tự cấu hình qua `LD_LIBRARY_PATH`

### 3. Service backend quản trị (xiaozhi-backend)

**Thông tin cấu hình:**

- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_manager_backend:0.5`
- Ánh xạ cổng: `8081:8080`

**Chức năng:**

- Cung cấp RESTful API
- Quản lý thiết bị và người dùng

**Hỗ trợ file cấu hình:**

- Import file cấu hình tùy chỉnh bằng volume mount
- Đường dẫn file cấu hình: `../../manager/backend/config:/root/config`

### 4. Service frontend quản trị (xiaozhi-frontend)

**Thông tin cấu hình:**

- Image: `docker.jsdelivr.fyi/hackers365/xiaozhi_manager_frontend:0.5`
- Ánh xạ cổng: `8080:80`

**Chức năng:**

- Giao diện Web quản trị (lối vào nội bộ)
- Quản lý trạng thái thiết bị và cấu hình hệ thống

## Quy trình triển khai

### 1. Chuẩn bị môi trường

Đảm bảo hệ thống đã cài Docker và Docker Compose:

```bash
docker --version
docker compose version
```

### 2. Chuẩn bị file cấu hình

Đảm bảo các thư mục và file sau tồn tại:

```text
xiaozhi-esp32-server-golang/
├─ docker/docker-composer/
│  └─ docker-compose.yml
├─ config/
│  ├─ config.yaml
│  ├─ config.json
│  └─ (file cấu hình khác)
├─ logs/
│  └─ (thư mục log)
└─ manager/backend/config/
   ├─ config.yaml
   └─ (file cấu hình khác)
```

**Mô tả import file cấu hình:**

- File cấu hình chương trình chính được import qua volume mount `../../config:/workspace/config`
- File cấu hình backend được import qua volume mount `../../manager/backend/config:/root/config`

### 3. Khởi động service

**Bắt buộc vào thư mục `docker/docker-composer/` trước khi chạy lệnh:**

```bash
cd docker/docker-composer/
docker compose up -d

docker compose ps
docker compose logs -f
```

### 4. Truy cập service

- Giao diện frontend quản trị: `http://<IP hoặc domain server>:8080`
- Backend API: `http://localhost:8081`
- WebSocket: `ws://localhost:8989`
- MQTT: `localhost:2882`
- UDP: `localhost:8888`
- MySQL: `localhost:23306`

## Thao tác thường dùng

```bash
cd docker/docker-composer/

docker compose ps

docker compose logs

docker compose logs -f main-server

docker compose restart

docker compose down

docker compose down -v

docker compose pull

docker compose up -d
```

## Piper TTS offline

Stack local mount thư mục `tts_server/tts-model` vào `main-server` để dùng Piper/VITS offline. Các endpoint kiểm tra nhanh:

- `GET http://localhost:19001/healthz` - health của embedded TTS server.
- `GET http://localhost:19001/piper/voices` - danh sách giọng đọc phát hiện được từ model `.onnx` kèm `.onnx.json`.
- `POST http://localhost:19001/piper/tts` - tổng hợp giọng Piper ra WAV/PCM.

Trong UI TTS, provider `Piper TTS offline` sẽ tự tải danh sách giọng từ `/piper/voices`; khi chọn giọng, `model_path`, `model_config_path`, sample rate và tham số giọng được tự điền. Nếu danh sách trống, kiểm tra file `.onnx.json`, quyền ghi thư mục model và volume mount `tts_server/tts-model`.

## Cấu hình mạng

Dự án dùng network tùy chỉnh `xiaozhi-network`:

- MySQL: `mysql:3306`
- Backend: `backend:8080`
- Frontend: `frontend:80`
- Chương trình chính: `main-server:8989` (WebSocket) / `main-server:2883` (MQTT) / `main-server:8888` (UDP)

**Tổng hợp ánh xạ cổng:**

- 8080 → giao diện frontend quản trị
- 8081 → Backend API
- 8989 → WebSocket
- 2882 → MQTT
- 8888 → UDP
- 23306 → MySQL

## Persist dữ liệu

### Dữ liệu MySQL

Persist qua Docker volume `mysql_data`, dữ liệu không mất khi container restart.

### File cấu hình

- Cấu hình chương trình chính: `../../config:/workspace/config`
- Cấu hình backend: `../../manager/backend/config:/root/config`

Sau khi sửa cấu hình, restart service tương ứng để có hiệu lực:

```bash
cd docker/docker-composer/
docker compose restart main-server

docker compose restart backend
```

### File log

- Log chương trình chính: `../../logs:/workspace/logs`

## Cách import file cấu hình

### 1. Cấu hình chương trình chính

**Vị trí:**

```text
xiaozhi-esp32-server-golang/config/
├─ config.yaml
├─ config.json
├─ mqtt_config.json
└─ (file cấu hình khác)
```

**Import:**

1. Đặt file cấu hình vào `config/`
2. Sau khi khởi động, file tự động được mount vào container `/workspace/config/`
3. Sau khi sửa, restart service chương trình chính:

```bash
cd docker/docker-composer/
docker compose restart main-server
```

### 2. Cấu hình backend quản trị

**Vị trí:**

```text
xiaozhi-esp32-server-golang/manager/backend/config/
├─ config.yaml
└─ (file cấu hình khác)
```

**Import:**

1. Đặt file cấu hình vào `manager/backend/config/`
2. Sau khi khởi động, file tự động được mount vào container `/root/config/`
3. Sau khi sửa, restart backend service:

```bash
cd docker/docker-composer/
docker compose restart backend
```

### 3. File thư viện ten_vad

**Mô tả:**

- Docker image đã bao gồm thư viện ten_vad (`/workspace/lib/ten-vad/`)
- Đường dẫn runtime library đã tự cấu hình qua `LD_LIBRARY_PATH`
- Dùng ten_vad không cần mount thêm

## Health check

MySQL service đã cấu hình health check:

```yaml
healthcheck:
  test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-ppassword"]
  timeout: 20s
  retries: 10
  interval: 10s
  start_period: 30s
```

## Xử lý sự cố

### 1. Service khởi động thất bại

```bash
cd docker/docker-composer/

docker compose logs [tên_service]

# Kiểm tra chiếm dụng cổng (Linux)
netstat -tulpn | grep [cổng]
```

### 2. Kết nối database thất bại

```bash
cd docker/docker-composer/

docker compose ps mysql

docker compose logs mysql

docker compose exec mysql mysql -u root -ppassword
```

### 3. Vấn đề kết nối mạng

```bash
cd docker/docker-composer/

docker network ls
docker network inspect xiaozhi-network

docker compose exec main-server ping mysql
```

## Gợi ý tối ưu hiệu năng

1. Thiết lập giới hạn tài nguyên cho từng service trong môi trường production
2. Cấu hình log rotation để tránh log quá lớn
3. Backup dữ liệu MySQL định kỳ
4. Tích hợp hệ thống giám sát

## Lưu ý bảo mật

1. Đổi mật khẩu database mặc định trong môi trường production
2. Chỉ expose cổng theo nhu cầu
3. Cấu hình firewall và kiểm soát truy cập
4. Dùng nguồn image đáng tin cậy

---

## Bước tiếp theo

### Truy cập backend quản trị

Sau khi service khởi động, truy cập http://<IP hoặc domain server>:8080 để vào backend quản trị.

**[Hướng dẫn sử dụng trang quản trị →](manager_console_guide.md)**

### Cấu hình thiết bị ESP32

Tham khảo [hướng dẫn tích hợp phía ESP32](esp32_xiaozhi_backend_guide.md) để hoàn tất kết nối thiết bị.
