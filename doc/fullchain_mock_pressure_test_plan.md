# Phương án mock kiểm thử tải toàn chuỗi VAD/ASR/LLM/TTS (chờ xác nhận)

> Mục tiêu: không gọi dịch vụ ASR/LLM/TTS trả phí thật, nhưng vẫn giữ hành vi WebSocket toàn chuỗi hiện có; hỗ trợ kiểm thử tải đồng thời cao, tiêm độ trễ có kiểm soát và thống kê quan sát được.

## 1. Mục tiêu thiết kế

1. **Chuỗi hoàn chỉnh**: giữ luồng chính “audio thiết bị vào -> VAD -> ASR -> LLM -> TTS -> phát audio xuống”.
2. **Không tốn chi phí bên ngoài**: ASR/LLM/TTS đều trả về dữ liệu mock cục bộ, không truy cập dịch vụ cloud bên thứ ba.
3. **Ít xâm lấn**: mở rộng provider `mock` dựa trên cơ chế provider factory hiện có, hạn chế sửa luồng nghiệp vụ chính.
4. **Kiểm thử tải được và tái lập được**: hỗ trợ trả về cố định, trả về theo template, tiêm lỗi theo xác suất, tiêm độ trễ theo cấu hình.
5. **Có thể đối chiếu với dịch vụ thật**: thông qua chuyển đổi cấu hình, về sau có thể khôi phục provider thật để so sánh hiệu năng.

## 2. Phương án tổng thể

Áp dụng phương án **mock cấp Provider + tái sử dụng client kiểm thử tải**:

- Thêm ba provider:
  - `asr/mock`
  - `llm/mock`
  - `tts/mock`
- Thêm mục cấu hình tương ứng trong backend quản trị (`type=asr|llm|tts`, `provider=mock`).
- Gắn cấu hình mock qua vai trò/agent để thực hiện mock toàn chuỗi trong session.
- Phía kiểm thử tải tiếp tục dùng công cụ kiểm thử tải websocket hiện có (`ws_multi`) để đẩy audio đồng thời.

Cách này đảm bảo:

- WebSocket protocol, state machine của session và logic điều phối message đều chạy qua code path thật.
- Chỉ thay thế lệnh gọi tới dịch vụ cloud bên ngoài, chi phí thấp nhất và rủi ro nhỏ nhất.

## 3. Thiết kế hành vi mock

### 3.1 ASR Mock

Input: stream frame âm thanh (giữ interface hiện có).
Output: văn bản nhận dạng (cố định/luân phiên/theo quy tắc).

Cấu hình khuyến nghị:

- `mode`: `fixed` | `sequence` | `echo_hint`
- `fixed_text`: trả về cố định, ví dụ “Xin chào, đây là văn bản kiểm thử tải”
- `sequence_texts`: mảng văn bản, luân phiên theo request
- `first_token_delay_ms`: mô phỏng độ trễ gói đầu tiên
- `final_delay_ms`: mô phỏng độ trễ gói kết thúc
- `error_rate`: xác suất 0~1 để tiêm lỗi nhận dạng

### 3.2 LLM Mock

Input: văn bản ASR + message ngữ cảnh.
Output: văn bản trả lời (có thể kèm thông tin độ dài ngữ cảnh).

Cấu hình khuyến nghị:

- `mode`: `fixed` | `template` | `echo`
- `fixed_answer`: câu trả lời cố định
- `template`: template, ví dụ `"Đã nhận: {{input}}"`
- `first_token_delay_ms`: độ trễ token đầu tiên
- `stream_chunk_chars`: số ký tự mỗi mảnh khi streaming
- `total_delay_ms`: mô phỏng tổng thời gian hoàn tất
- `error_rate`: xác suất thất bại

### 3.3 TTS Mock

Input: văn bản LLM.
Output: frame Opus/PCM có thể phát được (khuyến nghị ưu tiên Opus để tương thích luồng hiện tại).

Cấu hình khuyến nghị:

- `audio_source`: `builtin_silence` | `builtin_beep` | `file`
- `file_path`: đường dẫn audio có sẵn (wav/opus cục bộ)
- `frame_duration_ms`: độ dài chia frame (ví dụ 20ms)
- `first_frame_delay_ms`: độ trễ frame đầu tiên
- `inter_frame_delay_ms`: độ trễ giữa các frame
- `error_rate`: xác suất thất bại

> Để giảm độ phức tạp, bản đầu tiên khuyến nghị trả về “frame im lặng + độ trễ cố định”; sau đó mới bổ sung “beep/phát lại file”.

## 4. Ma trận kịch bản kiểm thử tải

### Kịch bản A: chuỗi thành công thuần túy (baseline)

- ASR trả văn bản cố định
- LLM trả câu ngắn cố định
- TTS trả frame im lặng
- Mục tiêu: đo đồng thời ổn định tối đa, RT trung bình, P95/P99

### Kịch bản B: chuỗi độ trễ cao

- ASR/LLM/TTS lần lượt tiêm độ trễ 100~500ms
- Mục tiêu: đo ngưỡng timeout và tình trạng dồn hàng đợi

### Kịch bản C: chuỗi tiêm lỗi

- Đặt `error_rate` là 1%/5%/10%
- Mục tiêu: đo khôi phục lỗi, độ ổn định kết nối và chiến lược retry

### Kịch bản D: chuỗi văn bản dài

- LLM xuất văn bản rất dài (ví dụ 500~1500 chữ)
- Mục tiêu: đo chia frame TTS, backpressure khi gửi và độ ổn định bộ nhớ

## 5. Chỉ số và tiêu chuẩn nghiệm thu (khuyến nghị)

Chỉ số cốt lõi:

- Tỷ lệ session thành công (trả được giọng nói)
- Độ trễ frame đầu tiên end-to-end (`listen stop` -> packet audio đầu tiên)
- Độ trễ hoàn tất end-to-end (`listen stop` -> `tts finish`)
- Số session hoạt động mỗi giây / đồng thời đỉnh
- Tỷ lệ lỗi (chia theo giai đoạn ASR/LLM/TTS)
- Tài nguyên dịch vụ: CPU, bộ nhớ, Goroutine, số lần GC

Nghiệm thu khuyến nghị (có thể điều chỉnh sau):

- Tỷ lệ thành công >= 99%
- Ở mức đồng thời mục tiêu, P95 độ trễ frame đầu tiên < 1.5s
- Chạy liên tục 30 phút không rò rỉ bộ nhớ rõ ràng (RSS biến động trong phạm vi kiểm soát)

## 6. Bước triển khai (chia hai giai đoạn)

### Phase 1 (tối thiểu khả dụng, 1~2 ngày)

1. Thêm đăng ký ba mock provider ASR/LLM/TTS.
2. Mỗi provider hỗ trợ trả về cố định + độ trễ cố định + tỷ lệ lỗi.
3. Backend quản trị thêm ba cấu hình mock và có thể đặt làm mặc định.
4. Chạy thông `ws_multi` và xuất kết quả kiểm thử tải baseline.

### Phase 2 (nâng cao, 1~2 ngày)

1. Thêm trả lời theo template, trả lời theo sequence, phát lại file audio.
2. Thêm log chỉ số chi tiết hơn (thời gian theo từng giai đoạn).
3. Thêm script kiểm thử tải (chạy hàng loạt kịch bản + tổng hợp báo cáo).

## 7. Rủi ro và cách né tránh

1. **Định dạng audio không khớp**: output của mock tts cần nhất quán với giải mã downstream hiện tại.
   - Né tránh: bản đầu tiên dùng lại đường mã hóa phổ biến hiện có và thêm log kiểm tra định dạng.
2. **Log quá lớn khi đồng thời cao**: log chi tiết ở tải cao sẽ ảnh hưởng hiệu năng.
   - Né tránh: trong chế độ kiểm thử tải, hạ cấp log level và xuất tổng hợp các chỉ số chính.
3. **Cấu hình nhầm sang dịch vụ thật**: khiến hệ thống vẫn gọi interface bên ngoài.
   - Né tránh: môi trường kiểm thử tải chặn mạng hoặc thêm kiểm tra whitelist provider (không phải mock thì từ chối khởi động).

## 8. Nội dung triển khai sau khi xác nhận

Sau khi xác nhận, có thể sửa code trực tiếp theo danh sách sau:

1. Thêm `internal/domain/asr/mock`, `internal/domain/llm/mock`, `internal/domain/tts/mock`.
2. Gắn provider `mock` vào provider factory / điểm đăng ký pool.
3. Bổ sung mẫu cấu hình mặc định (có thể chọn mock trực tiếp trong backend quản trị).
4. Thêm unit test tối thiểu (ít nhất kiểm thử hành vi provider).
5. Cung cấp danh sách lệnh chạy kiểm thử tải (bậc thang đồng thời + thu thập chỉ số).

---

## Các lựa chọn cần xác nhận

Cần xác nhận 4 điểm sau trước khi bắt đầu chỉnh sửa chính thức:

1. **Độ chi tiết mock**: có đồng ý mock theo cấp provider không (khuyến nghị)?
2. **Output TTS**: bản đầu tiên có chấp nhận “frame im lặng” làm audio mock không (nhanh nhất)?
3. **Mục tiêu đồng thời kiểm thử tải**: trước mắt lấy mức đồng thời nào làm mục tiêu (ví dụ 100/300/500)?
4. **Ngưỡng nghiệm thu**: có dùng tiêu chuẩn nghiệm thu mặc định trong tài liệu này không?
