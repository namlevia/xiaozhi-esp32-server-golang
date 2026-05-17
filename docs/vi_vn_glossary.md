# Glossary Việt hóa

Tài liệu này dùng làm chuẩn thống nhất khi Việt hóa giao diện, thông báo, tài liệu và nội dung hướng dẫn trong repo.

## Nguyên tắc chung

- Không dịch identifier trong code: tên biến, hàm, struct, class, component, file, package, module.
- Không dịch API contract: route, HTTP method, query/body field, JSON key, response key, enum value, DB field, migration name.
- Không dịch config key: tên key trong YAML/JSON/TOML, env var, flag name, provider value, model id, topic, queue name.
- Không dịch tên công nghệ hoặc giao thức phổ biến: MQTT, UDP, WebSocket, HTTP, JWT, API, OpenAPI, MCP, OTA, VAD, ASR, LLM, TTS.
- Có thể dịch phần mô tả, label, placeholder, button, tooltip, error message, log vận hành, comment cấu hình và tài liệu.
- Giữ nguyên token mẫu trong prompt/template như `{{assistant_name}}`, `<token>`, `:id`, `{device_id}`.
- Với tên thương hiệu/provider như Doubao, Xunfei, Qwen, OpenAI, ModelScope, giữ nguyên hoặc dùng tên đang hiển thị trong sản phẩm.
- Nếu một chuỗi vừa chứa key kỹ thuật vừa có mô tả, chỉ dịch phần mô tả.

## Tone tiếng Việt

- Dùng tiếng Việt thân thiện, rõ ràng, thiên về kỹ thuật-vận hành.
- Ưu tiên câu ngắn, trực tiếp, dễ hiểu trên UI.
- Với lỗi API, nêu nguyên nhân và hành động gợi ý khi có thể.
- Với cấu hình, dùng giọng hướng dẫn: “Nhập…”, “Chọn…”, “Bật…”, “Kiểm tra…”.
- Tránh dịch quá khẩu ngữ trong màn hình admin; giữ cảm giác chuyên nghiệp.
- Thống nhất cách xưng hô trung tính, không dùng “bạn” quá nhiều nếu không cần.

## Bảng thuật ngữ

| Nhóm | English | 中文 | Tiếng Việt đề xuất | Ghi chú |
|---|---|---|---|---|
| device | Device | 设备 | Thiết bị | Danh từ chung cho ESP32/client. |
| device | Device ID | 设备ID | ID thiết bị | Không dịch key `device_id`. |
| device | Device name | 设备名称 | Tên thiết bị | UI label. |
| device | Device nickname | 设备昵称 | Biệt danh thiết bị | Dùng khi là tên hiển thị thân thiện. |
| device | Online device | 在线设备 | Thiết bị đang online | Có thể rút gọn “Đang online” trong badge. |
| device | Offline device | 离线设备 | Thiết bị offline | Giữ “offline” nếu UI đã dùng online/offline. |
| device | Bind device | 绑定设备 | Liên kết thiết bị | Hành động gán thiết bị vào agent/user. |
| device | Activate device | 激活设备 | Kích hoạt thiết bị | Dùng cho flow activation/challenge code. |
| device | Activation code | 激活码 | Mã kích hoạt | Không dịch field/key. |
| agent | Agent | 智能体 | Trợ lý | Có thể dùng “Agent” trong tài liệu kỹ thuật nếu cần phân biệt. |
| agent | Agent name | 智能体名称 | Tên trợ lý | UI label. |
| agent | Default agent | 默认智能体 | Trợ lý mặc định |  |
| agent | Edit agent | 编辑智能体 | Chỉnh sửa trợ lý |  |
| agent | Agent configuration | 智能体配置 | Cấu hình trợ lý |  |
| agent | Prompt | 提示词 | Prompt | Giữ “prompt”; có thể chú thích “lời nhắc hệ thống”. |
| agent | Role prompt | 角色提示词 | Prompt vai trò |  |
| role | Role | 角色 | Vai trò |  |
| role | Global role | 全局角色 | Vai trò toàn cục |  |
| role | User role | 用户角色 | Vai trò người dùng |  |
| role | Permission | 权限 | Quyền |  |
| role | Admin | 管理员 | Quản trị viên |  |
| role | Normal user | 普通用户 | Người dùng thường |  |
| config | Configuration | 配置 | Cấu hình |  |
| config | Config item | 配置项 | Mục cấu hình |  |
| config | Provider | 提供商 | Nhà cung cấp | Dùng cho ASR/LLM/TTS/VAD provider. |
| config | Enable | 启用 / 开启 | Bật | Button/switch. |
| config | Disable | 禁用 / 关闭 | Tắt | Button/switch. |
| config | Save settings | 保存设置 | Lưu cài đặt |  |
| config | Test configuration | 测试配置 | Kiểm tra cấu hình |  |
| config | Service address | 服务地址 | Địa chỉ dịch vụ |  |
| config | API Key | API Key | API Key | Không dịch. |
| config | Access token | 访问令牌 | Access token | Có thể dùng “mã truy cập” trong mô tả, giữ token trong kỹ thuật. |
| VAD | Voice Activity Detection | 语音活动检测 | VAD | Lần đầu trong docs: “VAD (phát hiện hoạt động giọng nói)”. |
| VAD | Silence detection | 静音检测 | Phát hiện khoảng lặng |  |
| VAD | Voice duration | 语音时长 | Thời lượng giọng nói |  |
| VAD | Idle duration | 空闲时长 | Thời gian chờ |  |
| VAD | Threshold | 阈值 | Ngưỡng |  |
| ASR | Automatic Speech Recognition | 语音识别 | ASR | Lần đầu trong docs: “ASR (nhận dạng giọng nói)”. |
| ASR | Transcription | 转写 / 识别文本 | Văn bản nhận dạng |  |
| ASR | Streaming recognition | 流式识别 | Nhận dạng dạng stream |  |
| ASR | Final result | 最终结果 | Kết quả cuối |  |
| ASR | Partial result | 中间结果 / 临时结果 | Kết quả tạm thời |  |
| LLM | Large Language Model | 大语言模型 | LLM | Lần đầu trong docs: “LLM (mô hình ngôn ngữ lớn)”. |
| LLM | Model | 模型 | Mô hình |  |
| LLM | Chat completion | 对话补全 | Hoàn tất hội thoại | Nếu xuất hiện trong API docs. |
| LLM | System prompt | 系统提示词 | System prompt | Có thể chú thích “prompt hệ thống”. |
| LLM | Tool call | 工具调用 | Gọi công cụ |  |
| TTS | Text-to-Speech | 语音合成 | TTS | Lần đầu trong docs: “TTS (tổng hợp giọng nói)”. |
| TTS | Voice | 音色 | Giọng đọc | Nếu là preset voice, dùng “giọng”. |
| TTS | Voice option | 音色选项 | Tùy chọn giọng |  |
| TTS | Speech synthesis | 语音合成 | Tổng hợp giọng nói |  |
| TTS | Preview voice | 试听音色 | Nghe thử giọng |  |
| MQTT | MQTT | MQTT | MQTT | Không dịch. |
| MQTT | Broker | Broker | Broker MQTT | Giữ “broker”. |
| MQTT | Topic | 主题 / Topic | Topic | Không dịch topic value. |
| MQTT | Publish | 发布 | Publish | Trong UI có thể dùng “Gửi publish”. |
| MQTT | Subscribe | 订阅 | Subscribe | Trong UI có thể dùng “Đăng ký subscribe”. |
| MQTT | Client ID | 客户端ID | Client ID | Không dịch field/key. |
| UDP | UDP | UDP | UDP | Không dịch. |
| UDP | UDP server | UDP服务器 | UDP Server |  |
| UDP | Listen address | 监听地址 | Địa chỉ lắng nghe |  |
| UDP | Port | 端口 | Cổng |  |
| UDP | Packet | 数据包 | Gói tin |  |
| MCP | Model Context Protocol | 模型上下文协议 | MCP | Lần đầu trong docs có thể ghi đầy đủ. |
| MCP | MCP service | MCP服务 | Dịch vụ MCP |  |
| MCP | MCP server | MCP服务器 | MCP Server |  |
| MCP | MCP tool | MCP工具 | Công cụ MCP |  |
| MCP | Tool list | 工具列表 | Danh sách công cụ |  |
| MCP | Market | 市场 | Chợ MCP | Có thể dùng “MCP Market” nếu là tên tính năng. |
| OpenClaw | OpenClaw | OpenClaw / 龙虾 | OpenClaw | Giữ nguyên tên riêng. |
| OpenClaw | Enter keyword | 进入关键词 | Từ khóa vào OpenClaw | Không dịch keyword value nếu là trigger cần tương thích, trừ khi đổi chủ đích. |
| OpenClaw | Exit keyword | 退出关键词 | Từ khóa thoát OpenClaw |  |
| OpenClaw | Warmup assistant | 暖场助手 | Trợ lý chờ phản hồi | Dùng cho prompt warmup. |
| OTA | Over-the-Air update | OTA升级 / 固件升级 | OTA | Lần đầu trong docs: “OTA (cập nhật từ xa)”. |
| OTA | Firmware | 固件 | Firmware | Có thể dùng “firmware” thay “phần mềm thiết bị”. |
| OTA | Version | 版本 | Phiên bản |  |
| OTA | Update URL | 升级地址 | URL cập nhật | Không dịch URL/key. |
| OTA | Signature key | 签名密钥 | Khóa ký |  |
| voice clone | Voice clone | 声音复刻 / 音色复刻 | Nhân bản giọng nói | Dùng cho tên tính năng. |
| voice clone | Clone voice | 复刻音色 | Nhân bản giọng | Hành động. |
| voice clone | Voice clone task | 复刻任务 | Tác vụ nhân bản giọng |  |
| voice clone | Quota | 额度 | Hạn mức |  |
| voice clone | Sample audio | 样本音频 | Âm thanh mẫu |  |
| speaker identification | Speaker identification | 声纹识别 | Nhận diện người nói | Dễ hiểu hơn “nhận diện vân giọng”. |
| speaker identification | Speaker | 说话人 / 声纹 | Người nói | Nếu là cấu hình giọng, tránh nhầm với loa. |
| speaker identification | Speaker group | 声纹组 | Nhóm người nói |  |
| speaker identification | Enrollment | 注册 / 录入 | Ghi danh giọng nói | Dùng trong docs, UI có thể dùng “Thêm mẫu giọng”. |
| speaker identification | Similarity score | 相似度 | Điểm tương đồng |  |
| knowledge base | Knowledge base | 知识库 | Kho tri thức |  |
| knowledge base | Document | 文档 | Tài liệu |  |
| knowledge base | Retrieval | 检索 | Truy xuất | Trong RAG context. |
| knowledge base | Recall test | 召回测试 | Kiểm tra truy xuất |  |
| knowledge base | Embedding | 向量化 / 嵌入 | Embedding | Giữ “embedding” trong tài liệu kỹ thuật. |
| knowledge base | RAG | RAG | RAG | Không dịch. |
| chat history | Chat history | 聊天历史 | Lịch sử trò chuyện |  |
| chat history | Message | 消息 | Tin nhắn |  |
| chat history | User message | 用户消息 | Tin nhắn người dùng |  |
| chat history | Assistant message | 助手消息 | Tin nhắn trợ lý |  |
| chat history | Conversation | 对话 | Cuộc trò chuyện |  |
| chat history | Audio record | 音频记录 | Bản ghi âm |  |
| chat history | Push message | 消息推送 / 语音推送 | Đẩy tin nhắn / Đẩy giọng nói | Chọn theo ngữ cảnh. |

## Quy ước dịch thường gặp

| Nguồn | Nên dịch là | Không nên dịch thành |
|---|---|---|
| 小智管理系统 | Hệ thống quản lý Xiaozhi | Hệ thống quản lý Tiểu Trí |
| 控制台 | Bảng điều khiển | Console nếu là UI phổ thông |
| 仪表板 | Dashboard | Bảng đồng hồ |
| 首页 | Trang chủ | Nhà |
| 刷新 | Làm mới | Refresh nếu button thuần Việt hóa |
| 添加 | Thêm | Gia tăng |
| 创建 | Tạo | Sáng tạo |
| 编辑 | Chỉnh sửa | Biên tập |
| 删除 | Xóa | Loại bỏ |
| 保存 | Lưu | Bảo tồn |
| 取消 | Hủy | Hủy bỏ nếu nút ngắn cần gọn |
| 确认 | Xác nhận | Đúng |
| 搜索 | Tìm kiếm | Lục soát |
| 请选择 | Vui lòng chọn | Hãy tuyển chọn |
| 请输入 | Vui lòng nhập | Hãy đưa vào |
| 加载中 | Đang tải | Trong lúc tải |
| 请求失败 | Yêu cầu thất bại | Request thất bại |
| 认证失败 | Xác thực thất bại | Chứng nhận thất bại |
| 未认证 | Chưa xác thực | Chưa chứng nhận |
| 无权限 | Không có quyền | Không quyền hạn |
| 数据库不可用 | Cơ sở dữ liệu không khả dụng | Database không thể dùng |
| 配置名称 | Tên cấu hình | Tên config nếu UI thuần Việt |
| 服务地址 | Địa chỉ dịch vụ | Địa chỉ phục vụ |
| 测试全部 | Kiểm tra tất cả | Test toàn bộ |

## Checklist khi Việt hóa

1. Xác định chuỗi là user-facing hay developer-only.
2. Nếu là API/config/identifier/key, giữ nguyên.
3. Nếu là label/message/docs/comment cấu hình, dịch theo glossary này.
4. Với prompt/default keyword, kiểm tra tác động hành vi trước khi đổi nghĩa.
5. Sau khi dịch frontend, kiểm tra UI desktop và mobile để tránh vỡ layout.
6. Sau khi dịch backend error, kiểm tra frontend vẫn hiển thị lỗi đúng ngữ cảnh.
