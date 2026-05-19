# Phương án tích hợp kho tri thức WeKnora (chờ xác nhận)

## 1. Mục tiêu và ràng buộc

- Trên nền `dify/ragflow` hiện có, thêm provider thứ ba: `weknora`.
- Quản trị viên có thể cấu hình tham số kết nối WeKnora; CRUD kho tri thức/tài liệu phía người dùng thường vẫn đi qua đồng bộ bất đồng bộ.
- Chuỗi truy xuất RAG cục bộ của chương trình chính hỗ trợ `weknora`, không gọi ngược về API truy xuất của console.
- Giữ nguyên cấu trúc dữ liệu hiện có: không thêm cột DB; đồng bộ tài liệu chỉ dùng `sync_status + sync_error`.
- Tiếp tục dùng chiến lược xóa hiện tại: xóa tài liệu; nếu kho tri thức do hệ thống tự tạo và phía remote rỗng thì xóa kho tri thức.

## 2. Căn cứ API chính thức

- Tổng quan API và xác thực (`X-API-Key`, Base URL `/api/v1`):
  - <https://github.com/Tencent/WeKnora/blob/main/docs/api/README.md>
- Quản lý kho tri thức:
  - <https://github.com/Tencent/WeKnora/blob/main/docs/api/knowledge-base.md>
- Quản lý tri thức (file/URL/tri thức thủ công, trạng thái phân tích):
  - <https://github.com/Tencent/WeKnora/blob/main/docs/api/knowledge.md>
- Tìm kiếm tri thức:
  - <https://github.com/Tencent/WeKnora/blob/main/docs/api/knowledge-search.md>
- Quản lý model (dùng để lấy Embedding model mặc định):
  - <https://github.com/Tencent/WeKnora/blob/main/docs/api/model.md>

## 3. Ánh xạ API (hành động hệ thống -> WeKnora API)

1. Tạo kho tri thức remote (`external_kb_id`)  
`POST /api/v1/knowledge-bases`

2. Cập nhật metadata kho tri thức remote (tên/mô tả/cấu hình chunk)  
`PUT /api/v1/knowledge-bases/:id`

3. Xóa kho tri thức remote  
`DELETE /api/v1/knowledge-bases/:id`

4. Tạo tài liệu (tải file lên)  
`POST /api/v1/knowledge-bases/:id/knowledge/file` (multipart)

5. Truy vấn trạng thái phân tích tài liệu  
`GET /api/v1/knowledge/:id` (dùng `parse_status`: `pending/processing/failed/completed`)

6. Xóa tài liệu  
`DELETE /api/v1/knowledge/:id`

7. Kiểm tra kho tri thức có rỗng hay không (dùng cho tự động xóa kho)  
`GET /api/v1/knowledge-bases/:id/knowledge?page=1&page_size=1`

8. Truy xuất (test recall + RAG chương trình chính)  
`POST /api/v1/knowledge-search`

## 4. Ánh xạ dữ liệu và trạng thái

1. Ánh xạ trường
- Local `knowledge_bases.external_kb_id` <-> WeKnora `knowledge_base.id`
- Local `knowledge_base_documents.external_doc_id` <-> WeKnora `knowledge.id`
- `sync_provider = "weknora"`

2. Trạng thái đồng bộ tài liệu (một trường)
- Sau khi vào hàng đợi: `pending`
- Bắt đầu tải lên: `uploading`
- Tải lên thành công: `uploaded`
- Đang phân tích: `parsing`
- Phân tích thành công: `synced`
- Tải lên thất bại: `upload_failed`
- Phân tích thất bại: `parse_failed`
- Lỗi nội bộ như vào hàng đợi thất bại: `failed`

3. Chiến lược polling phân tích (đề xuất)
- `parse_poll_interval_ms`: mặc định `1000`
- `parse_timeout_ms`: mặc định `120000`
- Timeout xử lý như `parse_failed` và ghi vào `sync_error`

## 5. Phương án sửa backend (manager/backend)

1. `manager/backend/controllers/knowledge_sync.go`
- Thêm `weknoraKnowledgeSyncConfig` và `parseWeknoraKnowledgeSyncConfig`.
- Thêm nhánh `weknora` cho `syncKnowledgeBaseWithProvider/syncKnowledgeBaseDeleteBestEffort/syncKnowledgeDocumentBestEffort/syncKnowledgeDocumentDeleteBestEffort`.
- Thêm wrapper HTTP WeKnora (header xác thực `X-API-Key`, format log request/response thống nhất với hiện có).
- Tải tài liệu lên thống nhất qua `/knowledge/file`:
  - Tài liệu dạng file: chuyển tiếp upload trực tiếp.
  - Tài liệu dạng text: chuyển thành byte stream UTF-8 `.md` tạm rồi upload.
- Cập nhật tài liệu dùng chiến lược “tạo mới rồi xóa cũ”, tránh phụ thuộc request body update chưa ổn định.
- Sau khi xóa tài liệu, truy vấn kho tri thức remote có rỗng hay không; nếu thỏa điều kiện thì xóa kho tri thức remote.

2. `manager/backend/controllers/knowledge.go`
- Thêm `weknora` vào validate provider của `CreateKnowledgeBaseDocumentByUpload`.
- Thêm nhánh `queryKnowledgeTestByWeknora` cho test recall `TestKnowledgeBaseSearch`.
- Xử lý ngưỡng giữ nguyên quy tắc hiện tại: ngưỡng request > ngưỡng kho tri thức > ngưỡng toàn cục.
- Nếu API search WeKnora không có tham số ngưỡng native, lọc lần hai theo `score` ở local.

3. `manager/backend/controllers/admin.go`
- Cấu trúc tổng hợp `knowledge_search` hiện có đã hỗ trợ nhiều provider, không cần sửa schema.
- Giữ nguyên cấu trúc output `knowledge.default_provider + knowledge.providers`.

## 6. Phương án sửa chương trình chính (internal/domain/rag)

1. Thêm `internal/domain/rag/weknora_searcher.go`
- Implement interface `Searcher`.
- Gọi `POST /api/v1/knowledge-search`, truy xuất chính xác theo `knowledge_base_ids`.
- Tái sử dụng cơ chế song song, timeout, gom kết quả chịu lỗi hiện có.
- Ánh xạ kết quả hit sang `KnowledgeSearchHit`:
  - `Content <- content`
  - `Title <- knowledge_title` (nếu rỗng thì fallback về tên kho tri thức local)
  - `Score <- score`

2. Sửa `internal/domain/rag/manager.go`
- Thêm nhánh `weknora` cho `getSearcher()`.
- Giữ nguyên logic đọc provider config (đọc từ `knowledge.providers.weknora`).

## 7. Sửa frontend quản trị (manager/frontend)

1. `manager/frontend/src/views/admin/KnowledgeSearchConfig.vue`
- Thêm `weknora` vào dropdown provider.
- Thêm các mục cấu hình đề xuất:
  - `base_url` (mặc định `http://127.0.0.1:8080`)
  - `api_key`
  - `score_threshold` (mặc định `0.2`)
  - `chunk_size` (mặc định `1000`)
  - `chunk_overlap` (mặc định `200`)
  - `separators` (mặc định `["\n\n","\n","。","！","？",";","；"]`)
  - `enable_multimodal` (mặc định `true`)
  - `embedding_model_id` (đề xuất bắt buộc)
  - `summary_model_id` (tùy chọn)
  - `rerank_model_id` (tùy chọn)
  - `vlm_model_id` (tùy chọn)
  - `parse_poll_interval_ms`, `parse_timeout_ms` (tùy chọn)

2. `manager/frontend/src/views/user/KnowledgeBases.vue`
- Hiển thị provider không cần thêm cột mới vì đã có trường provider.
- Thêm nhánh `weknora` cho `accept` của thao tác tải file lên.
- Thêm mô tả về đường upload WeKnora và phân tích bất đồng bộ.

## 8. Quyết định triển khai quan trọng (đề xuất xác nhận)

1. Tài liệu text có bắt buộc đi qua API thủ công hay không
- Đề xuất bản đầu thống nhất đi qua `/knowledge/file` (đóng gói text thành `.md`) để giảm khác biệt API và rủi ro tương thích.

2. Nguồn Embedding model
- Đề xuất trước mắt để `embedding_model_id` là trường bắt buộc cho quản trị viên.
- Có thể tăng cường sau: nếu rỗng, lúc khởi động gọi `/api/v1/models` để tự chọn model `Embedding` mặc định.

3. Giới hạn định dạng file
- Tài liệu WeKnora chưa nêu whitelist nghiêm ngặt; đề xuất bản đầu dùng “giới hạn frontend tương đối rộng + backend/remote fallback lỗi”.
- Nếu cần whitelist nghiêm ngặt, có thể thu hẹp ở giai đoạn hai dựa trên các định dạng ổn định đã kiểm thử.

## 9. Checklist xác minh

1. Quản trị viên thêm cấu hình `weknora` và đặt làm mặc định.
2. Sau khi người dùng thường tạo kho tri thức, hệ thống tự tạo remote knowledge-base và ghi ngược `external_kb_id`.
3. Sau khi thêm tài liệu text/tải file lên, trạng thái chuyển theo `uploading -> uploaded -> parsing -> synced`.
4. Khi phân tích thất bại, `sync_status=parse_failed` và ghi `sync_error`.
5. Sau khi xóa tài liệu, tài liệu remote bị xóa; nếu kho remote rỗng thì tự xóa kho theo chiến lược.
6. Kiểm thử truy xuất có thể trả kết quả hit từ WeKnora, ngưỡng có hiệu lực ở local.
7. Chuỗi chat chương trình chính có thể tự động kích hoạt truy xuất khi liên kết kho tri thức `weknora`.

## 10. Rủi ro và rollback

1. Rủi ro
- Khác biệt phiên bản WeKnora khiến trường request body thay đổi, đặc biệt là trường cấu hình khi tạo kho tri thức.
- Phân tích tài liệu có thể tốn thời gian dài, cần phối hợp polling và timeout.
- Nếu API search thiếu tham số ngưỡng native, cần lọc lần hai ở local.

2. Rollback
- Chỉ cần tắt/xóa cấu hình `weknora` để ngừng sử dụng, không ảnh hưởng `dify/ragflow` hiện có.
- Ở tầng code, nhánh provider có thể rollback độc lập, không liên quan thay đổi cấu trúc DB.
