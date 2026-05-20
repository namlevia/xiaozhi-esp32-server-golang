# Xiaozhi Server Go - Android ARM64 Lite

Đây là bản thử nghiệm cho thiết bị Android ARM64 chạy bằng Termux hoặc môi trường shell tương tự. Đây không phải file APK cài như ứng dụng Android thông thường.

## Cách chạy nhanh

Giải nén ZIP vào thư mục có quyền ghi, sau đó chạy:

```bash
chmod +x xiaozhi_server
./xiaozhi_server
```

Nếu muốn lưu log ra file:

```bash
mkdir -p logs
./xiaozhi_server > logs/run-android.log 2>&1
```

## Cổng mặc định

- Web quản trị: `http://127.0.0.1:1234`
- WebSocket thiết bị: `1233`
- MQTT: `1235`
- UDP: `1236`
- TTS Edge local: `http://127.0.0.1:1232/healthz`, WebSocket `ws://127.0.0.1:1232/tts`

Nếu cổng `1234` bị ứng dụng khác chiếm, đổi đồng thời:

- `manager.json`: `server.port`
- `main_config.yaml`: `manager.backend_url`

Ví dụ đổi sang `38080`:

```json
"port": "38080"
```

```yaml
manager:
  backend_url: "http://127.0.0.1:38080"
```

Sau đó chạy lại `./xiaozhi_server` và mở `http://127.0.0.1:38080`.

## Lưu ý về bản Android ARM64 Lite

- Bản này ưu tiên web quản trị và server chính chạy trong shell Android.
- TTS Edge local đã được bật qua service nhúng `/tts`; lần tổng hợp cần internet vì Edge TTS vẫn gọi dịch vụ Microsoft ở phía server local.
- Piper native và ASR native Android chưa bật đầy đủ như Windows/Linux/macOS vì cần Go binding và `.so` Android cho sherpa/onnxruntime, Piper/espeak và các module liên quan.
- Android có thể giới hạn tiến trình nền; nên giữ Termux mở khi đang test.
