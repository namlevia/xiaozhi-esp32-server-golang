# Môi trường chạy

#### I. Triển khai FunASR

Tham khảo [tài liệu triển khai Docker của FunASR](https://github.com/modelscope/FunASR/blob/main/runtime/docs/SDK_advanced_guide_online_zh.md).

#### II. Clone code

```bash
git clone 'https://github.com/hackers365/xiaozhi-esp32-server-golang'
```

#### III. Cấu hình `config/config.yaml`, xem chi tiết tại [mô tả cấu hình](config.md)

Các mục chính cần sửa như sau:

```yaml
# 1. Nhận diện giọng nói ASR
asr:
  provider: "funasr"
  funasr:
    host: "127.0.0.1"      # IP của dịch vụ FunASR WebSocket đã triển khai
    port: "10096"          # Cổng WebSocket của FunASR đã triển khai
    mode: "offline"        # Chế độ, dùng offline là được
    # ...

# 2. TTS
tts:
  provider: "xiaozhi"      # Loại TTS sử dụng; khuyến nghị doubao_ws, cũng có thể chọn edge miễn phí
  doubao_ws:
    appid: "6886011847"                         # appid của bạn
    access_token: "access_token"                # access token của bạn
    cluster: "volcano_tts"
    voice: "zh_female_wanwanxiaohe_moon_bigtts" # Âm sắc, mặc định là Wanwan Xiaohe
    ws_host: "openspeech.bytedance.com"
    use_stream: true
  edge:
    voice: "zh-CN-XiaoxiaoNeural"
    rate: "+0%"
    volume: "+0%"
    pitch: "+0Hz"
    connect_timeout: 10
    receive_timeout: 60
  # ....

# 3. LLM model lớn
llm:
  provider: "deepseek"                        # Provider, tương ứng với key bên dưới
  deepseek:
    type: "openai"                            # Kiểu interface server tương thích
    model_name: "Pro/deepseek-ai/DeepSeek-V3" # Tên model
    api_key: "api_key"                        # API key
    base_url: "https://api.siliconflow.cn/v1" # Interface service, mặc định SiliconFlow
    max_tokens: 500
  # ...
```

#### IV. Khởi động Docker

Tại root dự án, khởi động Docker và mount thư mục `config` cùng cổng (http/websocket:8989, các cổng khác map theo nhu cầu):

```bash
docker run -itd --name xiaozhi_server -v $(pwd)/config:/workspace/config -p 8989:8989 hackers365/xiaozhi_server:latest

# Nếu không kết nối được ở trong nước, dùng mirror sau:
docker run -itd --name xiaozhi_server -v $(pwd)/config:/workspace/config -p 8989:8989 docker.jsdelivr.fyi/hackers365/xiaozhi_server:latest
```

**Mô tả hỗ trợ ten_vad:**

- Docker image đã tự bao gồm file thư viện ten_vad, không cần mount thêm
- Nếu dùng ten_vad làm VAD provider, chỉ cần đặt `vad.provider: "ten_vad"` trong file cấu hình

Lúc này có thể kết nối tới:

```text
ws://IP_máy:8989/xiaozhi/v1/
```

để chat.

# Môi trường phát triển

```bash
docker run -itd --name xiaozhi_server_golang -v $(pwd):/workspace/ -p 8989:8989 hackers365/xiaozhi_golang:0.1

# Nếu không kết nối được ở trong nước, dùng mirror sau:
docker run -itd --name xiaozhi_server_golang -v $(pwd):/workspace/ -p 8989:8989 docker.jsdelivr.fyi/hackers365/xiaozhi_golang:0.1

go build -o xiaozhi_server cmd/server/*.go
```

**Mô tả ten_vad trong môi trường phát triển:**

- Image môi trường phát triển đã bao gồm dependency biên dịch và runtime của ten_vad
- Nếu cần dùng ten_vad trong môi trường phát triển, hãy đảm bảo thư mục `lib/ten-vad` tồn tại ở root dự án
- Khi biên dịch, hệ thống sẽ tự dùng header và file thư viện của ten_vad
