# Hướng dẫn sử dụng repo độc lập MCP Audio Server

## Tổng quan

MCP Audio Server đã được tách thành repo độc lập; khuyến nghị dùng trực tiếp dự án độc lập này để chạy, debug và phát triển tiếp.

Tên dự án độc lập:

- `mcp_audio_server`
- `github.com/hackers365/mcp_audio_server`

Mục tiêu cốt lõi là minh họa:

- Cách trả về `ResourceLink` thông qua công cụ `musicPlayer`
- Cách đọc dữ liệu âm thanh phân trang thông qua `resource/read`
- Cách dùng `BlobResourceContents` để trả về mảnh âm thanh mã hóa base64

Repo độc lập này vừa có thể chạy trực tiếp, vừa phù hợp làm template tích hợp.

## Cách dùng khuyến nghị

Khuyến nghị dùng MCP Audio Server trong repo độc lập.

Nên lấy repo độc lập trước, rồi vào thư mục dự án:

```bash
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server
```

## Năng lực service cung cấp

Service hiện chỉ expose hai loại năng lực:

1. Công cụ `musicPlayer`
2. Tài nguyên `resource://read_from_http`

### `musicPlayer`

- Tác dụng: tìm nhạc theo tên bài hát người dùng nhập và trả về tài nguyên có thể phát
- Tham số đầu vào: `query`
- Trả về: `ResourceLink`

Ý nghĩa các trường quan trọng trong `ResourceLink` trả về:

- `URI`: `resource://read_from_http`
- `Name`: tên bài hát thực tế
- `Description`: URL audio thực tế
- `MIMEType`: `audio/mpeg`

### `resource://read_from_http`

- Tác dụng: đọc dữ liệu audio từ xa theo phân trang
- Cách gọi: thông qua `resource/read`
- Tham số truyền qua `Arguments`

Định dạng tham số request:

```json
{
  "url": "URL audio thực tế",
  "start": 0,
  "end": 102400
}
```

Mô tả tham số:

- `url`: địa chỉ audio thực tế, lấy từ `ResourceLink.Description`
- `start`: offset byte bắt đầu
- `end`: offset byte kết thúc, không bao gồm vị trí này

Nội dung trả về là `BlobResourceContents`:

- `MIMEType`: `audio/mpeg`
- `Blob`: dữ liệu nhị phân audio sau khi mã hóa base64

Khi đọc hết dữ liệu, server sẽ trả về `[DONE]` sau khi mã hóa base64 làm dấu kết thúc.

## Luồng gọi

Luồng đầy đủ như sau:

1. Client gọi `musicPlayer`
2. Công cụ tìm bài hát và trả về `ResourceLink`
3. Client gọi `resource/read` tới `resource://read_from_http`
4. Mỗi lần truyền `url`, `start`, `end` qua `Arguments`
5. Server trả về `BlobResourceContents` mã hóa base64
6. Client decode rồi phát liên tục theo audio stream cho đến khi nhận `[DONE]`

## Cách chạy

Repo độc lập hỗ trợ hai kiểu truyền tải:

- Mặc định: `stdio`
- Tùy chọn: HTTP Streamable MCP

### Chế độ stdio

Khởi động trực tiếp:

```bash
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server
go run .
```

### Chế độ HTTP

Chỉ định rõ HTTP transport:

```bash
cd mcp_audio_server
go run . -t http
```

Hoặc:

```bash
cd mcp_audio_server
go run . --transport http
```

Thông tin lắng nghe ở chế độ HTTP:

- Cổng: `3001`
- Đường dẫn: `/mcp`
- Địa chỉ đầy đủ: `http://localhost:3001/mcp`

## Lưu ý sử dụng hiện tại

Repo độc lập có thể build và chạy trực tiếp. Trước khi dùng, nên chú ý các điểm sau:

- Tìm kiếm bài hát và lấy URL thật phụ thuộc vào `github.com/scroot/music-sd/pkg/netease` và `github.com/scroot/music-sd/pkg/qq`
- Độ ổn định của kết quả tìm kiếm nhạc và link có thể phát phụ thuộc vào năng lực site bên ngoài
- Nếu chuyển dự án độc lập này vào dự án khác, thường cần đồng bộ bổ sung các dependency và logic tìm kiếm nói trên

Nếu mục tiêu là tích hợp nhanh công cụ audio của riêng bạn, khuyến nghị ưu tiên tái sử dụng protocol và luồng dữ liệu thay vì tái sử dụng trực tiếp phần tìm kiếm bài hát.

## Các phần nên giữ nguyên khi dùng làm template tích hợp

Nếu muốn biến dự án độc lập này thành MCP Audio Server của riêng bạn, khuyến nghị giữ các quy ước protocol sau:

- Công cụ trả về `ResourceLink`
- `resource/read` dùng `Arguments` để đọc phân trang
- Dữ liệu audio trả về qua `BlobResourceContents.Blob`
- Nội dung `Blob` giữ dạng mã hóa base64
- MIME type audio nhất quán với dữ liệu thực tế; repo độc lập hiện dùng `audio/mpeg`
- Khi stream kết thúc, trả về `[DONE]`

Như vậy có thể giữ tương thích với logic tiêu thụ audio trong service chính hiện tại.

## Tương thích với service chính hiện tại

Logic tiêu thụ công cụ MCP loại audio trong service chính hiện tại đã xử lý theo cách sau:

- Nhận diện `ResourceLink`
- Gọi phân trang `resource/read` bằng `Arguments`
- Decode `BlobResourceContents.Blob`
- Parse định dạng audio theo MIME type
- Phát liên tục cho đến khi đọc xong

Vì vậy, hình thái protocol của dự án độc lập này có thể tiếp tục được dùng làm template tham khảo cho công cụ MCP loại audio.
