# Hướng dẫn tích hợp interface IndexTTS vLLM

Tài liệu này mô tả yêu cầu interface phía server khi dự án tích hợp `indextts_vllm`, áp dụng cho:

- Suy luận TTS của chương trình chính (`/audio/speech`)
- Giao diện quản trị lấy danh sách âm sắc (`/audio/voices`)
- Nhân bản giọng nói người dùng (`/audio/clone`, dùng cho luồng nhân bản của dự án này)

## 1. Checklist tương thích nhanh

Dịch vụ IndexTTS của bạn tối thiểu cần đáp ứng ba điểm sau:

- Cung cấp `POST /audio/speech`, tham số đầu vào tương thích phong cách OpenAI TTS: `input`, `voice`, `model`
- Cung cấp `GET /audio/voices`, trả về danh sách âm sắc có thể liệt kê được (đối tượng JSON)
- Nếu dùng năng lực “nhân bản giọng nói” của dự án này, cung cấp `POST /audio/clone` (`multipart/form-data`)

Định dạng âm thanh trả về khuyến nghị: `audio/wav` (16-bit PCM).

## 2. Ánh xạ mục cấu hình (quản trị viên -> cấu hình TTS -> IndexTTS(vLLM))

| Trường phía quản trị | Mục đích | Vị trí gửi |
| --- | --- | --- |
| `api_url` | Địa chỉ dịch vụ IndexTTS | Dùng làm URL cơ sở để ghép endpoint |
| `api_key` | Xác thực tùy chọn | `Authorization: Bearer <api_key>` |
| `model` | Tên model | Body request `/audio/speech` trường `model` |
| `voice` | Âm sắc mặc định | Body request `/audio/speech` trường `voice` |
| `frame_duration` | Thời lượng frame (ms) | Tham số cắt frame âm thanh cục bộ |

Ghi chú:

- Khi nhấp dropdown “âm sắc” trong giao diện quản trị, hệ thống sẽ dùng giá trị `api_url` mới nhất trong ô nhập hiện tại để lấy `/audio/voices`.
- `api_url` hỗ trợ nhập địa chỉ cơ sở (ví dụ `http://127.0.0.1:7860`), đồng thời tương thích với địa chỉ đã bao gồm đường dẫn cụ thể (ví dụ `/audio/speech`).

## 3. Yêu cầu interface

### 3.1 `GET /audio/voices`

Mục đích: dropdown “âm sắc” trong trang cấu hình quản trị, tùy chọn âm sắc phía người dùng.

Header request:

- `Accept: application/json`
- `Authorization: Bearer <api_key>` (tùy chọn)

Ví dụ response khuyến nghị:

```json
{
  "demo_speaker": ["assets/speaker/demo.wav"],
  "narrator_cn_female": ["assets/speaker/narrator_cn_female.wav"]
}
```

Yêu cầu:

- Kiểu trả về nên là đối tượng JSON (tên key sẽ được xem là ID âm sắc).
- Dự án này sẽ lọc bỏ âm sắc hệ thống có prefix `indextts_vllm`, sau đó thêm âm sắc nhân bản của người dùng.

### 3.2 `POST /audio/speech`

Mục đích: tổng hợp TTS của chương trình chính, nghe thử sau khi nhân bản.

Header request:

- `Content-Type: application/json`
- `Accept: audio/wav,application/octet-stream,*/*`
- `Authorization: Bearer <api_key>` (tùy chọn)

Ví dụ body request:

```json
{
  "model": "indextts-vllm",
  "input": "Xin chào, chào mừng bạn sử dụng IndexTTS.",
  "voice": "demo_speaker"
}
```

Response:

- Thành công: stream âm thanh nhị phân (khuyến nghị `audio/wav`)
- Thất bại: HTTP 4xx/5xx, kèm thông tin lỗi có thể đọc được

### 3.3 `POST /audio/clone` (cần cho chức năng nhân bản của dự án này)

Mục đích: được gọi khi `/user/voice-clones` gửi tác vụ nhân bản.

Kiểu request: `multipart/form-data`

Trường form:

- `voice`: ID âm sắc mong muốn tạo ra
- `audio`: file âm thanh tham chiếu (wav/mp3/m4a, v.v.)

Ví dụ response:

```json
{
  "voice": "demo_speaker_clone_001",
  "ok": true
}
```

Yêu cầu:

- Response nên chứa trường `voice`; nếu thiếu, dự án này sẽ fallback dùng giá trị trường `voice` trong request.

## 4. Tham khảo tương thích (`api_server.py`)

Có thể tham khảo phong cách triển khai sau:

- `POST /audio/speech`: đọc `input`, `voice`, `model`
- `GET /audio/voices`: trả về dictionary âm sắc khả dụng

Liên kết tham khảo:

- https://github.com/hackers365/index-tts-vllm/blob/master/api_server.py

## 5. Xử lý sự cố thường gặp

### 5.1 Giao diện quản trị báo lỗi khi nhấp dropdown âm sắc

Ưu tiên kiểm tra:

- `api_url` có truy cập được không (giá trị nhập mới nhất)
- `/audio/voices` có trả về đối tượng JSON không
- Có cần `api_key` hay không

### 5.2 Tổng hợp thành công nhưng phát âm thanh bất thường

Ưu tiên kiểm tra:

- Server có trả về WAV chuẩn không (PCM16, sample rate đúng)
- Đường truyền trung gian có transcoding hoặc cắt cụt dữ liệu không
- Header response `Content-Type` có đúng không

### 5.3 Tác vụ nhân bản thất bại

Ưu tiên kiểm tra:

- `/audio/clone` có nhận request multipart gồm `voice + audio` không
- JSON response có parse được không, có chứa `voice` khả dụng không
