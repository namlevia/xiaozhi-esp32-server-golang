# Hướng dẫn triển khai gói khởi động một lệnh

## Tải xuống

Truy cập [trang Release](https://github.com/hackers365/xiaozhi-esp32-server-golang/releases) để tải gói tương ứng với nền tảng:

| Nền tảng | Tên file |
|-----|-------|
| Windows | `xiaozhi-server-windows-xxx.zip` |
| Linux | `xiaozhi-server-linux-xxx.tar.gz` |
| macOS | `xiaozhi-server-macos-xxx.tar.gz` |

---

## Giải nén và cấu trúc thư mục

Sau khi giải nén, cấu trúc thư mục như sau:

```text
xiaozhi-aio/
├── xiaozhi_server          # Chương trình chính
├── config/                 # Thư mục file cấu hình
├── models/                 # Thư mục model, dùng khi chạy ASR/TTS cục bộ
└── data/                   # Thư mục dữ liệu
```

---

## Khởi động dịch vụ

### Windows

Nhấp đúp `start.bat`.

### Linux

```bash
# Dependency runtime của ten_vad
sudo apt install -y libc++1 libc++abi1

chmod +x xiaozhi_server
LD_LIBRARY_PATH="$PWD/ten-vad/lib/Linux/x64:${LD_LIBRARY_PATH:-}" ./xiaozhi_server
```

### macOS

```bash
chmod +x xiaozhi_server
./build/macos/fix_rpath.sh ./xiaozhi_server
./xiaozhi_server
```

Nếu cấu trúc thư mục được giữ như sau:

```text
./xiaozhi_server
./ten-vad/lib/macOS/ten_vad.framework
```

thì sau khi chạy `fix_rpath.sh`, gói macOS mặc định không cần tự đặt `DYLD_FRAMEWORK_PATH` nữa.

Nếu đang debug từ thư mục tạm của IDE, hoặc đã di chuyển binary thủ công khiến cấu trúc thư mục tương đối bị hỏng, có thể dùng cách fallback:

```bash
DYLD_FRAMEWORK_PATH="$PWD/ten-vad/lib/macOS" ./xiaozhi_server
```

Nếu tự đóng gói bản phát hành macOS trong repo source, cần chạy thêm một lần trước khi phát hành:

```bash
./build/macos/fix_rpath.sh ./xiaozhi_server
```

Bước này sửa `rpath` trong binary từ đường dẫn source trên máy build thành `@executable_path/ten-vad/lib/macOS`, để gói phát hành có thể chạy trực tiếp khi cấu trúc thư mục đúng.

---

## Bước tiếp theo

### 1. Truy cập Web console

Mở trình duyệt và truy cập: **http://<IP hoặc domain server>:8080**

<!-- Vị trí ảnh chụp: màn hình đăng nhập -->
> Hình: màn hình đăng nhập Web console

### 2. Cấu hình dịch vụ

Lần đầu sử dụng, hãy hoàn tất cài đặt theo wizard cấu hình. Xem thêm:

**[Hướng dẫn sử dụng trang quản trị →](manager_console_guide.md)**

---

## Dịch vụ nhận diện giọng nói, tùy chọn

Chương trình đã tích hợp dịch vụ nhận diện giọng nói.

---

## Câu hỏi thường gặp

### Q1: Sau khi khởi động không truy cập được Web console?

Kiểm tra firewall và đảm bảo cổng 8080 có thể truy cập.

### Q2: Khởi động lại dịch vụ như thế nào?

Tắt chương trình rồi chạy lại. File cấu hình được lưu trong thư mục `config/`.

### Q3: Xem log như thế nào?

Console sẽ xuất log realtime. Nếu cần lưu lại, có thể redirect:

```bash
./xiaozhi_server > server.log 2>&1
```
