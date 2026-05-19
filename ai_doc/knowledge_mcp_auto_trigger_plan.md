# Phương án tự kích hoạt truy xuất kho tri thức không gây gián đoạn (chat thiết bị, v2)

## Bối cảnh

- Công cụ truy xuất hiện tại `search_knowledge` chủ yếu phụ thuộc vào việc model có chủ động gọi hay không.
- Cần hỗ trợ “truy xuất định hướng theo ID kho tri thức đã hit”, giảm request tới các kho tri thức không liên quan.

## Thay đổi cốt lõi

1. Nâng cấp tham số đầu vào của tool
- Thêm trường tùy chọn `knowledge_base_ids: number[]` cho `search_knowledge`.
- Giữ lại `query`, `top_k`.
- Logic tương thích: khi không truyền `knowledge_base_ids`, truy xuất theo toàn bộ kho tri thức khả dụng của trợ lý hiện tại.

2. Ngữ nghĩa truy xuất định hướng
- Khi truyền `knowledge_base_ids`, chỉ truy xuất trong các kho tri thức đó.
- ID không hợp lệ (chưa liên kết/không tồn tại/inactive/thiếu `external_kb_id`) sẽ tự động bị bỏ qua theo best effort.

3. Chiến lược thực thi song song
- Gửi request song song theo “chiều kho tri thức”, mỗi kho tri thức hit sẽ khởi tạo request truy xuất riêng.
- Provider vẫn do cấu hình riêng của kho tri thức quyết định (`dify`/`ragflow`).
- Sau khi gom toàn bộ kết quả hit, sort toàn cục theo score rồi cắt theo `top_k`.

4. Chiến lược timeout (đã xác nhận mặc định)
- Timeout từng kho: `2500ms`
- Timeout tổng: `2500ms`
- Timeout/thất bại một phần không chặn luồng chính; nếu toàn bộ thất bại thì trả lỗi.

5. Nâng cấp gợi ý route cho LLM
- Prompt hệ thống gửi kèm danh sách “ID:tên kho tri thức khả dụng”.
- Hướng dẫn model truyền `knowledge_base_ids` khi có thể xác định; nếu không chắc thì có thể không truyền.

## Bước triển khai

1. Thêm `knowledge_base_ids` vào cấu trúc tham số của `search_knowledge`.
2. Truyền xuyên suốt `knowledge_base_ids` trong chuỗi gọi `ChatSessionOperator -> LocalMcpSearchKnowledge -> rag.Search`.
3. Thêm lọc theo ID và kiểm soát timeout tổng trong `rag.Search`.
4. Sửa `dify_searcher` và `ragflow_searcher` để truy xuất song song theo kho tri thức, đồng thời thêm timeout từng kho.
5. Điều chỉnh quy tắc truy xuất kho tri thức trong prompt hệ thống để hỗ trợ hướng dẫn `knowledge_base_ids`.

## Tương thích và fallback

- Các lệnh gọi cũ không truyền `knowledge_base_ids` không bị ảnh hưởng.
- Khi một provider thất bại cục bộ, chỉ ghi log và bỏ qua, vẫn giữ kết quả từ provider khác.

## Tiêu chí nghiệm thu

- Tool có thể nhận và áp dụng `knowledge_base_ids`.
- Trong kịch bản nhiều kho tri thức, có thể truy xuất song song và trả kết quả tổng hợp.
- Timeout từng kho và timeout tổng đều mặc định là 2500ms.
- Đường gọi cũ, không truyền `knowledge_base_ids`, vẫn hoạt động.
