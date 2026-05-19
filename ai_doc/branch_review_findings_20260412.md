# Ghi chú review code nhánh (2026-04-12)

Nhánh: `codex/optimize-tool-invocation-for-concurrency`

## 1. [P1] Phát media thất bại nhưng vẫn bị đánh dấu là có media output

- Vị trí:
  - `internal/app/server/chat/tool.go:257`
  - `internal/app/server/chat/tool.go:271`
  - `internal/app/server/chat/tool.go:275`
  - `internal/app/server/chat/llm.go:873`
- Mô tả vấn đề:
  - Khi `handleAudioContent` / `handleResourceLink` gọi thất bại, logic hiện tại vẫn set:
    - `execResult.hasMediaOutput = true`
    - `execResult.shouldStopLLMProcessing = true`
  - Tầng trên vì vậy cho rằng “media đã được xuất”, rồi đi theo nhánh chặn `tts_stop`/không tiếp tục LLM.
- Ảnh hưởng rủi ro:
  - Thực tế media chưa phát thành công, nhưng luồng hội thoại lại kết thúc như “đã xuất media thành công”, có thể gây im lặng ở client, trạng thái không nhất quán hoặc không có phản hồi tiếp theo.

## 2. [P2] Chống trùng ToolCall ID rỗng có thể loại nhầm lệnh gọi lặp hợp lệ

- Vị trí:
  - `internal/app/server/chat/tool.go:154`
  - `internal/app/server/chat/tool.go:160`
  - `internal/app/server/chat/tool.go:44`
  - `internal/app/server/chat/tool.go:67`
- Mô tả vấn đề:
  - Hiện tại với `ToolCall.ID` rỗng, hệ thống tạo định danh bằng `auto_<name>_<arguments>` rồi dùng để chống trùng.
  - Nếu model hợp lệ tạo hai lệnh gọi “không có ID và tham số giống nhau”, lệnh gọi thứ hai sẽ bị bỏ qua.
- Ảnh hưởng rủi ro:
  - Có thể khiến số lượng/quan hệ tương ứng giữa `tool_calls` trong assistant và `tool_result` sau đó không nhất quán, ảnh hưởng context các lượt sau và độ tin cậy của tool call.
