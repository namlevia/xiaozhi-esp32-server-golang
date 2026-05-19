# Hướng dẫn chức năng chợ MCP

Tài liệu này giới thiệu chức năng **chợ MCP** trong trang quản trị: cách kết nối chợ MCP bên thứ ba, tổng hợp danh sách dịch vụ, nhập cấu hình dịch vụ và đưa vào danh sách dịch vụ MCP toàn cục của hệ thống.

Tài liệu liên quan:

- [Mô tả kiến trúc MCP](./mcp.md)
- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)

---

## 1. Định vị chức năng

Chợ MCP dùng để giải quyết vấn đề “kết nối dịch vụ MCP từ xa mất nhiều thao tác”, hỗ trợ:

- Cấu hình nhiều kết nối chợ MCP, ví dụ ModelScope.
- Tổng hợp danh mục dịch vụ từ nhiều chợ.
- Xem chi tiết dịch vụ như endpoint, giao thức truyền tải.
- Nhập nhanh cấu hình dịch vụ vào hệ thống hiện tại.
- Bật, tắt, chỉnh sửa hoặc xóa các dịch vụ đã nhập.

Dịch vụ sau khi nhập sẽ tham gia merge vào cấu hình dịch vụ MCP toàn cục của hệ thống, cùng có hiệu lực với các dịch vụ MCP được cấu hình thủ công.

---

## 2. Quyền hạn và điểm vào

Quyền hạn:

- Chỉ quản trị viên được thao tác.

Điểm vào trong trang quản trị:

- `Quản trị viên -> Chợ MCP`

Trang gồm hai tab:

- `Khám phá chợ`
- `Dịch vụ đã nhập`

---

## 3. Khái niệm cốt lõi

### 3.1 Chợ MCP

Đại diện cho một nguồn danh mục chợ MCP có thể truy cập, gồm:

- Tên chợ.
- Định danh provider.
- URL danh mục (`catalog_url`).
- Mẫu URL chi tiết (`detail_url_template`, tùy chọn).
- Token xác thực, nếu có.
- Trạng thái bật/tắt.

### 3.2 Danh sách dịch vụ tổng hợp

Hệ thống sẽ kéo danh mục dịch vụ từ các chợ đang bật và hiển thị tổng hợp, hỗ trợ:

- Tìm kiếm theo tên dịch vụ, mô tả hoặc Service ID.
- Xem chi tiết.
- Nhập cấu hình.

Khi một phần chợ kéo dữ liệu thất bại, trang sẽ hiển thị danh sách cảnh báo “một phần chợ kéo dữ liệu thất bại” và không ảnh hưởng việc hiển thị kết quả từ các chợ khác.

### 3.3 Dịch vụ đã nhập

Dịch vụ sau khi nhập sẽ trở thành mục cấu hình độc lập trong hệ thống và có thể tham gia kết nối MCP lúc runtime. Các cấu hình được hỗ trợ gồm:

- Tên.
- Loại truyền tải (`sse` / `streamablehttp`).
- URL.
- Headers dạng JSON.
- Chợ nguồn và định danh provider, là thông tin metadata tùy chọn.
- Trạng thái bật/tắt.

---

## 4. Quy trình thao tác thường dùng cho quản trị viên

## 4.1 Thêm kết nối chợ MCP

Trong tab `Khám phá chợ`, bấm `Thêm kết nối`, rồi điền:

- `Provider`: ưu tiên chọn preset provider tích hợp sẵn để tự điền mẫu URL danh mục.
- `Tên`.
- `URL danh mục`.
- `Mẫu URL chi tiết`, tùy chọn.
- `Bật`.
- `Token`, nếu chợ yêu cầu.

Nên kiểm tra kết nối trước khi lưu và sử dụng.

## 4.2 Kiểm tra kết nối chợ

Trong menu thao tác của danh sách chợ, bấm `Kiểm tra`:

- Thành công sẽ trả về số lượng dịch vụ có thể phát hiện.
- Thất bại sẽ báo lỗi kết nối danh mục hoặc lỗi xác thực.

Phù hợp để kiểm tra:

- Token không hợp lệ.
- URL danh mục sai.
- Chợ tạm thời không khả dụng.

## 4.3 Duyệt và tìm kiếm dịch vụ tổng hợp

Trong khu vực `Danh sách dịch vụ tổng hợp`, có thể:

- Nhập từ khóa để tìm kiếm dịch vụ.
- Xem kết quả theo phân trang.
- Bấm `Chi tiết` để xem thông tin endpoint của dịch vụ.

Trang chi tiết dịch vụ thường gồm:

- Tên dịch vụ.
- Chợ nguồn.
- Service ID.
- Mô tả.
- Danh sách endpoint, gồm giao thức truyền tải và URL.

## 4.4 Nhập nhanh cấu hình dịch vụ

Trong popup chi tiết dịch vụ, bấm `Nhập cấu hình dịch vụ và hot reload`:

- Hệ thống sẽ tạo một hoặc nhiều cấu hình dịch vụ đã nhập dựa trên chi tiết dịch vụ.
- Sau khi nhập thành công, danh sách `Dịch vụ đã nhập` sẽ được refresh.
- Trang sẽ chuyển sang tab `Dịch vụ đã nhập`.

“Hot reload” nghĩa là sau khi nhập cấu hình xong, dịch vụ có thể tham gia tập dịch vụ runtime ngay mà không cần khởi động lại backend.

## 4.5 Thêm hoặc sửa dịch vụ đã nhập thủ công

Trong tab `Dịch vụ đã nhập`, có thể bấm `Thêm dịch vụ` để nhập thủ công hoặc chỉnh sửa mục đã nhập.

Các trường quan trọng:

- `Truyền tải`: hiện hỗ trợ `SSE` và `StreamableHTTP`.
- `URL`: điểm vào của dịch vụ MCP từ xa.
- `Headers(JSON)`: dùng để mang thông tin xác thực, ví dụ `Authorization`.
- `Bật`: nếu tắt, dịch vụ sẽ không tham gia tập dịch vụ khả dụng lúc runtime.

`Headers(JSON)` phải là object JSON, ví dụ:

```json
{
  "Authorization": "Bearer <token>"
}
```

---

## 5. Quan hệ với cấu hình MCP toàn cục

Chợ MCP không thay thế trang `Cấu hình MCP`, mà là một nguồn bổ sung.

Tập dịch vụ MCP toàn cục khả dụng lúc runtime được merge từ hai phần:

- Dịch vụ toàn cục do quản trị viên bảo trì thủ công trong trang `Cấu hình MCP`.
- Dịch vụ được nhập từ chợ MCP và đang bật.

Vì vậy cách dùng được khuyến nghị là:

1. Dùng chợ MCP để nhanh chóng phát hiện và nhập dịch vụ.
2. Bật và chọn dịch vụ theo nhu cầu trong `Cấu hình MCP` hoặc trong cấu hình trợ lý.

---

## 6. API backend

Các API liên quan phía quản trị, yêu cầu quyền quản trị viên:

### 6.1 Quản lý kết nối chợ

- `GET /admin/mcp-markets`
- `POST /admin/mcp-markets`
- `PUT /admin/mcp-markets/:id`
- `DELETE /admin/mcp-markets/:id`
- `POST /admin/mcp-markets/:id/test`

### 6.2 Khám phá chợ và xem chi tiết

- `GET /admin/mcp-market/providers`
- `GET /admin/mcp-market/services`
- `GET /admin/mcp-market/services/:market_id/*service_id`
- `POST /admin/mcp-market/import`

### 6.3 Quản lý dịch vụ đã nhập

- `GET /admin/mcp-market/imported-services`
- `POST /admin/mcp-market/imported-services`
- `PUT /admin/mcp-market/imported-services/:id`
- `DELETE /admin/mcp-market/imported-services/:id`

---

## 7. Câu hỏi thường gặp và cách kiểm tra

### 7.1 Danh sách tổng hợp rỗng

Thứ tự kiểm tra:

1. Kiểm tra kết nối chợ đã bật chưa.
2. Chạy `Kiểm tra` cho chợ đó.
3. Kiểm tra Token có hợp lệ không.
4. Kiểm tra URL danh mục và mẫu URL chi tiết có đúng không.

### 7.2 Nhập thành công nhưng runtime không thấy dịch vụ

Nguyên nhân thường gặp:

- Dịch vụ đã nhập đang bị tắt.
- Công tắc MCP toàn cục đang tắt.
- Trợ lý có cấu hình `mcp_service_names` nhưng không chứa tên dịch vụ đó.

### 7.3 Khi sửa chợ, để trống Token thì sao?

Trong popup chỉnh sửa, Token để trống thường có nghĩa là “không sửa Token hiện có”; giao diện sẽ hiển thị trạng thái đã được che một phần.

---

## 8. Khuyến nghị sử dụng

- Ưu tiên dùng preset provider tích hợp sẵn để giảm vấn đề do khác biệt field của API danh mục.
- Đặt tên thống nhất cho các dịch vụ cần dùng ổn định lâu dài sau khi nhập, để trợ lý có thể chọn theo tên.
- Với dịch vụ từ xa ở môi trường production, nên cấu hình xác thực bằng `Headers(JSON)` và có quy trình xoay vòng token.
