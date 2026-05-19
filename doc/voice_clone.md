# Hướng dẫn chức năng nhân bản giọng nói

Tài liệu này giới thiệu chức năng **nhân bản giọng nói (Voice Clone)** trong dự án, bao gồm luồng tạo/nghe thử/thử lại của người dùng thường và quản lý hạn mức nhân bản của quản trị viên.

Trang và tài liệu liên quan:

- Quản trị viên `Quản lý cấu hình TTS` (cung cấp cấu hình TTS khả dụng cho người dùng)
- Quản trị viên `Quản lý người dùng -> Hạn mức nhân bản`
- Người dùng thường `Nhân bản giọng nói`
- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)

---

## 1. Tổng quan chức năng

Chức năng nhân bản giọng nói cho phép người dùng tải audio lên (hoặc ghi âm bằng trình duyệt), tạo “âm sắc nhân bản” trên provider TTS được hỗ trợ, sau đó chọn âm sắc này trong agent/vai trò để phát giọng nói.

Provider nhân bản hiện đã được frontend và backend hỗ trợ:

- `minimax`
- `cosyvoice`
- `aliyun_qwen` (Qwen)

Provider TTS không nằm trong danh sách trên, dù dùng được cho tổng hợp TTS thông thường, cũng không thể dùng cho nhân bản giọng nói.

---

## 2. Vai trò và quyền hạn

### 2.1 Người dùng thường

Có thể:

- Tạo âm sắc nhân bản
- Xem trạng thái tác vụ nhân bản
- Nghe thử audio gốc và audio nhân bản
- Sửa tên bản nhân giọng
- Thử lại tác vụ thất bại

### 2.2 Quản trị viên

Có thể:

- Cấu hình và bật provider TTS hỗ trợ nhân bản
- Đặt hạn mức nhân bản cho từng người dùng theo `cấu hình TTS` (tùy chọn)

---

## 3. Điều kiện trước khi dùng

Trước khi sử dụng, hãy xác nhận:

1. Quản trị viên đã tạo và bật ít nhất một cấu hình TTS (provider là `minimax` / `cosyvoice` / `aliyun_qwen`)
2. Người dùng thường có thể thấy cấu hình TTS đó trong trang “nhân bản giọng nói”
3. (Tùy chọn) quản trị viên đã phân bổ hạn mức nhân bản cho người dùng đó

Ghi chú:

- Nếu chưa cấu hình hạn mức, hệ thống mặc định tương thích hành vi cũ, thường được xem là “không giới hạn”

---

## 4. Luồng sử dụng của người dùng thường

Lối vào:

- `Người dùng thường -> Nhân bản giọng nói`

## 4.1 Tạo âm sắc nhân bản

Nhấp `Tạo âm sắc nhân bản`, rồi điền:

- `Tên nhân bản` (tùy chọn; nếu để trống sẽ dùng tên file)
- `Cấu hình TTS` (bắt buộc chọn cấu hình hỗ trợ nhân bản)
- `Nguồn audio` (tải audio lên / ghi âm bằng trình duyệt)
- `Văn bản tương ứng với audio` (có bắt buộc hay không tùy năng lực provider)
- `Ngôn ngữ văn bản` (như `zh-CN` / `en-US`)

Sau khi gửi có thể xuất hiện hai kết quả:

- Thành công ngay (ít gặp)
- Trả về “đã gửi tác vụ nhân bản, đang xử lý ở nền” (thường gặp, bất đồng bộ)

## 4.2 Xem trạng thái tác vụ

Danh sách sẽ hiển thị:

- Provider
- Cấu hình TTS liên kết
- ID âm sắc nhân bản
- Trạng thái tác vụ
- Nguyên nhân thất bại (nếu có)
- Thời gian tạo

Các trạng thái thường gặp có thể hiểu là:

- Đang xếp hàng / đang xử lý
- Đã hoàn tất (có thể nghe thử)
- Thất bại (có thể xem nguyên nhân thất bại và thử lại)

## 4.3 Nghe thử và quản lý

Mỗi bản ghi nhân bản hỗ trợ các thao tác sau:

- `Audio gốc`: phát mẫu audio người dùng đã gửi
- `Nghe thử bản nhân`: phát âm sắc nhân bản do provider trả về (chỉ hiển thị khi trạng thái thành công)
- `Sửa`: sửa tên bản nhân
- `Nhân bản lại`: gửi lại tác vụ thất bại (chỉ hiển thị ở trạng thái thất bại)

---

## 5. Khác biệt provider và lưu ý

## 5.1 Minimax

Frontend và backend sẽ kiểm tra ràng buộc audio, quy tắc thường gặp:

- Định dạng audio thường yêu cầu `WAV`
- Thời lượng audio khuyến nghị/yêu cầu không dưới `10 giây`

Trang sẽ hiển thị gợi ý trong khu vực tải lên/ghi âm và chặn gửi khi thời lượng không đủ.

## 5.2 CosyVoice

Đặc điểm:

- Hỗ trợ nhân bản
- Trong tình huống thường gặp, yêu cầu điền “văn bản tương ứng với audio” (do interface năng lực provider trả về)

Có bắt buộc thực tế hay không hãy lấy theo gợi ý năng lực provider hiện tại trên trang.

## 5.3 Qwen (`aliyun_qwen`)

Đặc điểm:

- Hỗ trợ nhân bản
- Hỗ trợ nhiều định dạng audio hơn (như `WAV/MP3/M4A`, lấy theo gợi ý trên trang)
- Sau khi chọn loại âm sắc nhân bản này, runtime sẽ tự động chuyển sang model nhân bản tương ứng (frontend sẽ hiển thị gợi ý)

---

## 6. Quản lý hạn mức nhân bản (quản trị viên)

Lối vào:

- `Quản trị viên -> Quản lý người dùng -> Hạn mức nhân bản`

Quản trị viên có thể cấu hình hạn mức nhân bản cho một người dùng thường theo `ID cấu hình TTS`:

- `-1`: không giới hạn số lần
- `0`: cấm tạo
- `số nguyên dương`: số lần nhân bản tối đa

Thống kê hạn mức thường tính theo “số lần gửi tác vụ nhân bản” (thử lại sau thất bại cũng nên được đưa vào chiến lược đếm, hãy dùng theo quy tắc nghiệp vụ hiện tại).

---

## 7. Mô tả interface (phía người dùng)

### 7.1 Phát hiện năng lực

- `GET /user/voice-clone/capabilities?provider=<provider>`

Mục đích:

- Lấy provider có được bật hay không
- Có yêu cầu điền transcript hay không
- Phạm vi độ dài văn bản
- Danh sách ngôn ngữ hỗ trợ

### 7.2 Bản ghi nhân bản và thao tác tác vụ

- `POST /user/voice-clones` (tạo nhân bản, `multipart/form-data`)
- `GET /user/voice-clones` (danh sách)
- `PUT /user/voice-clones/:id` (sửa tên)
- `POST /user/voice-clones/:id/retry` (thử lại khi thất bại)
- `GET /user/voice-clones/:id/preview` (nghe thử âm sắc nhân bản)

### 7.3 Quản lý audio gốc

- `GET /user/voice-clones/:id/audios`
- `GET /user/voice-clones/audios/:audio_id/file`

---

## 8. Mô tả interface (hạn mức quản trị viên)

- `GET /admin/users/:id/voice-clone-quotas`
- `PUT /admin/users/:id/voice-clone-quotas`

---

## 9. Câu hỏi thường gặp và xử lý sự cố

### 9.1 Trang không thấy cấu hình TTS để chọn

Kiểm tra:

1. Quản trị viên đã bật cấu hình TTS chưa
2. Provider TTS có thuộc danh sách hỗ trợ nhân bản không (`minimax/cosyvoice/aliyun_qwen`)
3. Người dùng hiện tại có quyền truy cập cấu hình đó không

### 9.2 Khi gửi báo “provider này yêu cầu điền văn bản tương ứng với audio”

Điều này cho biết năng lực provider yêu cầu transcript bắt buộc; hãy bổ sung văn bản tương ứng với audio rồi gửi lại.

### 9.3 Khi gửi báo không đủ hạn mức

Quản trị viên cần vào `Quản lý người dùng -> Hạn mức nhân bản` để tăng hạn mức cho `ID cấu hình TTS` tương ứng của người dùng đó hoặc đặt thành `-1`.

### 9.4 Nhân bản thành công nhưng không nghe thử được

Kiểm tra:

1. Trạng thái tác vụ đã hoàn tất chưa
2. Interface preview của provider có bình thường không
3. Trình duyệt có chặn tự động phát audio không (nhấp thủ công nút phát để thử lại)

---

## 10. Gợi ý sử dụng

- Chuẩn bị cấu hình TTS riêng cho từng kịch bản (tiện kiểm soát hạn mức và quy kết chi phí)
- Audio gửi lên nên dùng giọng người rõ, môi trường ít nhiễu
- Transcript nên nhất quán với nội dung audio, giúp hiệu quả và độ ổn định nhân bản tốt hơn
