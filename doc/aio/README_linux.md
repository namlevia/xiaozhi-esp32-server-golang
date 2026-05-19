# Hướng dẫn sử dụng Xiaozhi Server trên Linux

Chào mừng bạn sử dụng gói Linux AIO của Xiaozhi Server. Tài liệu này bao gồm hướng dẫn cài dependency, khởi động và cấu hình.

## Cấu trúc thư mục

```text
xiaozhi_server-linux-amd64-<version>/
├── xiaozhi_server              # Chương trình chính
├── ten-vad/
│   └── lib/Linux/x64/
│       ├── libten_vad.so       # Thư viện dependency VAD
│       ├── libsherpa-onnx-c-api.so
│       ├── libsherpa-onnx-cxx-api.so
│       └── libonnxruntime.so   # Thư viện dependency ONNX Runtime
├── main_config.yaml            # File cấu hình chính
├── manager.json                # Cấu hình backend quản trị
├── asr_server.json             # Cấu hình dịch vụ ASR
├── models/                     # Thư mục file model
├── data/                       # Thư mục dữ liệu
└── logs/                       # Thư mục log
```

## Dependency chạy

### Yêu cầu hệ thống

| Hệ thống | Phiên bản tối thiểu | Trạng thái kiểm thử |
|------|----------|----------|
| Ubuntu | 18.04 LTS | Đã kiểm thử |
| Debian | 10 (Buster) | Dự kiến tương thích, chưa kiểm thử |
| CentOS / RHEL | 8 | Dự kiến tương thích, chưa kiểm thử |

**Yêu cầu runtime**:

- **Kiến trúc**: x86_64 (amd64)

### Cài dependency

#### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y libc++1 libc++abi1
```

#### CentOS / RHEL / Fedora

```bash
sudo dnf install -y libcxx libcxxabi
# Hoặc
sudo yum install -y libcxx libcxxabi
```

#### Distribution khác

Hãy cài package tương ứng cho các thư viện sau:

- `libc++.so.1` — LLVM C++ standard library
- `libc++abi.so.1` — LLVM C++ ABI

## Khởi động nhanh

```bash
# Thêm quyền thực thi
chmod +x xiaozhi_server

# Khởi động service
./xiaozhi_server
```

### Chạy nền

Dùng `nohup`:

```bash
nohup ./xiaozhi_server > logs/output.log 2>&1 &
```

Hoặc dùng `systemd` (khuyến nghị cho môi trường production), xem phần bên dưới.

## Cổng và service

| Cổng | Nguồn cấu hình | Mô tả |
|------|----------|------|
| **8080** | `manager.json` → `server.port` | **Backend quản trị**: Web console + HTTP API |
| **8989** | `main_config.yaml` → `websocket.port` | **WebSocket service chính**: thiết bị/client kết nối |
| **9000** | `asr_server.json` → `server.port` | **Dịch vụ ASR/voiceprint**: interface nội bộ nhận diện giọng nói |
| **2883** | Cấu hình console | **MQTT service**: thiết bị kết nối MQTT |
| **8990** | Cấu hình console | **UDP service**: thiết bị giao tiếp UDP |
| **6060** | Cấu hình console | **pprof**: phân tích hiệu năng (mặc định tắt) |

## Địa chỉ truy cập

### Backend quản trị

- **Truy cập local**: `http://localhost:8080/`
- **Truy cập LAN**: `http://<IP server>:8080/`

### Kết nối thiết bị/client

- **WebSocket**: `ws://<IP server>:8989/`
- **MQTT**: `<IP server>:2883`
- **UDP**: `<IP server>:8990`

## Sửa cấu hình

### Cổng cần sửa trong file cấu hình

Các cổng sau cần restart service sau khi sửa mới có hiệu lực:

| Cổng | File cấu hình | Mục cấu hình |
|------|----------|--------|
| 8080 | `manager.json` | `server.port` |
| 8989 | `main_config.yaml` | `websocket.port` |
| 9000 | `asr_server.json` | `server.port` |

### Cấu hình console

Các cổng sau và toàn bộ cấu hình khác được thay đổi qua console quản trị:

- **Cấu hình cổng**: MQTT (2883), UDP (8990), pprof (6060)
- **Cấu hình chức năng**: LLM, TTS, ASR, nhận diện voiceprint, v.v.
- Truy cập `http://localhost:8080/` để vào backend quản trị
- Thay đổi cấu hình có hiệu lực realtime, không cần restart service

## Triển khai production bằng systemd

Tạo file service `/etc/systemd/system/xiaozhi.service`:

```ini
[Unit]
Description=Xiaozhi Server
After=network.target

[Service]
Type=simple
User=YOUR_USER
WorkingDirectory=/path/to/xiaozhi_server-linux-amd64
ExecStart=/path/to/xiaozhi_server-linux-amd64/xiaozhi_server
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Khởi động service:

```bash
# Reload cấu hình
sudo systemctl daemon-reload

# Bật tự khởi động cùng hệ thống
sudo systemctl enable xiaozhi

# Khởi động service
sudo systemctl start xiaozhi

# Xem trạng thái
sudo systemctl status xiaozhi

# Xem log
sudo journalctl -u xiaozhi -f
```

## Cấu hình firewall

Nếu server đã bật firewall, cần mở các cổng tương ứng:

```bash
# Ubuntu/Debian (ufw)
sudo ufw allow 8080/tcp  # Backend quản trị
sudo ufw allow 8989/tcp  # WebSocket
sudo ufw allow 2883/tcp  # MQTT
sudo ufw allow 8990/udp  # UDP

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=8989/tcp
sudo firewall-cmd --permanent --add-port=2883/tcp
sudo firewall-cmd --permanent --add-port=8990/udp
sudo firewall-cmd --reload
```

## Câu hỏi thường gặp

### Báo thiếu shared library

Dùng lệnh `ldd` để kiểm tra thư viện bị thiếu:

```bash
ldd xiaozhi_server
ldd ten-vad/lib/Linux/x64/libten_vad.so
```

Dựa vào output để cài package hệ thống tương ứng.

### Phiên bản glibc quá thấp

Nếu xuất hiện `version 'GLIBC_2.xx' not found`, nghĩa là phiên bản glibc của hệ thống quá cũ. Khuyến nghị:

- Nâng cấp hệ thống lên phiên bản mới hơn
- Hoặc chạy bằng Docker container

### Cổng bị chiếm

```bash
# Xem tình trạng chiếm dụng cổng
sudo lsof -i :số_cổng
# Hoặc
sudo netstat -tulpn | grep số_cổng

# Sửa số cổng trong file cấu hình hoặc kết thúc process đang chiếm cổng
```
