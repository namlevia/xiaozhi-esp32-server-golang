# Hướng dẫn gọi từ xa MCP theo chiều thiết bị/agent

Tài liệu này giới thiệu **năng lực debug gọi từ xa MCP** trong console quản trị, bao gồm:

- Tạo MCP endpoint theo chiều agent
- Lấy danh sách công cụ và gọi từ xa theo chiều agent
- Lấy danh sách công cụ và gọi từ xa theo chiều thiết bị
- Khác biệt quyền giữa quản trị viên và người dùng thường

Tài liệu liên quan:

- [Mô tả kiến trúc MCP](./mcp.md)
- [Mô tả chức năng MCP Market](./mcp_market.md)
- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)

---

## 1. Định vị chức năng

Chức năng này chủ yếu dùng để “debug và xác minh”:

- Xem nhanh agent/thiết bị hiện tại expose những MCP tool nào
- Trực tiếp dựng tham số và gọi tool trong console
- Lấy MCP endpoint theo chiều agent để client MCP bên ngoài kết nối test

Kịch bản phù hợp:

- Xác minh một MCP service từ xa đã có hiệu lực hay chưa
- Kiểm tra tool schema / tham số mẫu
- Xử lý khác biệt hành vi MCP giữa agent và thiết bị

---

## 2. Khác biệt giữa hai chiều gọi

## 2.1 Chiều agent (Agent)

Đặc điểm:

- Nhìn từ góc “cấu hình agent”
- Hỗ trợ lấy MCP endpoint của agent đó (kèm token)
- Hỗ trợ kéo danh sách tool và gọi tool trực tiếp
- Chịu ảnh hưởng bởi cấu hình agent (như `mcp_service_names`)

Mục đích thường gặp:

- Xác minh tập MCP tool khả dụng sau khi agent lọc
- Copy endpoint cho client debug bên ngoài sử dụng

## 2.2 Chiều thiết bị (Device)

Đặc điểm:

- Nhìn từ góc “kết nối thiết bị cụ thể”
- Trực tiếp request chi tiết tool/gọi tool thông qua context kết nối hiện tại của thiết bị
- Thường phụ thuộc vào thiết bị online và WebSocket controller khả dụng

Mục đích thường gặp:

- Xử lý tình huống “cùng một agent nhưng tool biểu hiện khác nhau trên các thiết bị khác nhau”
- Xác minh năng lực MCP phía session online hiện tại của thiết bị

---

## 3. Lối vào trang (quản trị viên / người dùng thường)

### 3.1 Quản trị viên

- `Quản trị viên -> Quản lý agent` (endpoint / tools / call theo chiều agent)
- `Quản trị viên -> Quản lý thiết bị` (tools / call theo chiều thiết bị)

### 3.2 Người dùng thường

- `Agent của tôi` (tools / call theo chiều agent)
- `Thiết bị của tôi` / `Thiết bị agent` (tools / call theo chiều thiết bị)
- `Chỉnh sửa agent` (cấu hình `mcp_service_names`, ảnh hưởng phạm vi service thấy được theo chiều agent)

---

## 4. Chiều agent: luồng debug đầy đủ

## 4.1 Cấu hình MCP service khả dụng cho agent (tùy chọn nhưng khuyến nghị)

Trong trang chỉnh sửa agent, có thể đặt `mcp_service_names` (danh sách tên service, phân tách bằng dấu phẩy):

- Để trống: dùng toàn bộ MCP service toàn cục đã bật
- Có điền: chỉ dùng các service được chỉ định (phải là service tồn tại và đã bật trong hệ thống)

Hệ thống sẽ xử lý trường này bằng cách:

- Loại trùng
- Bỏ khoảng trắng
- Kiểm tra hợp lệ (tên service phải tồn tại trong tập service toàn cục đã bật)

## 4.2 Lấy MCP Endpoint của agent

Console có thể lấy URL MCP endpoint riêng cho agent, định dạng tương tự:

```text
ws(s)://<host>/mcp?token=<jwt>
```

Mô tả:

- Endpoint suy ra domain và protocol dựa trên `external.websocket.url` trong cấu hình OTA mặc định
- Token chứa context người dùng hiện tại và agent (dùng cho kiểm tra quyền/gắn kết)
- Phù hợp để client MCP bên ngoài debug tạm thời, không khuyến nghị chia sẻ công khai

## 4.3 Lấy danh sách tool

Console sẽ request chi tiết MCP tool theo chiều agent, nội dung trả về thường gồm:

- `name`
- Mô tả tool
- Tham số schema
- Tham số mẫu (nếu phía thiết bị/server cung cấp)

Nếu không lấy được (ví dụ controller chưa khởi tạo hoặc client tạm thời không truy cập được), backend sẽ trả về danh sách rỗng thay vì báo lỗi, để trang tiếp tục thao tác được.

## 4.4 Gọi tool trực tiếp

Điền trong console:

- `tool_name`
- `arguments` (JSON)

Sau khi gọi, có thể xem body trả về đầy đủ trong khung kết quả (định dạng JSON).

---

## 5. Chiều thiết bị: luồng debug đầy đủ

## 5.1 Lấy danh sách tool của thiết bị

Sau khi chọn thiết bị, console sẽ dùng định danh thiết bị (nội bộ sẽ map sang tên thiết bị) để request chi tiết MCP tool từ WebSocket controller.

Tình huống thất bại thường gặp:

- Thiết bị không online
- Thiết bị không thuộc người dùng hiện tại (góc nhìn người dùng)
- WebSocket controller tạm thời không khả dụng

Trong các tình huống này, interface thường trả về danh sách tool rỗng hoặc lỗi quyền.

## 5.2 Gọi MCP tool của thiết bị

Tương tự chiều agent, điền:

- `tool_name`
- `arguments` (JSON)

Khác biệt là body gọi dùng context `device_id` (backend thực tế sẽ truyền tên thiết bị), nên gần với môi trường thực thi “session thiết bị hiện tại” hơn.

---

## 6. Khác biệt quyền và interface (quản trị viên vs người dùng thường)

### 6.1 Interface người dùng thường

Chiều agent:

- `GET /user/agents/:id/mcp-endpoint`
- `GET /user/agents/:id/mcp-tools`
- `POST /user/agents/:id/mcp-call`

Chiều thiết bị:

- `GET /user/devices/:id/mcp-tools`
- `POST /user/devices/:id/mcp-call`

Hỗ trợ lọc service của agent:

- `GET /user/agents/:id/mcp-services/options`

Người dùng thường chỉ có thể thao tác agent/thiết bị thuộc về mình.

### 6.2 Interface quản trị viên

Chiều agent:

- `GET /admin/agents/:id/mcp-endpoint`
- `GET /admin/agents/:id/mcp-tools`
- `POST /admin/agents/:id/mcp-call`

Chiều thiết bị:

- `GET /admin/devices/:id/mcp-tools`
- `POST /admin/devices/:id/mcp-call`

Quản trị viên có thể debug agent/thiết bị xuyên người dùng (miễn là bản ghi tồn tại và đường kết nối bình thường).

---

## 7. Logic tạo Endpoint (chiều agent)

Endpoint agent phụ thuộc vào:

1. Cấu hình OTA mặc định (`type=ota` và `is_default=true`)
2. `external.websocket.url` trong cấu hình OTA
3. Token ổn định được tạo dựa trên ID người dùng hiện tại + ID agent

Kết quả tạo sẽ dùng:

- Cùng protocol (`ws` / `wss`)
- Cùng host (domain/IP + cổng)
- Path cố định `/mcp`

Vì vậy nếu không tạo được endpoint, hãy ưu tiên kiểm tra cấu hình WebSocket public của OTA.

---

## 8. Câu hỏi thường gặp và xử lý sự cố

### 8.1 Danh sách tool rỗng

Nguyên nhân có thể:

- Thiết bị không online (chiều thiết bị)
- WebSocket controller chưa khởi tạo
- Client chưa trả về chi tiết tool
- Chiều agent không còn service khả dụng sau khi bị `mcp_service_names` lọc

Thứ tự kiểm tra khuyến nghị:

1. Xác nhận trạng thái online của thiết bị
2. Kiểm tra MCP service toàn cục đã bật chưa
3. Kiểm tra cấu hình `mcp_service_names` của agent
4. Thử lấy lại tool trong console

### 8.2 Khi gọi báo lỗi tham số JSON

Khu vực tham số trong console yêu cầu JSON object hợp lệ, ví dụ:

```json
{
  "query": "hello"
}
```

Lỗi thường gặp:

- Dùng dấu nháy đơn
- Có dấu phẩy thừa ở cuối
- Tầng trên cùng không phải object

### 8.3 Lấy endpoint agent thất bại

Thường do thiếu cấu hình OTA mặc định hoặc chưa cấu hình `external.websocket.url`.

### 8.4 Đã import MCP service nhưng agent gọi không thấy

Kiểm tra:

1. Service import đã bật chưa
2. Công tắc tổng cấu hình MCP toàn cục và trạng thái bật của service
3. Agent có loại trừ service này qua `mcp_service_names` hay không

---

## 9. Best practice

- Trước tiên xác minh tool “chiều thiết bị” dùng được ở phía quản trị viên, sau đó mới xác minh kết quả lọc tool “chiều agent”
- Với agent production, khuyến nghị cấu hình rõ `mcp_service_names` để tránh expose tool không liên quan cho model
- Xem endpoint là lối vào debug nhạy cảm, tránh phát tán URL có token trên kênh công khai
