# Báo cáo sweep cuối cho Việt hóa

## Kết quả đã làm

Đã quét toàn repo để tìm Unicode Han/Chinese, bỏ qua hoặc không ưu tiên xử lý với các vùng như `vendor`, `node_modules`, `dist`, lockfile, binary và generated assets.

Đã sửa an toàn thêm một số phần user-facing còn sót, tiêu biểu:

- `manager/frontend/src/views/OpenAPIDocs.vue`
- `manager/frontend/src/views/admin/Users.vue`
- `manager/frontend/src/views/user/APITokens.vue`

## Biên dịch/kiểm thử đã chạy

- Frontend build: `manager/frontend -> npm run build` ✅
- Backend test: `manager/backend -> go test ./...` ✅

## Phân loại các chuỗi Chinese/Han còn lại

### 1. Cần giữ vì protocol/provider/model/key/default content

Các nhóm này không đổi để tránh ảnh hưởng behavior hoặc dữ liệu kỹ thuật:

- `config/config.yaml`, `build/common/main_config.yaml`
  - `system_prompt`
  - `greeting_list`
  - `wakeup_words`
  - `instruct_text: "你好"`
- `manager/frontend/src/composables/useAgentFormOptions.js`
  - prompt mặc định của agent
  - keyword mặc định OpenClaw như `打开龙虾`, `进入龙虾`, `关闭龙虾`, `退出龙虾`
- `manager/backend/controllers/admin.go`
  - prompt fallback hardcode mặc định
- nhiều file config/provider form còn giữ tên provider/model theo sản phẩm, ví dụ `智谱AI`, `阿里云`, `豆包`, hoặc content mẫu provider-specific

### 2. Chỉ là comment/log/dev-only, không user-facing

Các nhóm này còn tiếng Trung nhưng không trả trực tiếp cho end user:

- `.github/workflows/*.yml`
- nhiều comment trong `cmd/*`, `internal/app/server/*`, `manager/backend/*`, `manager/frontend/src/*`
- các README nội bộ trong `internal/**/README.md`, `test/**/README.md`, `manager/backend/config/README.md`
- các file mock/test như `cmd/mock_ai_server/*`, `test/*`

### 3. Docs song ngữ hoặc nội dung đối chiếu thuật ngữ

Các phần dưới đây có thể còn tiếng Trung có chủ đích vì dùng để đối chiếu thuật ngữ hoặc giữ nguyên ví dụ kỹ thuật:

- `README.md` nếu còn giữ nội dung song ngữ/upstream
- một phần `docs/vi_vn_glossary.md` vì bảng glossary có cột `中文` làm đối chiếu thuật ngữ
- ví dụ prompt, wake word, keyword hoặc tên provider/model cần giữ nguyên để tránh đổi hành vi

### 4. Phần còn lại đáng chú ý trong UI/runtime nhưng hiện giữ nguyên có chủ đích

- `manager/frontend/src/views/OpenAPIDocs.vue`
  - vẫn còn nhiều ví dụ response/request và heading tiếng Trung trong nội dung tài liệu tĩnh
  - đây là user-facing, nhưng thuộc một cụm tài liệu lớn; có thể Việt hóa tiếp toàn bộ nếu muốn đồng nhất 100%
- `manager/frontend/src/views/mobile/MobileLogin.vue`, `MobileMore.vue`
  - còn nhiều nhãn/placeholder tiếng Trung
  - an toàn để Việt hóa tiếp ở vòng sau
- `manager/frontend/src/views/admin/*Config*.vue`
  - còn chuỗi Han trong comment dev-only và một số label/provider name lẫn lộn
  - cần rà kỹ để tránh đổi identifier hoặc tên provider/product
- `manager/frontend/src/views/user/AgentHistory.vue`
  - còn chủ yếu comment dev-only, không ảnh hưởng runtime text

## Danh sách giữ lại có lý do

| Khu vực | Ví dụ | Lý do giữ |
|---|---|---|
| Prompt mặc định | `你是一个叫小智/小志的台湾女孩...` | Nội dung mặc định có thể ảnh hưởng behavior của agent/thiết bị |
| Wake words | `小智`, `你好小智` | Từ khóa kích hoạt, đổi có thể ảnh hưởng tương thích |
| OpenClaw keywords | `打开龙虾`, `退出龙虾` | Trigger keyword mặc định, cần giữ để không phá flow hiện có |
| Provider/model branding | `智谱AI`, `阿里云`, `豆包` | Tên sản phẩm/nhà cung cấp, không nên dịch bừa |
| Docs song ngữ/đối chiếu | `README.md`, `docs/vi_vn_glossary.md` | Giữ phần tiếng Trung khi là ví dụ, keyword hoặc cột đối chiếu thuật ngữ |
| Workflow/dev comments | `.github/workflows`, comment code | Không phải text user-facing trực tiếp |

## Kết luận

Sweep cuối đã xử lý thêm một nhóm UI user-facing an toàn và xác nhận build/test phù hợp đều pass. Chuỗi Chinese/Han còn lại hiện chủ yếu rơi vào 4 nhóm: prompt/keyword mặc định ảnh hưởng behavior, tên provider/model, comment/log/dev-only, và docs gốc song ngữ/upstream.

Nếu muốn vòng cuối sâu hơn nữa, nên ưu tiên tiếp:

1. `manager/frontend/src/views/OpenAPIDocs.vue`
2. `manager/frontend/src/views/mobile/MobileLogin.vue`
3. `manager/frontend/src/views/mobile/MobileMore.vue`
4. các màn config admin còn label tiếng Trung nhưng không phải provider/key
