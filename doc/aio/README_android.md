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

- Web quản trị: `http://127.0.0.1:8080`
- WebSocket thiết bị: `8989`
- MQTT: `2883`
- UDP: `8990`

Nếu cổng `8080` bị ứng dụng khác chiếm, đổi đồng thời:

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
- ASR/TTS nhúng native Android chưa được bật mặc định như Windows/Linux/macOS vì cần thư viện `.so` riêng cho Android.
- Nếu cần ASR/TTS đầy đủ trên Android, cần bổ sung và kiểm thử thêm thư viện native Android cho sherpa/onnxruntime, Piper/espeak và các module liên quan.
- Android có thể giới hạn tiến trình nền; nên giữ Termux mở khi đang test.
