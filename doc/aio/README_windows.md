# Hướng dẫn sử dụng Xiaozhi Server trên Windows

Chào mừng bạn sử dụng gói Windows AIO của Xiaozhi Server. Tài liệu này bao gồm hướng dẫn khởi động, cấu hình và mô tả cổng.

## Nên tải bản Lite hay Full?

- **Lite**: ZIP nhẹ hơn, lần chạy đầu cần internet để tự tải model ASR tiếng Việt, Piper voice và dữ liệu `espeak-ng-data`. Nếu tải lỗi do mạng yếu hoặc GitHub bị chặn, hãy chạy lại khi mạng ổn định hoặc tải bản Full.
- **Full Offline**: ZIP lớn hơn nhưng có sẵn model, phù hợp máy không có internet hoặc muốn chạy offline hoàn toàn ngay sau khi giải nén.

Khi dùng bản Lite, chương trình sẽ tự tạo lại các thư mục `models/asr/zipformer-vietnamese-30m`, `tts-model` và `espeak-ng-data` trong lần chạy đầu.

## Cấu trúc thư mục

```text
xiaozhi_server-windows-amd64-<version>/
├── xiaozhi_server.exe          # Chương trình chính
├── onnxruntime.dll             # Thư viện dependency ONNX Runtime
├── sherpa-onnx-c-api.dll       # Thư viện dependency Sherpa-ONNX
├── sherpa-onnx-cxx-api.dll     # Thư viện dependency Sherpa-ONNX C++
├── ten_vad.dll                 # Thư viện dependency VAD
├── start.bat                   # Script khởi động
├── main_config.yaml            # File cấu hình chính
├── manager.json                # Cấu hình backend quản trị
├── asr_server.json             # Cấu hình dịch vụ ASR
├── tts_server.json             # Cấu hình Edge/Piper TTS offline nhúng
├── models/                     # Thư mục file model
├── tts-model/                  # Model Piper/VITS offline
├── espeak-ng-data/             # Dữ liệu phát âm cho Piper/VITS
├── data/                       # Thư mục dữ liệu
└── logs/                       # Thư mục log
```

## Khởi động nhanh

Nhấp đúp `start.bat` để khởi động service. Sau khi khởi động, có thể xem log trong thư mục `logs/`.

> Gợi ý: khi khởi động lần đầu, chương trình sẽ tự tải file model cần thiết (nếu thư mục `models` đang trống).

## Cổng và service

| Cổng | Nguồn cấu hình | Mô tả |
|------|----------|------|
| **1234** | `manager.json` → `server.port` | **Backend quản trị**: Web console + HTTP API |
| **1233** | `main_config.yaml` → `websocket.port` | **WebSocket service chính**: thiết bị/client kết nối |
| **1231** | `asr_server.json` → `server.port` | **Dịch vụ ASR/voiceprint**: interface nội bộ nhận diện giọng nói |
| **1232** | `tts_server.json` → `server.port` | **Edge/Piper TTS offline nhúng**: `/healthz`, `/piper/voices`, `/piper/tts`, WebSocket `/tts` |
| **1235** | Cấu hình console | **MQTT service**: thiết bị kết nối MQTT |
| **1236** | Cấu hình console | **UDP service**: thiết bị giao tiếp UDP |
| **6060** | Cấu hình console | **pprof**: phân tích hiệu năng (mặc định tắt) |

## Địa chỉ truy cập

### Backend quản trị

- **Truy cập local**: `http://localhost:1234/`
- **Truy cập LAN**: `http://<IP máy hiện tại>:1234/`

### Kết nối thiết bị/client

- **WebSocket**: `ws://<IP server>:1233/`
- **MQTT**: `<IP server>:1235`
- **UDP**: `<IP server>:1236`

### TTS offline nhúng

- **Health**: `http://localhost:1232/healthz`
- **Danh sách giọng Piper**: `http://localhost:1232/piper/voices`
- **Edge Offline**: WebSocket `ws://localhost:1232/tts`
- **Piper Offline**: HTTP `POST http://localhost:1232/piper/tts`
- Gói ZIP chỉ kèm sẵn một số giọng nhẹ hơn: `ngochuyen` và `adam1`. Muốn thêm giọng khác, copy cặp file `.onnx` và `.onnx.json` vào thư mục `tts-model/`.

## Sửa cấu hình

### Cổng cần sửa trong file cấu hình

Các cổng sau cần khởi động lại service sau khi sửa mới có hiệu lực:

| Cổng | File cấu hình | Mục cấu hình |
|------|----------|--------|
| 1234 | `manager.json` | `server.port` |
| 1233 | `main_config.yaml` | `websocket.port` |
| 1231 | `asr_server.json` | `server.port` |

### Cấu hình console

Các cổng sau và toàn bộ cấu hình khác được thay đổi qua console quản trị:

- **Cấu hình cổng**: MQTT (1235), UDP (1236), pprof (6060)
- **Cấu hình chức năng**: LLM, TTS, ASR, nhận diện voiceprint, v.v.
- Truy cập `http://localhost:1234/` để vào backend quản trị
- Thay đổi cấu hình có hiệu lực realtime, không cần khởi động lại service

## Câu hỏi thường gặp

### Cảnh báo firewall

Khi chạy lần đầu, Windows có thể hiển thị cảnh báo firewall; hãy cho phép chương trình truy cập mạng.

### Cổng bị chiếm

Nếu khởi động thất bại và báo cổng bị chiếm, hãy:

1. Dùng `netstat -ano | findstr :số_cổng` để xem process đang chiếm cổng
2. Sửa số cổng trong file cấu hình
3. Hoặc kết thúc process đang chiếm cổng đó

### Thiếu DLL

Nếu báo thiếu file DLL, hãy đảm bảo các file sau nằm cùng thư mục với `xiaozhi_server.exe`:

- `onnxruntime.dll`
- `sherpa-onnx-c-api.dll`
- `sherpa-onnx-cxx-api.dll`
- `ten_vad.dll`

## Dừng service

Nhấn `Ctrl + C` trong cửa sổ khởi động hoặc đóng trực tiếp cửa sổ để dừng service.
