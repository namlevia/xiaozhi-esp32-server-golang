# Dịch vụ mock ASR/LLM/TTS độc lập, không sửa chương trình chính

Phương án này cung cấp một tiến trình mock chạy độc lập, dùng để thay thế các dịch vụ cloud ASR/LLM/TTS thật khi chạy kiểm thử tải.

## 1. Khởi động

```bash
go run ./cmd/mock_ai_server \
  -addr :18080 \
  -asr-text "Xin chào, đây là kết quả nhận dạng mock cho kiểm thử tải" \
  -llm-reply "Đây là phản hồi LLM mock" \
  -tts-mode silence
```

Kiểm tra sức khỏe:

```bash
curl http://127.0.0.1:18080/healthz
```

## 2. Interface được cung cấp

- `ws://127.0.0.1:18080/asr/`
  - Tương thích đầu vào WebSocket kiểu FunASR, nhận frame audio nhị phân.
  - Sau khi nhận `{"is_speaking": false}`, trả kết quả nhận dạng cuối cùng.

- `POST http://127.0.0.1:18080/v1/chat/completions`
  - Interface tương thích OpenAI Chat Completions.
  - Hỗ trợ `stream=false/true`.

- `POST http://127.0.0.1:18080/v1/audio/speech`
  - Interface tương thích OpenAI TTS.
  - Trả `audio/wav`, dạng im lặng hoặc beep.

## 3. Gợi ý cấu hình chương trình chính, chỉ đổi cấu hình và không sửa code

### ASR (FunASR)

- `host=127.0.0.1`
- `port=18080`
- Đường dẫn giao thức đang dùng theo triển khai hiện tại là `ws://host:port/`; nếu tầng cấu hình yêu cầu path, dùng `/asr/`.

> Nếu adapter ASR hiện tại phụ thuộc cứng vào root path `ws://host:port/`, có thể chuyển tiếp `/` sang `/asr/` ở tầng gateway.

### LLM tương thích OpenAI

- Chọn provider `eino` với `type=openai`.
- `base_url=http://127.0.0.1:18080/v1`.
- `api_key` có thể là giá trị bất kỳ nhưng không rỗng.
- `model_name` có thể là giá trị bất kỳ, ví dụ `mock-gpt`.

### TTS tương thích OpenAI

- Chọn provider `openai`.
- `api_url=http://127.0.0.1:18080/v1/audio/speech`.
- `response_format=wav`.
- `api_key` có thể là giá trị bất kỳ nhưng không rỗng.

## 4. Tham số có thể điều chỉnh

```bash
-asr-delay-ms         # Độ trễ trả kết quả cuối của ASR
-llm-first-delay-ms   # Độ trễ token đầu tiên của LLM
-llm-chunk-delay-ms   # Độ trễ giữa các chunk LLM streaming
-tts-first-delay-ms   # Độ trễ gói đầu tiên của TTS
-tts-mode             # silence|beep
-tts-duration-ms      # Thời lượng audio trả về
```

## 5. Gợi ý kiểm thử tải

1. Trước tiên kiểm tra một kết nối cục bộ để chắc chắn thiết bị đi hết chuỗi xử lý và nhận được audio.
2. Sau đó dùng `ws_multi` để chạy các bậc đồng thời, ví dụ 50/100/200/500.
3. Dùng các tổ hợp delay khác nhau để mô phỏng dao động của phụ thuộc bên ngoài, rồi quan sát P95/P99 và tỷ lệ lỗi.

## 6. Có cần tối ưu `ws_multi` không?

Kết luận: **nên tối ưu nhẹ, không bắt buộc refactor**. Hiện tại có thể dùng trực tiếp cho kiểm thử tải, nhưng để đo “hiệu năng chương trình chính” thay vì “nút cổ chai của client kiểm thử tải”, nên bổ sung các năng lực sau:

1. **Thêm chế độ phát lại audio thuần, ưu tiên làm trước**
   - Cách thường gặp hiện tại là TTS cục bộ trước rồi mới đẩy audio, khiến thời gian TTS của client bị trộn vào kết quả.
   - Nên thêm `-audio_file`/`-audio_dir` để gửi trực tiếp frame opus đã encode sẵn, hoặc frame sau khi chuyển wav sang opus.

2. **Xuất thống kê độ trễ có cấu trúc**
   - Thêm thống kê thời gian tới frame đầu tiên, thời gian hoàn tất toàn chuỗi, và phân loại mã lỗi.
   - Nên xuất JSONL để dễ hậu xử lý và tổng hợp P95/P99.

3. **Điều tiết kết nối và tốc độ gửi**
   - Thêm cơ chế tạo kết nối theo lô, ví dụ mỗi giây khởi động N client, tránh tạo kết nối đồng thời làm khuếch đại jitter phía client.
   - Thêm tham số jitter gửi gói để mô phỏng mạng thiết bị thật.

4. **Cho phép cấu hình retry và timeout khi thất bại**
   - Ví dụ `-dial_timeout`, `-read_timeout`, `-retry`, giúp kiểm thử tải dài ổn định hơn.

5. **Thu thập chỉ số tài nguyên, tùy chọn**
   - Ghi lại CPU/bộ nhớ của chính client để phân biệt “nút cổ chai server” và “nút cổ chai máy kiểm thử tải”.

Với phương án “dịch vụ mock độc lập” này, `ws_multi` **không đổi vẫn chạy được**, nhưng nên làm ít nhất mục 1 và 2 để kết luận kiểm thử tải đáng tin hơn.
