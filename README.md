# 🚀 xiaozhi-esp32-server-golang

> **Xiaozhi AI Backend for ESP32 Devices**

---

## Giới thiệu dự án | Project Overview

xiaozhi-esp32-server-golang là dịch vụ backend AI hiệu năng cao, xử lý streaming đầu-cuối, được thiết kế cho các kịch bản IoT và giọng nói thông minh. Dự án được phát triển bằng Go, tích hợp các năng lực cốt lõi như ASR (nhận dạng giọng nói), LLM (mô hình ngôn ngữ lớn) và TTS (tổng hợp giọng nói), hỗ trợ truy cập đa giao thức với mức đồng thời lớn, phục vụ tương tác thoại AI cho thiết bị thông minh và thiết bị biên.

## 🇻🇳 Ghi chú nhanh cho bản fork tiếng Việt

### Tổng quan ngắn

Bản fork này giữ nguyên lõi backend AI cho ESP32 nhưng bổ sung tài liệu và giao diện tiếng Việt để dễ triển khai, cấu hình và vận hành hơn. Hệ thống tập trung vào luồng thoại thời gian thực, hỗ trợ WebSocket hoặc MQTT+UDP, đồng thời tách rõ các lớp VAD, ASR, LLM, TTS, OTA, console quản trị và các module mở rộng như MCP, voice clone, speaker identification.

### Chạy nhanh sau khi fork

- Ưu tiên chạy bằng Docker Compose để lên nhanh MySQL, main server, backend và frontend.
- Xem hướng dẫn tiếng Việt tại [docs/vi_quickstart.md](docs/vi_quickstart.md).
- Sau khi hệ thống lên, kiểm tra lại các field quan trọng trong `config/config.yaml` như `manager.backend_url`, `udp.external_host`, `ota.test.websocket.url`, `ota.external.websocket.url` và provider ASR / LLM / TTS đang dùng.

### Cảnh báo production security

Trước khi đưa lên môi trường thật, bắt buộc thay toàn bộ giá trị mẫu hoặc mặc định như:

- `manager.auth_token`
- `manager.endpoint_auth_token`
- `mqtt.password`
- `mqtt_server.password`
- `mqtt_server.signature_key`
- `ota.signature_key`
- mọi `api_key`, `api_secret`, `access_token`

Không nên giữ `127.0.0.1`, IP mẫu, domain mẫu hoặc credential ví dụ trong cấu hình OTA / MQTT / UDP khi thiết bị kết nối từ mạng khác máy chủ.

### Các module chính

- **VAD**: phát hiện hoạt động giọng nói
- **ASR**: nhận dạng giọng nói thành văn bản
- **LLM**: suy luận hội thoại và điều phối tool
- **TTS**: tổng hợp giọng nói phản hồi
- **WebSocket / MQTT + UDP**: lớp transport cho thiết bị
- **Manager console**: frontend/backend quản trị cấu hình, thiết bị và kiểm thử
- **OTA**: cấp cấu hình kết nối và hỗ trợ cập nhật từ xa
- **MCP / OpenClaw / Knowledge Base / Voice Clone / Speaker Identification**: các tính năng mở rộng theo nhu cầu triển khai

---

## ✨ Tính năng chính | Key Features

- ⚡ **Chuỗi thoại AI streaming đầu-cuối**: toàn bộ luồng ASR → LLM → TTS được xử lý theo kiểu streaming để giảm độ trễ và hỗ trợ tương tác thời gian thực.
- 🎙️ **Nhận diện người nói và chuyển TTS động**: tự động đổi giọng TTS theo danh tính người nói để tạo trải nghiệm cá nhân hóa hơn.
- 🔌 **Lớp transport trừu tượng**: thống nhất WebSocket / MQTT UDP dưới cùng một abstraction để dễ mở rộng giao thức và gắn logic chính.
- 📬 **Xử lý theo hàng đợi tin nhắn**: LLM và TTS được xử lý bất đồng bộ qua queue, thuận tiện để chèn logic nghiệp vụ.
- 🌐 **Kết nối đa giao thức với độ đồng thời cao**: hỗ trợ số lượng lớn thiết bị kết nối và nhận đẩy tin nhắn cùng lúc.
- ♻️ **Pool tài nguyên và tái sử dụng kết nối hiệu quả**: giảm thời gian phản hồi và tăng thông lượng hệ thống.
- 🤖 **Tích hợp nhiều engine AI**: dựa trên framework Eino, hỗ trợ FunASR, OpenAI-compatible, Ollama, Doubao, EdgeTTS, CosyVoice và nhiều engine khác.
- 🧩 **Kiến trúc mô-đun, dễ mở rộng**: các module cốt lõi như VAD / ASR / LLM / TTS / MCP / vision hoạt động độc lập và có thể cắm/rút.
- 🎵 **MCP Audio Server**: hỗ trợ phân trang tài nguyên âm thanh, xử lý streaming, phát nhạc và điều khiển âm lượng.
- 🦞 **Tích hợp trợ lý OpenClaw**: tạo OpenClaw Endpoint riêng cho từng trợ lý, hỗ trợ xem trạng thái kết nối, test hội thoại, và route bằng từ khóa vào/ra chế độ (mặc định “bật OpenClaw/vào OpenClaw” và “tắt OpenClaw/thoát OpenClaw”).
- 🖥️ **Console web quản trị đầy đủ**: có trình hướng dẫn cấu hình, test khả dụng VAD/ASR/LLM/TTS toàn chuỗi, quản lý thiết bị, inject message, giám sát độ trễ thời gian thực và kiểm tra OTA.
- 🧠 **Tính năng nghiệp vụ nâng cao**: gồm chợ MCP và import, nhân bản giọng nói, knowledge base (Dify / RAGFlow / WeKnora), và debug gọi MCP từ thiết bị / trợ lý.
- 📦 **Triển khai một chạm thuận tiện**: có gói aio build sẵn dùng ngay (main server + console + speaker service), Docker one-click, và hỗ trợ build local trên Linux / Windows / macOS.
- 🔐 **Hệ thống an toàn và phân quyền** (đang lên kế hoạch): đã chừa sẵn interface cho xác thực người dùng và phân quyền.

---

[Phân tích kiến trúc trên DeepWiki](https://deepwiki.com/hackers365/xiaozhi-esp32-server-golang)

## 🚀 Bắt đầu nhanh | Quick Start

### Cách 1: gói khởi động một chạm (khuyến nghị)

Tải gói nén phù hợp với nền tảng của bạn, giải nén và chạy trực tiếp:

- **Trang Release**: <https://github.com/hackers365/xiaozhi-esp32-server-golang/releases>
- **Hướng dẫn sử dụng**: [doc/quickstart_bundle_tutorial.md](doc/quickstart_bundle_tutorial.md)

Sau khi khởi động, truy cập **http://<IP-hoặc-domain-của-server>:8080** để vào console web và cấu hình.

### Cách 2: triển khai bằng Docker

- [Docker Compose (có console)](doc/docker_compose.md)
- [Docker (không có console)](doc/docker.md)

### Cách 3: biên dịch cục bộ

Phù hợp cho môi trường phát triển hoặc khi bạn cần tùy biến sâu quá trình build.

**Cài dependency** (ví dụ trên Ubuntu)

```bash
# Go 1.20+
# Codec Opus
sudo apt-get install -y pkg-config libopus0 libopusfile-dev

# ONNX Runtime（1.21.0）
wget https://github.com/microsoft/onnxruntime/releases/download/v1.21.0/onnxruntime-linux-x64-1.21.0.tgz
tar -xzf onnxruntime-linux-x64-1.21.0.tgz
sudo cp -r onnxruntime-linux-x64-1.21.0/include/* /usr/local/include/onnxruntime/
sudo cp -r onnxruntime-linux-x64-1.21.0/lib/* /usr/local/lib/
sudo ldconfig

# Dependency runtime của ten_vad
sudo apt install -y libc++1 libc++abi1
```

> 📖 Xem [config.md](doc/config.md) để biết đầy đủ dependency và cấu hình cho Windows/macOS.

Xem [doc/compile_deploy.md](doc/compile_deploy.md) để tham khảo quy trình build tách riêng main program, frontend/backend console, speaker service và đóng gói AIO.

Có thể tham khảo thêm [tài liệu chính thức của FunASR](https://github.com/modelscope/FunASR/blob/main/runtime/docs/SDK_advanced_guide_online_zh.md) khi triển khai.

**Biên dịch và khởi động**

```bash
# Biên dịch
go build -o xiaozhi_server ./cmd/server/

# Khởi động (xem chi tiết file cấu hình tại config/config.yaml)
./xiaozhi_server -c config/config.yaml
```

---

## 📚 Điều hướng tài liệu | Docs

### Liên quan đến triển khai
- [Hướng dẫn gói khởi động một chạm](doc/quickstart_bundle_tutorial.md)
- [Triển khai Docker Compose](doc/docker_compose.md)
- [Triển khai Docker](doc/docker.md)
- [Hướng dẫn biên dịch và triển khai](doc/compile_deploy.md)
- [Giải thích chi tiết cấu hình](doc/config.md)

### Hướng dẫn sử dụng
- [Hướng dẫn dùng backend quản trị](doc/manager_console_guide.md)
- [Dịch vụ WebSocket và cấu hình OTA](doc/websocket_server.md)
- [Cấu hình MQTT + UDP](doc/mqtt_udp.md)
- [Giao thức MQTT UDP](doc/mqtt_udp_protocol.md)

### Module tính năng
- [Năng lực thị giác](doc/vision.md)
- [Nhận diện người nói](doc/speaker_identification.md)
- [Kiến trúc MCP](doc/mcp.md)
- [Tài nguyên âm thanh MCP](doc/mcp_resource.md)
- [Chợ MCP (khám phá/import/hot reload)](doc/mcp_market.md)
- [Tích hợp trợ lý OpenClaw (Endpoint/route từ khóa/test hội thoại)](doc/openclaw_integration.md)
- [Nhân bản giọng nói (thao tác người dùng và quota quản trị)](doc/voice_clone.md)
- [Kho tri thức (cấu hình provider/đồng bộ/test recall/RAG)](doc/knowledge_base.md)
- [Gọi MCP từ thiết bị/trợ lý (Endpoint/Tools/Call)](doc/mcp_remote_call_agent_device.md)

### Kết nối thiết bị
- [Hướng dẫn kết nối phía ESP32](doc/esp32_xiaozhi_backend_guide.md)
- [Giải thích xác thực OTA MQTT](doc/ota_mqtt_auth.md)

---

## 🧩 Kiến trúc mô-đun | Module Overview

| Mô-đun | Mô tả chức năng | Tech stack |
|------|----------|--------|
| VAD | Phát hiện hoạt động giọng nói | Silero VAD / WebRTC VAD / ten_vad |
| ASR | Nhận dạng giọng nói | FunASR / Doubao ASR |
| LLM | Suy luận bằng mô hình ngôn ngữ lớn | Tương thích framework Eino, OpenAI, Ollama, v.v. |
| TTS | Tổng hợp giọng nói | Doubao / EdgeTTS / CosyVoice |
| MCP | Truy cập đa giao thức, khám phá/import chợ MCP, debug gọi từ thiết bị/trợ lý | MCP Server / Endpoint / MCP Market / SSE / StreamableHTTP / WebSocket Controller / MCP Tool Call |
| OpenClaw | Endpoint theo từng trợ lý, chuyển chế độ bằng từ khóa vào/ra, chuyển tiếp và test hội thoại | OpenClaw WebSocket / Agent Endpoint / Chat Router |
| Vision | Xử lý thị giác | Doubao / Alibaba Cloud Vision |
| Nhận diện người nói | Nhận diện danh tính người nói | sherpa-onnx + cơ sở dữ liệu vector |
| Nhân bản giọng nói | Tạo và nghe thử giọng nhân bản ở phía người dùng | Minimax / CosyVoice / Qwen |
| Kho tri thức (RAG) | Đồng bộ tài liệu, test recall và truy xuất đối thoại | Dify / RAGFlow / WeKnora |

---

## 📈 Hiệu năng và kiểm thử | Performance & Testing

- [Báo cáo kiểm thử độ trễ](doc/delay_test.md)
- Backend quản trị cung cấp điểm vào để test khả dụng và độ trễ cho VAD / ASR / LLM / TTS

---

## 🛠️ Lộ trình | Roadmap

- AI chủ động

---

## 🤝 Đóng góp | Contributing

Hoan nghênh gửi Issue, PR hoặc đề xuất!

---

## 📄 License

MIT License

---

## 📬 Liên hệ | Contact

**WeChat cá nhân**: hackers365 (thêm WeChat để được mời vào nhóm trao đổi)

![WeChat cá nhân](https://github.com/user-attachments/assets/6b8d3d11-7bf5-4fa4-a73e-5109019dab85)


**Làm mã nguồn mở không dễ; sự ủng hộ của bạn sẽ giúp dự án tiếp tục được cập nhật**

<img width="250" height="250" alt="eab0f4d3d8b6f977863a7bef36e3d64b" src="https://github.com/user-attachments/assets/9a949cb3-d788-446b-a0b9-8542edbb0842" />




---

> © 2024 xiaozhi-esp32-server-golang
