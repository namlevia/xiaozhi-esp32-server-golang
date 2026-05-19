# Phương án bổ sung cờ ngắt LLM và tăng cường nội dung lịch sử (chờ xác nhận)

## 1. Mục tiêu

Triển khai hai việc trong dự án hiện tại:

1. Khi quá trình xử lý stream của LLM bị ngắt giữa chừng, ghi cờ ngắt vào `Extra` của tin nhắn lịch sử assistant tương ứng.  
2. Khi lắp ráp lịch sử cho yêu cầu LLM sau đó, nếu phát hiện cờ này thì thêm `" [người dùng đã ngắt]"` vào cuối `content` của tin nhắn đó trước khi gửi cho model.

Ghi chú: tài liệu này chỉ mô tả hướng triển khai, chưa trực tiếp sửa code.

---

## 2. Đường dẫn code hiện tại (điểm chính)

- Kích hoạt ngắt: `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/common.go:3`  
  `StopSpeaking()` sẽ hủy `SessionCtx`, khiến context xử lý LLM/TTS kết thúc.

- Xử lý stream LLM và ghi lịch sử: `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go:323`  
  Hiện tại `handleLLMResponse()` chỉ lưu tin nhắn assistant khi `llmResponse.IsEnd=true`;  
  nhánh `ctx.Done()` return trực tiếp, không lưu phần assistant “đã xuất ra nhưng bị ngắt”.

- Điểm lắp ráp lịch sử: `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go:1050`  
  `GetMessages()` hiện append trực tiếp `msg` trong lịch sử vào request, chưa tăng cường nội dung dựa trên `Extra`.

---

## 3. Nguyên tắc thiết kế

1. **Can thiệp tối thiểu**: chỉ sửa logic lưu lịch sử và lắp ráp lịch sử trong `llm.go`.  
2. **Không làm bẩn lịch sử gốc**: khi lắp ráp request, copy message rồi mới sửa `content`, không sửa trực tiếp object lịch sử trong bộ nhớ.  
3. **Tránh ghi DB lặp**: lịch sử khi bị ngắt chỉ ghi một lần và loại trừ với đường xử lý `IsEnd` bình thường.  
4. **Tương thích ngược**: lịch sử không có `Extra.interrupt` giữ nguyên hành vi cũ.

---

## 4. Chi tiết phương án

### 4.1 Ghi Extra khi bị ngắt (giai đoạn LLM)

Vị trí sửa: `handleLLMResponse()` tại `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go:324`

Logic mới:

1. Thêm trạng thái cục bộ trong hàm:
   - `assistantSaved bool`: tránh lưu lặp trong cùng một lần xử lý.

2. Tách một helper nội bộ (closure cục bộ trong hàm hoặc private method), chạy khi `ctx.Done()` được kích hoạt:
   - Lấy phần text hiện đã tích lũy từ `fullText.String()`;
   - Nếu text rỗng thì bỏ qua;
   - Tạo `assistantMsg := schema.AssistantMessage(text, nil)`;
   - Thiết lập:
     - `assistantMsg.Extra["interrupt"] = true`
     - `assistantMsg.Extra["interrupt_by"] = "user"`
     - `assistantMsg.Extra["interrupt_stage"] = "llm"`
   - Gọi `AddLlmMessage(ctx, assistantMsg)` để lưu lịch sử.

3. Gọi helper này ở các điểm return trong nhánh `ctx.Done()`, rồi mới return.

Ghi chú:
- Đường xử lý `IsEnd` bình thường giữ nguyên, không thêm cờ interrupt.
- Chỉ lưu khi thật sự có text đã tích lũy, tránh tạo message assistant rỗng.

---

### 4.2 Tăng cường content theo Extra khi lắp ráp lịch sử

Vị trí sửa: `GetMessages()` tại `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go:1050`

Logic mới:

1. Khi duyệt tin nhắn lịch sử, không append trực tiếp `msg` gốc; trước tiên tạo bản shallow copy, copy các trường cần thiết.  
2. Nếu thỏa mãn:
   - `msg.Role == schema.Assistant`
   - `msg.Extra != nil`
   - `msg.Extra["interrupt"] == true`
   - `msg.Content` không rỗng
   
   thì đổi nội dung phía request thành:
   - `newMsg.Content = msg.Content + " [người dùng đã ngắt]"`

3. Để tránh append lặp, nếu nội dung đã kết thúc bằng `" [người dùng đã ngắt]"` thì không append nữa.

Lưu ý:
- Chỉ sửa `content` trong “bản copy dùng để lắp ráp request”, không sửa lịch sử gốc.

---

### 4.3 Lọc user ở cuối lịch sử để tránh làm nhiễu lượt hiện tại

Vị trí sửa: `GetMessages()` tại `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go:1050`

Logic mới:

1. Sau `messageList := l.clientState.GetMessages(count)`, kiểm tra “tin nhắn cuối cùng trong lịch sử”:
   - Nếu tin nhắn cuối có `Role == schema.User`, loại bỏ tin nhắn đó khỏi `messageList`.

2. Chỉ lọc các message `user` liên tiếp ở cuối:
   - Khuyến nghị dùng vòng lặp lùi từ cuối danh sách, xóa các message `user` liên tiếp cho đến khi phần tử cuối không còn là `user` hoặc danh sách rỗng.

Mục đích:
- Tránh text user còn sót từ lượt trước trong lịch sử bị trộn với `userMessage` của lượt hiện tại, làm nhiễu ý định hội thoại hiện tại.

Lưu ý:
- Đây là bước lọc khi lắp ráp request, không sửa dữ liệu lịch sử gốc trong bộ nhớ.

---

## 5. Helper nên bổ sung

Nên đặt trong khu vực private method của `llm.go`:

1. `isInterruptedMessage(msg *schema.Message) bool`  
   Thống nhất cách kiểm tra `Extra.interrupt`, hỗ trợ cả bool và chuỗi `"true"` để chịu lỗi dữ liệu.

2. `decorateInterruptedContent(content string) string`  
   Thống nhất logic append, tránh lặp `" [người dùng đã ngắt]"`.

3. `cloneMessageForRequest(msg *schema.Message) *schema.Message`  
   Copy `Role/Content/Name/ToolCalls/ToolCallID/Extra/ResponseMeta`, tối thiểu đảm bảo có thể sửa `Content` và `Extra` an toàn.

---

## 6. Tương thích và rủi ro

1. `Extra` có tác động trực tiếp tới model hay không:  
   Hiện tầng adapter OpenAI không truyền tiếp `Extra` khi lắp request, nên hành vi model chủ yếu phụ thuộc vào việc ta append `" [người dùng đã ngắt]"` vào `content`.

2. Khác biệt trong lưu trữ lịch sử:  
   - Ở chế độ `redis`, `schema.Message` được lưu JSON trực tiếp, nên `Extra` có thể được giữ lại.  
   - Ở chế độ `manager`, hiện chỉ lưu `role/content/tool_calls`, nên `Extra` có thể bị mất.  
   Vì vậy nếu sau này cần năng lực này ở chế độ manager, cần mở rộng đồng bộ giao thức manager history.

3. Ảnh hưởng của wording:  
   `" [người dùng đã ngắt]"` là prompt rõ ràng, sẽ ảnh hưởng phong cách viết tiếp của model; đây là hành vi mong muốn của yêu cầu này.

---

## 7. Tiêu chí nghiệm thu (triển khai sau khi xác nhận)

1. Kịch bản: user đã vào lịch sử, assistant đang stream thì bị ngắt giữa chừng  
   - Lịch sử có thêm một tin nhắn assistant, `Extra.interrupt=true`.

2. Trước khi gửi LLM ở lượt kế tiếp, kiểm tra request messages  
   - Nội dung assistant tương ứng trở thành `"<đoạn gốc> [người dùng đã ngắt]"`.

3. Tin nhắn hoàn tất bình thường, không bị ngắt  
   - Không có `Extra.interrupt`, `content` không được thêm hậu tố.

4. Khi cuối lịch sử là user, user cuối đó bị lọc khỏi request, không trùng/lẫn với user của lượt hiện tại.

5. Không xuất hiện marker lặp, không xuất hiện bản ghi assistant rỗng.

---

## 8. Danh sách file triển khai (sau khi xác nhận)

- `/Users/shijingbo/git/xiaozhi-esp32-server-golang/internal/app/server/chat/llm.go`
- Tùy chọn: `/Users/shijingbo/git/xiaozhi-esp32-server-golang/test/interrupt_history/main.go` để xác minh demo
