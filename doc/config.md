# Mô tả file cấu hình xiaozhi-esp32-server-golang

File cấu hình này là cấu hình chính của dịch vụ backend IoT giọng nói AI, bao gồm toàn bộ tham số cốt lõi như khởi động service, kết nối protocol, năng lực AI, log và MCP.

## Mô tả các mục cấu hình chính

- **server/pprof**: cấu hình phân tích hiệu năng, khuyến nghị bật khi phát triển/debug.
- **chat**: tham số chat, điều khiển thời gian idle và im lặng của session.
- **auth**: công tắc xác thực người dùng, có thể mở rộng hệ thống quyền về sau.
- **system_prompt**: prompt hệ thống toàn cục, ảnh hưởng phong cách chat của LLM.
- **log**: cấu hình đường dẫn, level, rotation log, v.v.
- **redis**: nếu cần dùng Redis để lưu trữ thì cấu hình mục này.
- **websocket**: IP và cổng lắng nghe của dịch vụ WebSocket.
- **mqtt**: tham số kết nối MQTT server bên ngoài.
- **mqtt_server**: tham số MQTT server tích hợp (TLS tùy chọn).
- **udp**: tham số liên quan đến UDP server.
- **vad**: cấu hình phát hiện hoạt động giọng nói (VAD), hỗ trợ `webrtc_vad` / `silero_vad`.
- **asr**: cấu hình nhận diện giọng nói tự động (ASR), hỗ trợ `funasr` / `aliyun_funasr` / `doubao`.
- **tts**: cấu hình tổng hợp giọng nói (TTS), hỗ trợ nhiều engine như `doubao`, `edge`, `xiaozhi`, v.v.
- **llm**: cấu hình mô hình ngôn ngữ lớn (LLM), hỗ trợ nhiều model tương thích OpenAI.
- **vision**: cấu hình model thị giác.
- **ota**: thông tin interface OTA trả về, thích ứng nhiều môi trường.
- **wakeup_words**: danh sách wake word.
- **mcp**: cấu hình kết nối đa protocol MCP, hỗ trợ toàn cục và phía thiết bị.
- **enable_greeting**: có bật lời chào khi khởi động hay không.

### Khuyến nghị chỉnh sửa

- Chỉ cần điều chỉnh IP, cổng, khóa, API Key và các tham số theo môi trường triển khai thực tế.
- Ý nghĩa chi tiết của tham số hãy xem comment trong từng module.
- Nếu cần mở rộng năng lực AI, có thể bổ sung provider và tham số trong các module `llm` / `tts` / `vad` / `asr` / `vision`.

## Ví dụ file cấu hình

```yaml
# Cấu hình phân tích hiệu năng/pprof
server:
  pprof:
    enable: false  # Có bật phân tích hiệu năng pprof hay không
    port: 6060     # Cổng lắng nghe pprof

# Tham số chat
chat:
  max_idle_duration: 30000        # Thời gian idle tối đa (ms)
  chat_max_silence_duration: 200  # Thời gian im lặng tối đa (ms)

# Công tắc xác thực người dùng
auth:
  enable: false

# Prompt hệ thống toàn cục
system_prompt: "Bạn là LeviaTech AI, một trợ lý tiếng Việt thân thiện, tự nhiên và hữu ích. Hãy trả lời ngắn gọn, rõ ràng, giữ giọng nói ấm áp như đang trò chuyện với người thật. Ưu tiên tiếng Việt, không dùng emoji, mã nguồn hoặc thẻ XML nếu người dùng không yêu cầu."

# Cấu hình log
log:
  path: "../logs/"
  file: "server.log"
  level: "debug"
  max_age: 3
  rotation_time: 10  # Thời gian rotation log
  stdout: true

# Cấu hình lưu trữ Redis (nếu có Redis thì cấu hình; không cấu hình vẫn có thể chạy)
redis:
  host: "127.0.0.1"
  port: 6379
  password: "ticket_dev"
  db: 0
  key_prefix: "xiaozhi"

# Cấu hình lắng nghe dịch vụ WebSocket
websocket:
  host: "0.0.0.0"
  port: 8989

# Tham số kết nối MQTT server bên ngoài (địa chỉ MQTT server cần kết nối; nếu mqtt_server bên dưới bật true thì có thể đặt là localhost)
mqtt:
  broker: "127.0.0.1"      # Địa chỉ MQTT server
  type: "tcp"              # Kiểu tcp hoặc ssl
  port: 2883
  client_id: "xiaozhi_server"
  username: "admin"        # Tên người dùng
  password: "test!@#"      # Mật khẩu

# Tham số MQTT server tích hợp
mqtt_server:
  enable: true             # Có bật hay không
  listen_host: "0.0.0.0"   # IP lắng nghe
  listen_port: 2883        # Cổng lắng nghe
  client_id: "xiaozhi_server"
  username: "admin"        # Tên người dùng quản trị viên
  password: "test!@#"      # Mật khẩu quản trị viên
  tls:
    enable: false          # Có bật TLS hay không
    port: 8883             # Cổng cần lắng nghe
    pem: "config/server.pem"  # File pem
    key: "config/server.key"  # File key

# Mô tả hành vi:
# - Khi mqtt_server.enable=true, mqtt_server tích hợp sẽ publish message vòng đời qua
#   /p2p/device_public/_server/lifecycle sau khi thiết bị kết nối/ngắt kết nối.
# - Chương trình chính sẽ dựa trên message vòng đời này để tạo trước hoặc tái sử dụng MQTT transport,
#   ánh xạ trạng thái online của thiết bị, đồng thời preheat MCP phía thiết bị theo best-effort.
# - Các hành vi này không thêm mục cấu hình mới; hello vẫn phụ trách thương lượng cấp chat như audio_params và thông tin UDP.

# Cấu hình UDP server
udp:
  external_host: "127.0.0.1"  # IP UDP server trả về trong message hello
  external_port: 8990         # Cổng UDP server trả về trong message hello
  listen_host: "0.0.0.0"      # IP lắng nghe
  listen_port: 8990           # Cổng lắng nghe

# Cấu hình phát hiện hoạt động giọng nói (VAD, hỗ trợ nhiều provider)
vad:
  provider: "webrtc_vad"  # Tùy chọn webrtc_vad/silero_vad
  webrtc_vad:
    pool_min_size: 5
    pool_max_size: 1000
    pool_max_idle: 100
    vad_sample_rate: 16000
    vad_mode: 2
  silero_vad:
    model_path: "config/models/vad/silero_vad.onnx"
    threshold: 0.5
    min_silence_duration_ms: 100
    sample_rate: 16000     # chỉ 16000
    channels: 1
    pool_size: 10
    acquire_timeout_ms: 3000

# Cấu hình nhận diện giọng nói tự động (ASR)
asr:
  provider: "funasr"  # funasr / aliyun_funasr / doubao
  funasr:
    host: "127.0.0.1"
    port: "10096"
    mode: "offline"
    sample_rate: 16000     # chỉ 16000
    chunk_size: [5, 10, 5]
    chunk_interval: 10
    max_connections: 5
    timeout: 30
    auto_end: true  # Có tự động kết thúc hay không

  # Aliyun FunASR
  aliyun_funasr:
    api_key: ""
    ws_url: "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
    model: "fun-asr-realtime"
    format: "pcm"
    sample_rate: 16000     # chỉ 16000
    vocabulary_id: ""
    disfluency_removal_enabled: false
    timeout: 30

# Cấu hình tổng hợp giọng nói (TTS)
tts:
  provider: "doubao_ws"  # Chọn loại TTS: doubao, doubao_ws, cosyvoice, xiaozhi, v.v.
  doubao:
    appid: "appid của bạn"
    access_token: "access_token"    # Cần sửa thành token của bạn
    model: "seed-tts-1.1"
    voice: "BV001_streaming"
    api_url: "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
  doubao_ws:
    appid: "appid của bạn"          # Cần sửa thành appid của bạn
    access_token: "access_token"    # Cần sửa thành token của bạn
    model: "seed-tts-1.1"
    resource_id: ""                 # Khuyến nghị điền instance ID trong console, như TTS-SeedTTS2.xxxxx
    voice: ""
    ws_url: "wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream"
  cosyvoice:
    api_url: "https://tts.linkerai.cn/tts"  # Địa chỉ
    spk_id: "spk_id"                        # Âm sắc
    frame_duration: 60
    target_sr: 24000
    audio_format: "mp3"
    instruct_text: "Xin chào"
  edge:
    voice: "zh-CN-XiaoxiaoNeural"
    rate: "+0%"
    volume: "+0%"
    pitch: "+0Hz"
    connect_timeout: 10
    receive_timeout: 60
  edge_offline:
    server_url: "ws://localhost:8080/tts"
    timeout: 30
    sample_rate: 16000     # chỉ 16000
    channels: 1
    frame_duration: 20
  xiaozhi:
    server_addr: "wss://api.tenclass.net/xiaozhi/v1/"
    device_id: "ba:8f:17:de:94:94"
    client_id: "e4b0c442-98fc-4e1b-8c3d-6a5b6a5b6a6d"
    token: "test-token"

# Cấu hình mô hình ngôn ngữ lớn (LLM, bổ sung nhiều provider)
llm:
  provider: "qwen_72b"
  deepseek:
    type: "openai"
    model_name: "Pro/deepseek-ai/DeepSeek-V3"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  deepseek2_5:
    type: "openai"
    model_name: "deepseek-ai/DeepSeek-V2.5"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  qwen_72b:
    type: "openai"
    model_name: "Qwen/Qwen2.5-72B-Instruct"
    api_key: "api_key"
    base_url: "https://api.siliconflow.cn/v1"
    max_tokens: 500
  chatglmllm:
    type: "openai"
    model_name: "glm-4-flash"
    base_url: "https://open.bigmodel.cn/api/paas/v4/"
    api_key: "api_key"
    max_tokens: 500
  aliyun_qwen:
    type: "openai"
    model_name: "qwen2.5-72b-instruct"
    base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
    api_key: "api_key"
    max_token: 500
  doubao_deepseek:
    type: "openai"
    model_name: "deepseek-v3"
    api_key: "api_key"
    base_url: "https://ark.cn-beijing.volces.com/api/v3"
    max_tokens: 500

# Cấu hình model thị giác
vision:
  enable_auth: false
  vision_url: "http://192.168.208.214:8989/xiaozhi/api/vision"
  vllm:
    provider: "aliyun_vision"
    aliyun_vision:
      type: "openai"
      model_name: "qwen-vl-plus-latest"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      api_key: "api_key"
      max_token: 500
    doubao_vision:
      type: "openai"
      model_name: "doubao-1.5-vision-lite-250315"
      api_key: "api_key"
      base_url: "https://ark.cn-beijing.volces.com/api/v3"
      max_tokens: 500

# Cấu hình môi trường interface OTA
ota:
  test:
    websocket:
      url: "ws://192.168.208.214:8989/xiaozhi/v1/"
    mqtt:
      endpoint: "192.168.208.214"
  external:
    websocket:
      url: "wss://www.youdomain.cn/go_ws/xiaozhi/v1/"
    mqtt:
      endpoint: "www.youdomain.cn"

# Danh sách wake word
wakeup_words: ["LeviaTech", "LeviaTech AI", "Xin chào LeviaTech"]

# Cấu hình kết nối đa protocol MCP
mcp:
  global:
    enabled: true
    servers:
      - name: "filesystem"
        sse_url: "http://localhost:3001/sse"
        enabled: true
      - name: "memory"
        sse_url: "http://localhost:3002/sse"
        enabled: false
    reconnect_interval: 5
    max_reconnect_attempts: 10
  device:
    enabled: true
    websocket_path: "/xiaozhi/mcp/"
    max_connections_per_device: 5

# Có bật lời chào khi khởi động hay không
enable_greeting: true
```
