# Tài liệu đối nối API MemOS provider độc lập

> Tài liệu chính thức: `https://memos-docs.openmem.net/cn/api_docs/start/overview`

> Ví dụ `base_url`: `https://memos.memtensor.cn/api/openmem/v1`

> Mục tiêu: tích hợp MemOS như một provider độc lập, không dùng chung client `mem0` nữa.

## 1. Nguyên tắc

- Không đoán API path.
- Không route `memos` sang client `mem0`.
- Chỉ dựa trên tài liệu chính thức do user cung cấp để mapping field và endpoint.

## 2. Ràng buộc cải tạo trong repo hiện tại

Interface hệ thống `MemoryProvider` yêu cầu triển khai:

```go
AddMessage(ctx, agentID, message) error
GetMessages(ctx, agentID, limit) ([]*schema.Message, error)
GetContext(ctx, agentID) (string, error)
Search(ctx, agentID, query, topK, threshold, timeRangeDays) (string, error)
Flush(ctx, agentID) error
ResetMemory(ctx, agentID) error
```

Vì vậy khi đối nối MemOS, cần tìm API tương ứng hoặc tổ hợp API trong tài liệu chính thức và hoàn tất mapping.

## 3. Điểm đối nối theo tài liệu chính thức

Các field sau đi theo path cố định chính thức, không mở cấu hình endpoint trong console:

1. Cách xác thực
   - Tên header:
   - Prefix token, ví dụ `Bearer `:

2. API ghi memory
   - Ví dụ request body:
   - Field quan trọng trong response:

3. API truy vấn memory
   - Tham số lọc như `agent_id`, `user_id`, `session_id`:
   - Ví dụ response body:

4. API search/recall
   - Tham số `query`, `top_k`, `threshold`, `time_range`:
   - Field response gồm text, score, timestamp:

5. API clear/delete
   - Chiều xóa: cấp user, session hoặc agent:

6. Có API flush/index refresh hay không
   - Nếu không có, `Flush` degrade semantic như thế nào:

## 4. Kế hoạch code

```text
internal/domain/memory/memos/
  memos_client.go        # HTTP call thật
  memos_client_test.go   # test request/response mapping
```

Và sửa:

- Cấu hình quản trị giữ `memos` (đã hỗ trợ).
- Config ví dụ giữ `memory.memos` (đã hỗ trợ).

## 5. Ghi chú môi trường

Môi trường hiện tại request tới site tài liệu chính thức trả 403 nên không thể tự động fetch nội dung tài liệu local.

Hiện đã triển khai theo path cố định, console không cho chỉnh endpoint path.

## 6. Ghi chú triển khai hiện tại

- URL request thực tế = `base_url + endpoint_path`, ví dụ `http://host/api/v1` + `/add/message`.
- Đã triển khai `memos_client.go`, mặc định dùng các interface:
  - `POST /add/message`
  - `GET /get/messages`
  - `POST /search/memory`
  - `POST /flush`
  - `POST /reset/memory`
- Path dùng semantic chính thức cố định: `/add/message`, `/get/messages`, `/search/memory`, `/flush`, `/reset/memory`.

## 7. Ràng buộc field Add Message

- `user_id` / `conversation_id` là bắt buộc.
- `agent_id` là tùy chọn, chỉ truyền khi có giá trị.
- Triển khai hiện tại dùng `agentID` map đồng thời tới `user_id` và `conversation_id`; khi `agentID` rỗng thì báo lỗi trực tiếp, không dùng placeholder mặc định.

## 8. Mapping field Search Memory

- `user_id`: map từ `agentID`.
- `conversation_id`: map từ `agentID`.
- `query`: truyền nguyên input người dùng.
- `memory_limit_number`: map từ `topK`.
- `relativity`: map từ `search_threshold`.
