# Phương án tích hợp asr_server theo cùng mô hình với manager/backend

## Mục tiêu

- **Giữ asr_server ở dạng repo độc lập**: có `go.mod`, `main.go` riêng và có thể clone, build, chạy độc lập.
- **Chương trình chính có thể khởi tạo asr_server**: giống `manager/backend`, tiến trình chính tham chiếu thư mục con qua `replace` và khi cần có thể khởi động dịch vụ HTTP của asr_server trong cùng tiến trình trên một cổng riêng, không cần chạy tiến trình riêng.

## Cách đưa vào: khuyến nghị dùng Git Submodule

Repo chính có thể có thư mục `asr_server/` theo hai cách:

| Cách | Mô tả |
|------|------|
| **Git Submodule (khuyến nghị)** | asr_server vẫn là repo Git độc lập; repo chính dùng `git submodule add` để tham chiếu. Thư mục thu được trỏ tới một commit cụ thể của asr_server; repo chính chỉ ghi lại đường dẫn submodule và commit hash. |
| Copy/di chuyển code | Đưa trực tiếp code asr_server vào thư mục repo chính; asr_server dùng chung lịch sử Git với repo chính hoặc trở thành một phần của repo chính. |

Phần bên dưới mô tả theo cách **Submodule**; phía repo chính, logic `replace` và khởi động nhúng giống với cách copy code.

## Tham khảo mô hình tích hợp của manager/backend

| Hạng mục | Cách làm của manager/backend |
|--------|----------------------|
| Thư mục | `manager/backend/` trong repo chính |
| Tên module | `xiaozhi/manager/backend`, nằm trong `go.mod` của backend |
| Tham chiếu từ repo chính | `replace xiaozhi/manager/backend => ./manager/backend` |
| Chạy độc lập | `manager/backend/main.go`: LoadWithPath → database.Init → router.Setup → r.Run() |
| Nhúng trong chương trình chính | `cmd/server/manager_http.go`: dùng cùng config/database/router và tự chạy `http.Server` trên cổng khác |

## Thiết kế tích hợp asr_server theo mô hình trên

### 1. Thư mục và module theo cách Submodule

- **asr_server cần có repo Git độc lập trước**. Nếu hiện đang nằm trong monorepo, có thể tách thành repo riêng hoặc dùng URL repo asr_server hiện có.
- **Thêm submodule trong repo chính** bằng lệnh sau tại root repo chính, khi thư mục `asr_server` chưa tồn tại:

  ```bash
  cd xiaozhi-esp32-server-golang
  git submodule add <URL repo asr_server> asr_server
  ```

  Sau khi hoàn tất, repo chính sẽ có:
  - Thư mục `asr_server/`, nội dung là một commit đang checkout của repo asr_server.
  - File `.gitmodules`, cùng bản ghi submodule có thể xem bằng `git submodule status`.

- **Đường dẫn thư mục**: trong repo chính là `xiaozhi-esp32-server-golang/asr_server/`, giống cách copy code. Code Go và `replace` trong `go.mod` của repo chính đều trỏ tới `./asr_server`.
- **Tên module**: giữ tên module hiện có của asr_server là **`voice_server`**, để khi là repo độc lập vẫn có thể `go build` trực tiếp mà không cần đổi import.
- **go.mod của repo chính**: thêm dòng:
  - `replace voice_server => ./asr_server`
- **go.mod của asr_server**: giữ `module voice_server`, không tham chiếu repo chính. Khi đứng độc lập thì không cần `replace`; khi tích hợp vào repo chính thì chỉ cần `replace` ở phía repo chính.

**Khi clone repo chính cần kéo submodule** bằng một trong hai cách:

```bash
# Clone và kéo submodule cùng lúc
git clone --recurse-submodules <URL repo chính>

# Hoặc clone trước rồi khởi tạo submodule
git clone <URL repo chính>
cd xiaozhi-esp32-server-golang
git submodule update --init --recursive
```

**CI / build tự động**: nếu repo chính cần build phần phụ thuộc asr_server, phải chạy `git submodule update --init --recursive` trước khi build, hoặc clone bằng `--recurse-submodules`.

### 2. Chạy độc lập, asr_server vẫn là repo riêng

- Khi clone hoặc mở riêng thư mục `asr_server`:
  - `go build -o asr_server .`
  - `./asr_server` dùng `config.json`, hoặc dùng `-config` để chỉ định đường dẫn, hành vi giữ như hiện tại.
- Không phụ thuộc repo chính; `replace` của repo chính chỉ ảnh hưởng quá trình build repo chính.

### 3. Chương trình chính khởi tạo asr_server nhúng

- **Điểm vào**: thêm `cmd/server/asr_server_http.go` trong repo chính, cùng cấp với `manager_http.go`.
- **Logic**, tương tự `manager_http`:
  1. Tiến trình chính quyết định có gọi hay không theo cấu hình, ví dụ `-asr-enable` và `-asr-config`.
  2. Dùng các package của asr_server:
     - `voice_server/config`: `InitConfig(configPath)`, rồi `GetConfig()` để lấy `*Config`.
     - `voice_server/internal/bootstrap`: `InitApp(cfg)` để lấy `*AppDependencies`.
     - `voice_server/internal/router`: `NewRouter(deps)` để lấy `*gin.Engine`.
  3. Dùng `deps.RateLimiter.Middleware(r)` làm Handler, chạy `http.Server` trên **cổng riêng**, ví dụ 8080, trong goroutine bằng `ListenAndServe`.
  4. Khi thoát, cung cấp `StopAsrServerHTTP()` để gọi `Shutdown` cho `http.Server` và giải phóng tài nguyên cần thiết, ví dụ component trong bootstrap cần `Close`.
- **Cấu hình**: asr_server vẫn dùng `config.json` riêng; khi nhúng, đường dẫn file cấu hình do tham số tiến trình chính hoặc cấu hình repo chính quyết định, ví dụ `asr_server/config.json` hoặc `config/asr_server.json`.

### 4. Danh sách thay đổi trong repo chính theo cách Submodule

| Vị trí | Thay đổi |
|------|------|
| Root repo chính | Chạy `git submodule add <URL repo asr_server> asr_server`, tạo thư mục `asr_server/` và `.gitmodules`. asr_server cần có repo Git độc lập trước. |
| `xiaozhi-esp32-server-golang/go.mod` | Thêm `replace voice_server => ./asr_server`; nếu code repo chính import `voice_server`, thêm `voice_server` vào `require`, hoặc để `go mod tidy` tự bổ sung. |
| `xiaozhi-esp32-server-golang/cmd/server/main.go` | Parse `-asr-enable`, `-asr-config`; nếu bật thì gọi `StartAsrServerHTTP(configPath)` trước `Run()`; sau `<-quit` gọi `StopAsrServerHTTP()`. |
| Thêm `xiaozhi-esp32-server-golang/cmd/server/asr_server_http.go` | Implement `StartAsrServerHTTP(configPath string)` và `StopAsrServerHTTP()`, dùng `voice_server/config`, `voice_server/internal/bootstrap`, `voice_server/internal/router`, cùng mô hình với `manager_http`. |

### 5. Phần asr_server cần expose

- **config**: đã có `InitConfig(path)` và `GetConfig()`, tiến trình chính có thể dùng trực tiếp.
- **bootstrap**: đã có `InitApp(cfg *config.Config)`, trả về `*AppDependencies`, tiến trình chính có thể dùng trực tiếp.
- **router**: đã có `NewRouter(deps) *gin.Engine`; tiến trình chính dùng `deps.RateLimiter.Middleware(r)` làm Handler.
- **Shutdown mềm**: nếu trong bootstrap có tài nguyên cần `Close()`, ví dụ VAD pool hoặc recognizer toàn cục, nên cung cấp hàm thống nhất như `Shutdown(deps *AppDependencies)` để `StopAsrServerHTTP()` gọi. Nếu hiện chưa có, có thể trước mắt chỉ `Server.Shutdown`, rồi bổ sung sau.

### 6. Phụ thuộc và build

- Dependency của asr_server như sherpa-onnx, qdrant, ten-vad vẫn giữ trong **asr_server/go.mod**. Repo chính **không** đưa dependency của asr_server trực tiếp lên `require` của `go.mod` chính, mà chỉ tham chiếu submodule bằng `require voice_server` hoặc tương đương; `go mod tidy` sẽ đồng bộ dependency cần thiết cho build repo chính.
- Nếu khi build repo chính báo thiếu dependency, có thể thêm rõ các dependency trực tiếp mà asr_server dùng vào `require` của `go.mod` chính.
- CGO và native lib như `ten_vad`, `.so/.dll` của sherpa-onnx vẫn bố trí theo cách hiện tại của asr_server, trong thư mục asr_server hoặc thư mục `lib/` thống nhất của repo chính, rồi mô tả trong script/tài liệu build.

### 7. Khác biệt so với manager/backend

- Module của manager/backend là `xiaozhi/manager/backend`, còn asr_server giữ `voice_server`, để khi asr_server là repo độc lập thì không cần đổi import.
- Repo chính chỉ cần `replace voice_server => ./asr_server`, không cần đổi đường dẫn package bên trong asr_server.
- Cách “khởi tạo” trong chương trình chính giống nhau: không gọi `main()` của asr_server, chỉ dùng lại config + bootstrap + router và chạy một HTTP service có cổng riêng trong tiến trình chính.

### 8. Tóm tắt theo cách Submodule

- **Repo độc lập**: asr_server là repo Git độc lập, có `go.mod` riêng với `module voice_server` và `main.go`, có thể clone, build, chạy độc lập.
- **Tích hợp vào repo chính**: repo chính dùng **Git submodule** để tham chiếu asr_server, tạo thư mục `asr_server/`; repo chính dùng `replace voice_server => ./asr_server`; sau khi clone repo chính cần chạy `git submodule update --init`, hoặc clone bằng `git clone --recurse-submodules`.
- **Khởi tạo trong chương trình chính**: repo chính thêm `asr_server_http.go`, theo cấu hình để khởi động dịch vụ HTTP của asr_server trong cùng tiến trình trên một cổng riêng, logic căn chỉnh với `manager_http.go`.

**Ghi chú biên dịch**: asr_server phụ thuộc sherpa-onnx qua CGO, nên repo chính dùng **build tag** để phần nhúng là tùy chọn:

- **Biên dịch mặc định**, không bật asr_server nhúng: `go build -o xiaozhi_server ./cmd/server`; khi đó `-asr-enable` sẽ báo asr_server chưa được biên dịch vào binary.
- **Biên dịch có asr_server nhúng**: `go build -tags asr_server -o xiaozhi_server ./cmd/server`; máy build cần có CGO và môi trường sherpa-onnx cần thiết.

Nếu xác nhận triển khai theo phương án này, có thể tiếp tục chi tiết hóa danh sách trách nhiệm của `Shutdown(deps)`, cổng mặc định, đường dẫn cấu hình và tên tham số/default trong `main.go` của repo chính.
