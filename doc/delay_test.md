#### Kết quả kiểm thử độ trễ

Có thể phản hồi trong khoảng 1-1.3s; nếu dùng model nhỏ hơn thì có thể nhanh hơn.

asr: funasr
llm: Aliyun API qwen2.5-72b-instruct
tts: cosyvoice

```text
time="2025-05-22 19:33:09.940" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1394 ms" caller="client.go:428"
time="2025-05-22 19:33:33.458" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1237 ms" caller="client.go:428"
time="2025-05-22 19:33:52.596" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1190 ms" caller="client.go:428"
time="2025-05-22 19:34:12.272" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1361 ms" caller="client.go:428"
time="2025-05-22 19:34:31.598" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1347 ms" caller="client.go:428"
time="2025-05-22 19:35:00.281" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1194 ms" caller="client.go:428"
time="2025-05-22 19:35:24.418" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 975 ms" caller="client.go:428"
time="2025-05-22 19:35:49.868" level=debug msg="Tổng thời gian từ khi nhận audio kết thúc đến frame đầu tiên asr->llm->tts: 1150 ms" caller="client.go:428"
```

---

## Kiểm thử trong backend quản trị

Gói khởi động một lệnh và triển khai Docker đều tích hợp backend quản trị Web, cung cấp giao diện kiểm thử trực quan.

Hỗ trợ các loại kiểm thử sau:

| Loại kiểm thử | Mô tả |
|---------|------|
| VAD | Kết nối và thời gian phản hồi của phát hiện hoạt động giọng nói |
| ASR | Kết nối và độ trễ gói đầu tiên của nhận diện giọng nói |
| LLM | Kết nối và độ trễ gói đầu tiên của suy luận model lớn |
| TTS | Kết nối và độ trễ gói đầu tiên của tổng hợp giọng nói |
| OTA | Kiểm tra kết nối MQTT/UDP |

Cách sử dụng chi tiết xem: **[Hướng dẫn sử dụng trang quản trị →](manager_console_guide.md)**
