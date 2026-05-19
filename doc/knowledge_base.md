# Hướng dẫn chức năng kho tri thức

Tài liệu này giới thiệu chức năng **kho tri thức (Knowledge Base / RAG)** trong dự án, bao gồm cấu hình provider phía quản trị viên, quản lý kho tri thức và tài liệu phía người dùng thường, kiểm thử truy hồi, và tích hợp truy hồi kho tri thức trong luồng chat của chương trình chính.

Tài liệu liên quan:

- [Hướng dẫn sử dụng trang quản trị](./manager_console_guide.md)
- [Mô tả kiến trúc MCP](./mcp.md) (công cụ truy hồi kho tri thức `search_knowledge` sẽ được kích hoạt qua luồng công cụ cục bộ)

---

## 1. Tổng quan chức năng

Chức năng kho tri thức dùng để cung cấp năng lực “trả lời dựa trên tài liệu” cho agent, gồm ba tầng:

1. Quản trị viên cấu hình provider truy hồi kho tri thức (Dify / RAGFlow / WeKnora)
2. Người dùng thường tạo kho tri thức và tài liệu, sau đó đồng bộ bất đồng bộ lên provider
3. Agent liên kết kho tri thức và kích hoạt công cụ cục bộ `search_knowledge` trong hội thoại để hoàn tất truy hồi

Provider hiện đã được trang quản trị frontend hỗ trợ:

- `dify`
- `ragflow`
- `weknora`

---

## 2. Phân công vai trò

## 2.1 Quản trị viên

Chịu trách nhiệm:

- Cấu hình provider truy hồi kho tri thức (toàn cục)
- Duy trì tham số kết nối provider và ngưỡng mặc định
- (Tùy chọn) quản lý kho tri thức thay người dùng

Lối vào:

- `Quản trị viên -> Cấu hình truy hồi kho tri thức`

## 2.2 Người dùng thường

Chịu trách nhiệm:

- Tạo/sửa/xóa kho tri thức của mình
- Quản lý tài liệu trong kho tri thức (nhập văn bản / tải file lên)
- Chủ động đồng bộ và thử lại
- Dùng “kiểm thử truy hồi” để xác minh hiệu quả khớp từ khóa
- Chọn kho tri thức liên kết trong agent

Lối vào:

- `Người dùng thường -> Kho tri thức của tôi`
- `Người dùng thường -> Chỉnh sửa agent (liên kết kho tri thức)`

---

## 3. Quản trị viên: cấu hình truy hồi kho tri thức (cấu hình Provider)

Trang quản trị hỗ trợ duy trì nhiều cấu hình provider và chỉ định provider mặc định.

Mục cấu hình thường gặp (khác nhau tùy provider):

- `Base URL`
- `API Key / Token`
- Ngưỡng truy hồi mặc định
- Tham số riêng của provider (như ngưỡng tương đồng RAGFlow, tham số phân đoạn WeKnora, v.v.)

### 3.1 Dify

Mục cấu hình điển hình:

- `base_url`
- `api_key`
- `score_threshold`
- Tham số provider khác

### 3.2 RAGFlow

Mục cấu hình điển hình:

- `base_url`
- `api_key`
- `similarity_threshold`

### 3.3 WeKnora

Mục cấu hình điển hình:

- `base_url`
- `api_key`
- `score_threshold`
- Tham số phân đoạn (`chunk_size` / `chunk_overlap` / `separators`)
- Tham số polling parse (`parse_poll_interval_ms` / `parse_timeout_ms`)

Trang quản trị còn hỗ trợ lấy danh sách model WeKnora (embedding / llm / rerank) để hỗ trợ điền cấu hình.

---

## 4. Người dùng thường: kho tri thức của tôi (quản lý KB)

Lối vào:

- `Người dùng thường -> Kho tri thức của tôi`

Thao tác hỗ trợ:

- Thêm/sửa kho tri thức
- Đặt trạng thái (`active` / `inactive`)
- Đặt ngưỡng truy hồi (có thể kế thừa toàn cục)
- Quản lý tài liệu
- Thử lại đồng bộ thủ công
- Kiểm thử truy hồi
- Xóa kho tri thức

### 4.1 Trường kho tri thức (người dùng thấy được)

Các cột hiển thị thường gặp:

- ID
- Tên
- Mô tả
- Provider
- Trạng thái
- Trạng thái đồng bộ
- Thời gian đồng bộ gần nhất
- Thao tác

Ghi chú:

- Khi đồng bộ thất bại, thông tin lỗi sẽ hiển thị dưới dạng tooltip trong cột “trạng thái đồng bộ” để tránh bảng quá rộng theo chiều ngang.

### 4.2 Trạng thái đồng bộ (thường gặp)

Kho tri thức và tài liệu đều có thể xuất hiện các trạng thái tương tự:

- Chờ đồng bộ
- Đang tải lên / đã tải lên / đang parse
- Đã đồng bộ
- Thất bại (bao gồm tải lên thất bại, parse thất bại, v.v.)

Nếu thất bại, có thể nhấp `thử lại đồng bộ` để đưa lại vào hàng đợi tác vụ bất đồng bộ.

---

## 5. Quản lý tài liệu (trong kho tri thức)

Mỗi kho tri thức có thể chứa nhiều tài liệu, hỗ trợ:

- Tài liệu dạng văn bản (chỉnh sửa online)
- Tạo tài liệu bằng cách tải file lên (theo giới hạn định dạng của provider)

Chức năng trang:

- Thêm tài liệu
- Sửa tài liệu (tài liệu dạng file thường không hỗ trợ sửa online)
- Xóa tài liệu
- Thử lại đồng bộ
- Tải file lên

### 5.1 Định dạng file tải lên

Frontend sẽ hiển thị gợi ý `accept` và mô tả tải lên khác nhau theo provider hiện tại của kho tri thức:

- Dify: hỗ trợ các định dạng văn bản/tài liệu thường gặp (như txt/md/pdf/html/xlsx/docx/csv, v.v.)
- RAGFlow: hỗ trợ phạm vi file rộng hơn (bao gồm ảnh, log, file cấu hình, v.v.)
- WeKnora: hỗ trợ phạm vi file rộng (bao gồm Office, ảnh, email, v.v.)

Định dạng tải lên cụ thể hãy lấy theo gợi ý trên trang.

---

## 6. Kiểm thử truy hồi (phía người dùng)

Trong danh sách kho tri thức, có thể thực hiện `kiểm thử truy hồi` cho từng kho tri thức để trực tiếp xác minh hiệu quả truy hồi của provider.

Mục kiểm thử:

- `query`: từ khóa hoặc câu hỏi kiểm thử
- `top_k`
- `threshold` (chỉ có hiệu lực cho lần kiểm thử này, có thể để trống)

Nội dung trả về:

- Số lượng kết quả khớp
- Nguồn kết quả khớp (title)
- score
- Đoạn văn bản khớp
- Thời gian phản hồi

### 6.1 Ưu tiên ngưỡng (mô tả logic)

Thông thường lấy ngưỡng theo thứ tự ưu tiên sau:

1. Ngưỡng trong request kiểm thử lần này (nếu có nhập)
2. Ngưỡng riêng của kho tri thức
3. Ngưỡng mặc định toàn cục của provider

### 6.2 Mô tả tham số WeKnora (quan trọng)

Kiểm thử truy hồi WeKnora hiện đã sử dụng theo chiều kho tri thức:

- `knowledge_base_ids` (danh sách ID kho tri thức)

Dùng để giới hạn chính xác phạm vi truy hồi trong kho tri thức hiện tại.

---

## 7. Agent liên kết kho tri thức

Trong trang chỉnh sửa agent, có thể chọn nhiều kho tri thức cho agent (multi-select).

Mô tả hành vi:

- Hỗ trợ liên kết nhiều kho tri thức
- Khi hội thoại, hệ thống sẽ dựa vào model để quyết định có kích hoạt truy hồi kho tri thức hay không
- Nếu có thể xác định kho tri thức cụ thể, lệnh gọi công cụ sẽ truyền `knowledge_base_ids`
- Khi truy hồi thất bại, hệ thống sẽ fallback về hội thoại LLM thông thường (frontend có nội dung gợi ý)

---

## 8. Truy hồi kho tri thức trong luồng hội thoại chương trình chính

Chương trình chính dùng công cụ cục bộ `search_knowledge` để thực hiện truy hồi kho tri thức.

Trường cốt lõi trong tham số gọi công cụ:

- `query`
- `top_k`
- `knowledge_base_ids` (tùy chọn, danh sách ID kho tri thức)

Mô tả hành vi:

- Không truyền `knowledge_base_ids`: truy hồi trong toàn bộ kho tri thức khả dụng đã liên kết với agent hiện tại
- Có truyền `knowledge_base_ids`: chỉ truy hồi trong các kho tri thức được chỉ định

Điều này giúp model thu hẹp phạm vi truy hồi khi đã biết câu hỏi thuộc về nguồn nào, cải thiện độ liên quan và giảm truy hồi nhiễu.

### 8.1 Tham số truy hồi WeKnora trong chương trình chính

Request truy hồi WeKnora của chương trình chính hiện đã dùng:

- `knowledge_base_ids`

Giữ nhất quán với kiểm thử truy hồi trong console.

---

## 9. Danh sách interface (phía người dùng)

### 9.1 CRUD kho tri thức

- `GET /user/knowledge-bases`
- `POST /user/knowledge-bases`
- `GET /user/knowledge-bases/:id`
- `PUT /user/knowledge-bases/:id`
- `DELETE /user/knowledge-bases/:id`
- `POST /user/knowledge-bases/:id/sync`

### 9.2 Kiểm thử truy hồi

- `POST /user/knowledge-bases/:id/test-search`

### 9.3 Quản lý tài liệu

- `GET /user/knowledge-bases/:id/documents`
- `POST /user/knowledge-bases/:id/documents`
- `POST /user/knowledge-bases/:id/documents/upload`
- `PUT /user/knowledge-bases/:id/documents/:doc_id`
- `DELETE /user/knowledge-bases/:id/documents/:doc_id`
- `POST /user/knowledge-bases/:id/documents/:doc_id/sync`

### 9.4 Agent liên kết kho tri thức

- `GET /user/agents/:id/knowledge-bases`
- `PUT /user/agents/:id/knowledge-bases`

---

## 10. Danh sách interface (phía quản trị viên)

### 10.1 Quản lý cấu hình provider

- `GET /admin/knowledge-search-configs`
- `POST /admin/knowledge-search-configs`
- `PUT /admin/knowledge-search-configs/:id`
- `DELETE /admin/knowledge-search-configs/:id`

### 10.2 Lấy model WeKnora (hỗ trợ cấu hình)

- `POST /admin/knowledge-search-configs/weknora/models`

### 10.3 Quản trị viên quản lý kho tri thức thay người dùng (theo chiều người dùng)

- `GET /admin/users/:id/knowledge-bases`
- `POST /admin/users/:id/knowledge-bases`
- `PUT /admin/users/:id/knowledge-bases/:kb_id`
- `DELETE /admin/users/:id/knowledge-bases/:kb_id`

---

## 11. Câu hỏi thường gặp và xử lý sự cố

### 11.1 Tạo kho tri thức xong nhưng luôn không có kết quả khớp

Ưu tiên kiểm tra:

1. Kho tri thức/tài liệu đã đồng bộ thành công chưa
2. Provider bên ngoài đã hoàn tất xây dựng chỉ mục chưa
3. Ngưỡng truy hồi có quá cao không
4. `query` có quá rộng hoặc lệch khỏi nội dung tài liệu không

### 11.2 Không thể sửa tài liệu sau khi tải file lên

Tài liệu được tạo bằng cách tải file lên thường được xử lý như “tài liệu dạng file”, frontend sẽ hạn chế chỉnh sửa online; khuyến nghị xóa rồi tải lại.

### 11.3 Phạm vi truy hồi WeKnora không đúng

Xác nhận:

- Kiểm thử truy hồi trong console có dùng kho tri thức hiện tại để phát động test không
- Lệnh gọi công cụ của agent có truyền đúng `knowledge_base_ids` không

---

## 12. Gợi ý sử dụng

- Tách nhiều kho tri thức theo từng miền nghiệp vụ khác nhau (như hậu mãi, sản phẩm, hợp đồng)
- Dùng “kiểm thử truy hồi” để tinh chỉnh ngưỡng trước, sau đó mới gắn vào agent
- Nêu rõ trong mô tả agent khi nào cần dùng kho tri thức để trả lời, giúp cải thiện chất lượng kích hoạt
