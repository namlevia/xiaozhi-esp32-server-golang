# Hướng dẫn sử dụng trang quản trị

## Truy cập trang quản trị

- Địa chỉ: http://<IP hoặc domain server>:8080

---

## I. Wizard cấu hình

Sau lần đăng nhập đầu tiên, hệ thống tự động vào wizard cấu hình, gồm 5 bước.

### Step 1: Cấu hình OTA

Cấu hình thông tin OTA server, dùng để cấp địa chỉ websocket và mqtt xuống phần cứng Xiaozhi.

<!-- Vị trí ảnh chụp: giao diện cấu hình OTA -->
> Hình: giao diện wizard cấu hình OTA

| Mục cấu hình | Mô tả |
|-------|------|
| MQTT Broker | Địa chỉ MQTT server |
| MQTT Port | Cổng MQTT (mặc định 1883) |
| UDP Port | Cổng UDP |
| ... | ... |

**Kiểm tra kết nối**: nhấp “Kiểm tra cấu hình hiện tại” để xác minh kết nối MQTT/UDP.

---

### Step 2: Cấu hình VAD

Chọn engine phát hiện hoạt động giọng nói:

<!-- Vị trí ảnh chụp: giao diện cấu hình VAD -->
> Hình: giao diện wizard cấu hình VAD

| Engine | Mô tả | Kịch bản khuyến nghị |
|-----|------|---------|
| Silero VAD | Độ chính xác cao | Môi trường production |
| WebRTC VAD | Nhẹ | Tài nguyên hạn chế |
| ten_vad | Bản C++ cục bộ | Nhu cầu hiệu năng cao |

---

### Step 3: Cấu hình ASR

Chọn engine nhận diện giọng nói:

<!-- Vị trí ảnh chụp: giao diện cấu hình ASR -->
> Hình: giao diện wizard cấu hình ASR

| Engine | Mô tả |
|-----|------|
| FunASR | Nhận diện cục bộ, cần tải model |
| Doubao ASR | API cloud |

---

### Step 4: Cấu hình LLM

Chọn mô hình ngôn ngữ lớn:

<!-- Vị trí ảnh chụp: giao diện cấu hình LLM -->
> Hình: giao diện wizard cấu hình LLM

| Engine | Mô tả |
|-----|------|
| Tương thích OpenAI | Hỗ trợ nhiều loại API |
| Ollama | Triển khai cục bộ |
| Doubao | Doubao của ByteDance |

---

### Step 5: Cấu hình TTS

Chọn engine tổng hợp giọng nói:

<!-- Vị trí ảnh chụp: giao diện cấu hình TTS -->
> Hình: giao diện wizard cấu hình TTS

| Engine | Mô tả |
|-----|------|
| Doubao TTS | API cloud |
| EdgeTTS | Microsoft TTS miễn phí |
| CosyVoice | Chất lượng cao cục bộ |

---

## II. Kiểm tra cấu hình

### Kiểm tra từng cấu hình

Trong từng trang cấu hình, nhấp nút “Kiểm tra” ở bên phải mục cấu hình:

<!-- Vị trí ảnh chụp: nút kiểm tra từng cấu hình -->
> Hình: nút kiểm tra cấu hình

Mô tả kết quả kiểm tra:

| Trường | Mô tả |
|-----|------|
| Trạng thái | Thành công/thất bại |
| Độ trễ gói đầu tiên | Thời gian phản hồi cấp mili giây |
| Message | Chi tiết lỗi (nếu thất bại) |

<!-- Vị trí ảnh chụp: popup kết quả kiểm tra -->
> Hình: popup kết quả kiểm tra cấu hình

### Kiểm tra hàng loạt

Trong trang quản lý cấu hình, nhấp “Kiểm tra tất cả” để kiểm tra hàng loạt toàn bộ cấu hình:

<!-- Vị trí ảnh chụp: giao diện kiểm tra hàng loạt -->
> Hình: giao diện kiểm tra hàng loạt

### Loại kiểm tra được hỗ trợ

| Loại kiểm tra | Mô tả |
|---------|------|
| VAD | Kết nối và thời gian phản hồi của phát hiện hoạt động giọng nói |
| ASR | Kết nối và độ trễ gói đầu tiên của nhận diện giọng nói |
| LLM | Kết nối và độ trễ gói đầu tiên của suy luận mô hình lớn |
| TTS | Kết nối và độ trễ gói đầu tiên của tổng hợp giọng nói |
| OTA | Kiểm tra kết nối MQTT/UDP |

---

## III. Giám sát độ trễ

Xem thống kê độ trễ gói đầu tiên của từng module trong hệ thống:

<!-- Vị trí ảnh chụp: giao diện giám sát độ trễ -->
> Hình: giao diện giám sát độ trễ

### Gợi ý tối ưu độ trễ

| Module | Hướng tối ưu |
|-----|---------|
| ASR | Dùng model cục bộ hoặc node API gần hơn |
| LLM | Chọn model nhỏ hơn hoặc dùng output streaming |
| TTS | Dùng edge TTS hoặc model cục bộ |

---

## IV. Quản lý cấu hình

### Sửa cấu hình

Vào “Quản lý cấu hình” → module tương ứng → sửa mục cấu hình.

<!-- Vị trí ảnh chụp: giao diện quản lý cấu hình -->
> Hình: giao diện quản lý cấu hình

### Bật/tắt cấu hình

Dùng công tắc để điều khiển cấu hình có hiệu lực hay không.

### Đặt cấu hình mặc định

Mỗi module có thể đặt một cấu hình mặc định; khi thiết bị không chỉ định, hệ thống dùng cấu hình mặc định.

---

## Câu hỏi thường gặp

### Q1: Kiểm tra cấu hình thất bại?

1. Kiểm tra kết nối mạng
2. Xác minh API key có đúng không
3. Xem log console của chương trình chính

### Q2: Khôi phục cấu hình mặc định như thế nào?

Xóa file cấu hình trong thư mục `config/`, rồi khởi động lại service.

### Q3: Sau khi sửa cấu hình có cần khởi động lại không?

Phần lớn cấu hình có hiệu lực realtime sau khi sửa; một số cấu hình module có thể cần thiết bị kết nối lại.
