### Kiểm thử tải

```
root@hackers365-System-Product-Name:~# docker run -itd --name websocket_meter docker.jsdelivr.fyi/hackers365/xiaozhi_websocket_client
87311584e5fef592f32e0b7d7062d9053e956d5e0d50edb220370ff37d2293ac
root@hackers365-System-Product-Name:~#
root@hackers365-System-Product-Name:~# docker exec -it websocket_meter /bin/bash
root@87311584e5fe:/workspace#
root@87311584e5fe:/workspace# ./ws_multi -h
Usage of ./ws_multi:
  -count int
        Số lượng client (default 10)
  -device string
        ID thiết bị
  -server string
        Địa chỉ server (default "ws://localhost:8989/xiaozhi/v1/")
  -text string
        Nội dung trò chuyện, nhiều câu cách nhau bằng dấu phẩy sẽ được gửi lần lượt (default "Xin chào")
root@87311584e5fe:/workspace# ./ws_multi -count 1 -server wss://joeyzhou.chat/ws/xiaozhi/v1/ -text "Xin chào, đang làm gì vậy, cùng ra ngoài chơi nhé"
Chạy client Xiaozhi
Server: wss://joeyzhou.chat/ws/xiaozhi/v1/
Số lượng client: 1
Nội dung gửi: Xin chào, đang làm gì vậy, cùng ra ngoài chơi nhé
2025-05-27 09:54:51.095 [info] [audio_utils.go:199] TTS cloud tới frame đầu tiên: 532 ms
2025-05-27 09:54:51.098 [info] [audio_utils.go:269] TTS cloud tới khi decode xong frame đầu tiên: 535 ms
2025-05-27 09:54:51.401 [info] [cosyvoice.go:306] TTS từ input tới khi lấy xong dữ liệu MP3: 838 ms
2025-05-27 09:54:51.748 [info] [audio_utils.go:199] TTS cloud tới frame đầu tiên: 344 ms
2025-05-27 09:54:51.752 [info] [audio_utils.go:269] TTS cloud tới khi decode xong frame đầu tiên: 347 ms
2025-05-27 09:54:51.901 [info] [cosyvoice.go:306] TTS từ input tới khi lấy xong dữ liệu MP3: 497 ms
2025-05-27 09:54:52.292 [info] [audio_utils.go:199] TTS cloud tới frame đầu tiên: 387 ms
2025-05-27 09:54:52.296 [info] [audio_utils.go:269] TTS cloud tới khi decode xong frame đầu tiên: 391 ms
2025-05-27 09:54:52.628 [info] [cosyvoice.go:306] TTS từ input tới khi lấy xong dữ liệu MP3: 723 ms
Client 0 bắt đầu chạy
Client 0 đã kết nối tới server: wss://joeyzhou.chat/ws/xiaozhi/v1/
Nhận message: {Type:hello Text: State: SessionID:cafd2800-1979-06d5-19cf-b8bf53bb55dc Transport:websocket AudioFormat:<nil>}
Gửi frame Opus: 20
Gửi frame Opus: 50
Gửi frame Opus: 59
```

#### Mô tả tổng quan

1. Chương trình sẽ dựa trên text người dùng nhập, gọi interface TTS để sinh dữ liệu audio rồi lần lượt gửi tới server.
2. Thống kê thời gian bắt đầu từ `type: listen, state: stop` và dừng khi nhận được frame audio đầu tiên từ server.

#### Mô tả tham số

- `-count`: số lượng kết nối đồng thời.
- `-device`: mặc định tự sinh `deviceId`; nếu chỉ định tham số này thì `-count` phải bằng 1.
- `-server`: địa chỉ WebSocket server.
- `-text`: nội dung cần gửi, phân tách bằng dấu `,` và gửi lặp.

#### Mô tả output

Có thể redirect output vào file log, rồi dùng lệnh như sau để theo dõi:

```bash
tail -f xx.log | grep 'thời gian phản hồi trung bình'
```
