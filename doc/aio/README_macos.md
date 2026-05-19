# Hướng dẫn sử dụng Xiaozhi Server trên macOS

Chào mừng bạn sử dụng gói macOS AIO của Xiaozhi Server. Tài liệu này bao gồm hướng dẫn cài dependency, khởi động và cấu hình.

## Cấu trúc thư mục

```text
xiaozhi_server-macos-<arch>-<version>/
├── xiaozhi_server              # Chương trình chính
├── ten-vad/
│   └── lib/macOS/
│       ├── ten_vad.framework/  # Framework VAD
│       ├── libonnxruntime.*.dylib
│       └── libsherpa-onnx-*.dylib
├── main_config.yaml            # File cấu hình chính
├── manager.json                # Cấu hình backend quản trị
├── asr_server.json             # Cấu hình dịch vụ ASR
├── tts_server.json             # Cấu hình Edge/Piper TTS offline nhúng
├── models/                     # Thư mục file model
├── tts-model/                  # Model Piper/VITS offline
├── data/                       # Thư mục dữ liệu
└── logs/                       # Thư mục log
```

> **Lưu ý**: bản macOS được chia thành **amd64** (Intel) và **arm64** (Apple Silicon), hãy tải đúng phiên bản khớp với máy Mac của bạn.

## Dependency chạy

### Yêu cầu hệ thống

- **Phiên bản macOS**: macOS 11 (Big Sur) hoặc cao hơn
- **Kiến trúc**: Intel (x86_64) hoặc Apple Silicon (arm64)

### Cài dependency

Dùng Homebrew để cài dependency cần thiết:

```bash
# Cài Homebrew (nếu chưa cài)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Cài dependency
brew install pkg-config
```

## Khởi động nhanh

```bash
# Thêm quyền thực thi
chmod +x xiaozhi_server

# Nếu đây là gói release bạn tự build, hãy sửa rpath trước
./build/macos/fix_rpath.sh ./xiaozhi_server

# Khởi động service
./xiaozhi_server
```

Ghi chú:

- Gói release chính thức nếu đã được đóng gói đầy đủ thường không cần chạy lại `fix_rpath.sh`
- Chỉ khi bạn tự build gói phân phối macOS trong source repo thì mới cần bổ sung bước này
- Bước này sẽ sửa `rpath` đường dẫn tuyệt đối trên máy build trong binary thành `@executable_path/ten-vad/lib/macOS`

### Cảnh báo bảo mật khi chạy lần đầu

Khi chạy lần đầu, macOS có thể hiển thị cảnh báo bảo mật vì chương trình chưa được Apple chứng thực. Hãy:

1. Mở “System Settings” → “Privacy & Security”
2. Tìm cảnh báo liên quan đến `xiaozhi_server`
3. Nhấp “Open Anyway” hoặc “Allow”

Hoặc dùng lệnh sau để gỡ quarantine:

```bash
xattr -cr xiaozhi_server
```

## Cổng và service

| Cổng | Nguồn cấu hình | Mô tả |
|------|----------|------|
| **8080** | `manager.json` → `server.port` | **Backend quản trị**: Web console + HTTP API |
| **8989** | `main_config.yaml` → `websocket.port` | **WebSocket service chính**: thiết bị/client kết nối |
| **9000** | `asr_server.json` → `server.port` | **Dịch vụ ASR/voiceprint**: interface nội bộ nhận diện giọng nói |
| **9001** | `tts_server.json` → `server.port` | **Edge/Piper TTS offline nhúng**: `/healthz`, `/piper/voices`, `/piper/tts`, WebSocket `/tts` |
| **2883** | Cấu hình console | **MQTT service**: thiết bị kết nối MQTT |
| **8990** | Cấu hình console | **UDP service**: thiết bị giao tiếp UDP |
| **6060** | Cấu hình console | **pprof**: phân tích hiệu năng (mặc định tắt) |

## Địa chỉ truy cập

### Backend quản trị

- **Truy cập local**: `http://localhost:8080/`
- **Truy cập LAN**: `http://<IP máy hiện tại>:8080/`

### Kết nối thiết bị/client

- **WebSocket**: `ws://<IP server>:8989/`
- **MQTT**: `<IP server>:2883`
- **UDP**: `<IP server>:8990`

### TTS offline nhúng

- **Health**: `http://localhost:9001/healthz`
- **Danh sách giọng Piper**: `http://localhost:9001/piper/voices`
- **Edge Offline**: WebSocket `ws://localhost:9001/tts`
- **Piper Offline**: HTTP `POST http://localhost:9001/piper/tts`
- Gói ZIP chỉ kèm sẵn một số giọng nhẹ hơn: `ngochuyen` và `adam1`. Muốn thêm giọng khác, copy cặp file `.onnx` và `.onnx.json` vào thư mục `tts-model/`.

## Sửa cấu hình

### Cổng cần sửa trong file cấu hình

Các cổng sau cần khởi động lại service sau khi sửa mới có hiệu lực:

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
- Thay đổi cấu hình có hiệu lực realtime, không cần khởi động lại service

## Chạy nền

### Dùng nohup

```bash
nohup ./xiaozhi_server > logs/output.log 2>&1 &
```

### Tạo service launchd (khuyến nghị)

Tạo `~/Library/LaunchAgents/com.xiaozhi.server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.xiaozhi.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/xiaozhi_server</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/path/to/xiaozhi_server-macos-<arch>-<version></string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/path/to/logs/output.log</string>
    <key>StandardErrorPath</key>
    <string>/path/to/logs/error.log</string>
</dict>
</plist>
```

Load service:

```bash
# Load service
launchctl load ~/Library/LaunchAgents/com.xiaozhi.server.plist

# Khởi động service
launchctl start com.xiaozhi.server

# Xem trạng thái
launchctl list | grep xiaozhi

# Dừng service
launchctl stop com.xiaozhi.server

# Unload service
launchctl unload ~/Library/LaunchAgents/com.xiaozhi.server.plist
```

## Cấu hình firewall

Nếu đã bật firewall, cần cho phép `xiaozhi_server` nhận kết nối inbound:

1. Mở “System Settings” → “Network” → “Firewall”
2. Nhấp “Options”
3. Tìm `xiaozhi_server`, đặt thành “Allow incoming connections”

Hoặc dùng lệnh trong terminal:

```bash
# Thêm ngoại lệ firewall (cần sudo)
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /path/to/xiaozhi_server
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblock /path/to/xiaozhi_server
```

## Câu hỏi thường gặp

### Cảnh báo bảo mật “đã bị hỏng”

Nếu có cảnh báo ứng dụng đã bị hỏng, chạy lệnh sau:

```bash
xattr -cr xiaozhi_server
```

### Load dynamic library thất bại

Nếu xuất hiện lỗi load `dylib`, kiểm tra:

```bash
# Xem dependency
otool -L xiaozhi_server

# Xem rpath
otool -l xiaozhi_server | grep -A2 LC_RPATH

# Đảm bảo dynamic library nằm đúng vị trí
ls -la ten-vad/lib/macOS/
```

Nếu `LC_RPATH` vẫn là đường dẫn source tuyệt đối trên máy build thay vì `@executable_path/ten-vad/lib/macOS`, hãy chạy:

```bash
./build/macos/fix_rpath.sh ./xiaozhi_server
```

Nếu bạn đang debug trong thư mục tạm của IDE, hoặc đã di chuyển binary thủ công khiến cấu trúc thư mục không nhất quán, có thể tạm dùng:

```bash
DYLD_FRAMEWORK_PATH="$PWD/ten-vad/lib/macOS" ./xiaozhi_server
```

### Cổng bị chiếm

```bash
# Xem tiến trình chiếm cổng
lsof -i :số_cổng

# Kết thúc tiến trình đang chiếm hoặc sửa cổng trong file cấu hình
```

### Chạy bản Intel trên Apple Silicon (M1/M2/M3)

Chạy bản Intel trên máy Mac Apple Silicon cần Rosetta 2:

```bash
# Cài Rosetta 2
softwareupdate --install-rosetta
```

Tuy nhiên, khuyến nghị tải bản arm64 tương ứng để có hiệu năng tốt nhất.
