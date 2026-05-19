# Tài liệu chức năng nhận diện voiceprint

> Nhận diện voiceprint (Speaker Identification) là một chức năng cốt lõi trong dự án xiaozhi-esp32-server-golang, dùng để nhận diện danh tính người dùng phía thiết bị và tự động chuyển âm sắc TTS theo kết quả nhận diện.

---

## I. Tổng quan chức năng

Nhận diện voiceprint trích xuất đặc trưng giọng nói của người dùng (embedding), so khớp với dữ liệu voiceprint đã đăng ký trước để nhận diện danh tính người dùng.

### Năng lực cốt lõi

| Năng lực | Mô tả |
|------|------|
| 🎤 **Đăng ký voiceprint** | Tải mẫu audio người dùng lên, trích xuất và lưu trữ đặc trưng voiceprint |
| 🔍 **Nhận diện voiceprint** | Nhận diện danh tính người nói theo thời gian thực |
| ✅ **Xác minh voiceprint** | Xác minh audio có thuộc về người dùng chỉ định hay không |
| 📡 **Nhận diện streaming** | Nhận diện voiceprint streaming realtime qua WebSocket |
| 🔊 **Chuyển TTS động** | Chuyển âm sắc TTS của người dùng tương ứng theo kết quả nhận diện |

---

## II. Kiến trúc hệ thống

### 2.1 Kiến trúc tổng thể

```text
┌──────────────────┐     ┌──────────────────────┐     ┌──────────────────┐
│   Thiết bị ESP32 │────▶│ xiaozhi-esp32-server │────▶│   voice-server   │
│  (thu audio)     │     │     (service chính)  │     │ (dịch vụ voiceprint) │
└──────────────────┘     └──────────────────────┘     └──────────────────┘
                                                              │
                                                              ▼
                                                      ┌──────────────────┐
                                                      │ Qdrant vector DB │
                                                      │ (lưu embedding)  │
                                                      └──────────────────┘
```

### 2.2 Mô tả component

| Component | Trách nhiệm |
|------|------|
| **xiaozhi-esp32-server** | Service chính, phụ trách kết nối thiết bị, quản lý session, xử lý kết quả nhận diện voiceprint |
| **voice-server (asr_server)** | Dịch vụ voiceprint, phụ trách trích xuất đặc trưng, đăng ký, nhận diện, xác minh |
| **Manager (backend quản trị)** | Web admin, cung cấp API và UI quản lý nhóm voiceprint, quản lý mẫu |
| **Qdrant** | Vector database, lưu vector đặc trưng voiceprint |

---

## III. Mô tả luồng đầy đủ

### 3.1 Luồng đăng ký voiceprint

```text
Người dùng tải audio lên → Manager API → interface đăng ký voice-server → trích xuất embedding → lưu vào Qdrant
                          │
                          ▼
                    Lưu file cục bộ + bản ghi database
```

**Chi tiết bước:**

1. Người dùng tải file audio lên trong giao diện Manager Web (định dạng WAV)
2. Backend Manager tạo UUID duy nhất và lưu file audio vào storage cục bộ
3. Gọi interface `/api/v1/speaker/register` của voice-server
4. voice-server dùng model sherpa-onnx để trích xuất đặc trưng voiceprint (vector 192 chiều)
5. Lưu đặc trưng voiceprint vào Qdrant vector database
6. Manager tạo bản ghi database `SpeakerSample`

### 3.2 Luồng nhận diện voiceprint realtime

```text
ESP32 thu audio → VAD phát hiện giọng nói → gửi đồng thời tới ASR và nhận diện voiceprint
                                              │
                                              ▼
                                    Nhận diện streaming qua WebSocket
                                              │
                                              ▼
                                    Lấy kết quả nhận diện khi giọng nói kết thúc
                                              │
                                              ▼
                                    Chuyển âm sắc TTS theo kết quả nhận diện
```

**Chi tiết bước:**

1. **VAD phát hiện**: audio do ESP32 thu được đi qua VAD (Voice Activity Detection)
2. **Gửi hai kênh**: khi phát hiện giọng nói, dữ liệu audio được gửi đồng thời tới:
   - Dịch vụ ASR (speech to text)
   - Dịch vụ nhận diện voiceprint (nhận diện streaming qua WebSocket)
3. **Xử lý streaming**: dịch vụ nhận diện voiceprint liên tục nhận audio chunk
4. **Lấy kết quả**: khi phát hiện giọng nói kết thúc (im lặng), gọi `FinishAndIdentify` để lấy kết quả nhận diện
5. **Chuyển TTS**: chuyển động sang âm sắc TTS mà người dùng tương ứng đã cấu hình theo kết quả nhận diện

### 3.3 Điều kiện bật

Nhận diện voiceprint chỉ khởi động khi đồng thời thỏa các điều kiện sau:

- `voice_identify.enable = true`: bật nhận diện voiceprint trong cấu hình toàn cục
- Cấu hình thiết bị có cấu hình nhóm voiceprint
- `speakerManager` đã khởi tạo thành công

---

## IV. Mô tả cấu hình

### 4.1 Cấu hình chương trình chính (`config.yaml`)

Thêm cấu hình sau trong `config.yaml`:

```yaml
# Cấu hình nhận diện voiceprint
voice_identify:
  enable: true                              # Có bật nhận diện voiceprint hay không
  base_url: "http://voice-server:8080"      # Địa chỉ dịch vụ voice-server
  threshold: 0.6                            # Ngưỡng nhận diện voiceprint, phạm vi 0.0-1.0
```

| Mục cấu hình | Kiểu | Mặc định | Mô tả |
|--------|------|--------|------|
| `enable` | bool | false | Có bật chức năng nhận diện voiceprint hay không |
| `base_url` | string | - | Địa chỉ HTTP của dịch vụ voice-server |
| `threshold` | float | 0.6 | Ngưỡng nhận diện, giá trị càng cao thì yêu cầu khớp càng nghiêm ngặt |

### 4.2 Cấu hình Docker Compose

#### Biến môi trường service Backend

```yaml
backend:
  environment:
    - SPEAKER_SERVICE_URL=http://voice-server:8080
```

#### Biến môi trường service voice-server

```yaml
voice-server:
  environment:
    - VAD_ASR_SPEAKER_ENABLED=true
    - VAD_ASR_SPEAKER_VECTOR_DB_HOST=qdrant
    - VAD_ASR_SPEAKER_VECTOR_DB_PORT=6334
    - VAD_ASR_SPEAKER_VECTOR_DB_COLLECTION_NAME=speaker_embeddings
    - VAD_ASR_SPEAKER_THRESHOLD=0.6
    - VAD_ASR_LOGGING_LEVEL=info
```

| Biến môi trường | Mô tả |
|----------|------|
| `VAD_ASR_SPEAKER_ENABLED` | Có bật chức năng nhận diện voiceprint hay không |
| `VAD_ASR_SPEAKER_VECTOR_DB_HOST` | Địa chỉ service Qdrant |
| `VAD_ASR_SPEAKER_VECTOR_DB_PORT` | Cổng gRPC của Qdrant |
| `VAD_ASR_SPEAKER_VECTOR_DB_COLLECTION_NAME` | Tên Qdrant Collection |
| `VAD_ASR_SPEAKER_THRESHOLD` | Ngưỡng nhận diện voiceprint |
| `VAD_ASR_LOGGING_LEVEL` | Log level |

---

## V. Mô tả API interface

### 5.1 API backend Manager

#### Quản lý nhóm voiceprint

| Method | Path | Mô tả |
|------|------|------|
| POST | `/api/speaker-groups` | Tạo nhóm voiceprint |
| GET | `/api/speaker-groups` | Lấy danh sách nhóm voiceprint |
| GET | `/api/speaker-groups/:id` | Lấy chi tiết nhóm voiceprint |
| PUT | `/api/speaker-groups/:id` | Cập nhật nhóm voiceprint |
| DELETE | `/api/speaker-groups/:id` | Xóa nhóm voiceprint |
| POST | `/api/speaker-groups/:id/verify` | Xác minh voiceprint |

#### Quản lý mẫu voiceprint

| Method | Path | Mô tả |
|------|------|------|
| POST | `/api/speaker-groups/:id/samples` | Thêm mẫu voiceprint |
| GET | `/api/speaker-groups/:id/samples` | Lấy danh sách mẫu |
| GET | `/api/speaker-samples/:id/audio` | Lấy file audio mẫu |
| DELETE | `/api/speaker-samples/:id` | Xóa mẫu |

### 5.2 API voice-server

#### HTTP interface

| Method | Path | Mô tả |
|------|------|------|
| POST | `/api/v1/speaker/register` | Đăng ký voiceprint |
| POST | `/api/v1/speaker/identify` | Nhận diện voiceprint |
| POST | `/api/v1/speaker/verify` | Xác minh voiceprint |
| GET | `/api/v1/speaker/list` | Lấy toàn bộ speaker |
| DELETE | `/api/v1/speaker/:id` | Xóa speaker |
| GET | `/api/v1/speaker/stats` | Lấy thông tin thống kê |

#### Nhận diện streaming qua WebSocket

**Địa chỉ kết nối:** `ws://voice-server:8080/api/v1/speaker/stream`

Luồng cơ bản:

1. Client tạo kết nối WebSocket tới endpoint streaming
2. Client gửi audio chunk liên tục
3. Khi audio kết thúc, client gửi tín hiệu kết thúc
4. Server trả về kết quả nhận diện speaker và score

---

## VI. Lưu trữ dữ liệu

### 6.1 Dữ liệu Manager

Manager thường lưu các bản ghi liên quan đến nhóm voiceprint và mẫu voiceprint trong database, đồng thời lưu file audio mẫu trong storage cục bộ hoặc volume được mount.

### 6.2 Dữ liệu vector

Dịch vụ voice-server lưu embedding voiceprint vào Qdrant. Collection thường dùng là:

```text
speaker_embeddings
```

Mỗi vector tương ứng với một mẫu hoặc một speaker đã đăng ký, dùng để so khớp cosine/distance khi nhận diện.

---

## VII. Gợi ý triển khai

1. Trước khi bật nhận diện voiceprint trong chương trình chính, hãy đảm bảo voice-server đã khởi động thành công.
2. Nếu dùng Docker Compose, backend và chương trình chính nên cùng trỏ tới cùng một service `voice-server`.
3. Nếu dùng Qdrant, hãy đảm bảo collection đã được tạo hoặc voice-server có quyền tạo collection.
4. Threshold nên được hiệu chỉnh bằng dữ liệu thực tế; giá trị quá thấp dễ nhận nhầm, quá cao dễ không nhận diện được.
5. Mẫu audio đăng ký nên rõ tiếng, ít nhiễu và đủ dài.

---

## VIII. Xử lý sự cố thường gặp

### 8.1 Không nhận diện được speaker

Kiểm tra:

- `voice_identify.enable` đã bật chưa
- Thiết bị đã cấu hình nhóm voiceprint chưa
- voice-server có truy cập được từ chương trình chính không
- Qdrant có chạy và collection có dữ liệu không
- Threshold có quá cao không

### 8.2 Backend không tải được mẫu hoặc nhóm voiceprint

Kiểm tra:

- `SPEAKER_SERVICE_URL` có đúng không
- Volume lưu audio mẫu có mount đúng không
- Log backend có lỗi gọi voice-server không

### 8.3 Kết quả nhận diện không ổn định

Khuyến nghị:

- Tăng chất lượng mẫu audio đăng ký
- Tăng số lượng mẫu cho mỗi speaker
- Điều chỉnh threshold theo dữ liệu thực tế
- Tránh môi trường có nhiều nhiễu hoặc nhiều người nói cùng lúc

---

## IX. Ghi chú

- Chức năng voiceprint chỉ phụ trách nhận diện người nói, không thay thế ASR.
- ASR chuyển giọng nói thành văn bản; voiceprint nhận diện ai đang nói.
- Khi kết hợp với TTS, hệ thống có thể chọn âm sắc TTS phù hợp theo người nói đã nhận diện.
