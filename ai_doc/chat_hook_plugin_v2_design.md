# Tài liệu thiết kế Chat Hook / Plugin V2

## 1. Mục tiêu tài liệu

Tài liệu này trả lời ba câu hỏi:

1. Ranh giới hợp lý của kiến trúc Hook trong chuỗi chat hiện tại là gì;
2. V2 nên ưu tiên bổ sung năng lực nào;
3. Làm thế nào để tiến hóa theo giai đoạn mà ít ảnh hưởng nhất tới luồng nghiệp vụ chính hiện có.

Mục tiêu của tài liệu này không phải là “xây dựng một plugin market hoàn chỉnh”, mà là đưa vào repo hiện tại một phương án Chat Hook / Plugin V2 **có thể quản trị, có thể mở rộng, có thể quan sát**.

---

## 2. TL;DR

### 2.1 Kết luận một câu

Phần triển khai hiện tại đã có nguyên mẫu **Chat Hook Framework** dùng được, nhưng chưa phù hợp để định nghĩa trực tiếp là “nền tảng plugin” hoàn chỉnh.

### 2.2 Luận điểm cốt lõi của V2

Trọng tâm của V2 không phải là “đập đi viết lại”, mà là:

- Giữ các điểm tích hợp nghiệp vụ hiện có;
- Tách rõ **Interceptor** (có thể sửa luồng chính) và **Observer** (chỉ quan sát);
- Bổ sung **hàng đợi có giới hạn, timeout, chiến lược drop, metric** cho thực thi async;
- Giới thiệu **PluginMeta + Registry + Lifecycle**;
- Hoàn thiện contract cho các event ASR / LLM / TTS / Metric.

### 2.3 Ưu tiên khuyến nghị

| Ưu tiên | Hạng mục đề xuất | Mục tiêu |
| --- | --- | --- |
| P0 | Quản trị Async Runtime | Tránh plugin chậm kéo sập hệ thống |
| P0 | Tách ngữ nghĩa Interceptor / Observer | Giảm nhầm lẫn ngữ nghĩa |
| P0 | Tài liệu contract payload | Giảm plugin dùng sai |
| P1 | PluginMeta / Registry | Quản lý được đăng ký plugin |
| P1 | Vòng đời / cấu hình | Hỗ trợ plugin có trạng thái |
| P2 | Nhiều queue / nhiều worker / tích hợp tracing | Hỗ trợ mở rộng phức tạp hơn |

---

## 3. Đánh giá hiện trạng

## 3.1 Phần thiết kế hiện tại đã đúng

Phần triển khai hiện tại đã có các ưu điểm sau:

1. **Chọn đúng điểm tích hợp**
   - Hook đã được gắn vào ASR final output, LLM input/output, TTS input/output start-stop, và Metric;
   - Đây đều là các vị trí có giá trị nghiệp vụ cao nhất trong luồng chat chính.

2. **Phân tầng cơ bản đúng**
   - `internal/pkg/hooks` chịu trách nhiệm framework thực thi chung;
   - `internal/domain/chat/hooks` chịu trách nhiệm context miền chat, event và typed payload;
   - `internal/app/server/chat/*` chỉ chịu trách nhiệm emit tại vị trí phù hợp.

3. **Typed façade đã được xây dựng**
   - Tầng nghiệp vụ không còn thao tác trực tiếp với `any`;
   - Đây là nền tảng tốt để V2 tiếp tục tăng cường contract và governance.

4. **Khả năng cắm mở đã dùng được**
   - Hiện tại đã có thể hỗ trợ plugin thống kê, rewrite text, chặn luồng, và các năng lực built-in khác.

## 3.2 Điểm yếu chính hiện tại

Vấn đề cần giải quyết nhất hiện tại không phải là “abstraction sai”, mà là “thiếu năng lực governance”.

### A. Trộn lẫn ngữ nghĩa

Hiện tại cùng một mô hình Hook đang gánh hai loại nhu cầu rất khác nhau:

- Interceptor có thể rewrite luồng chính;
- Metric / audit / telemetry chỉ quan sát.

Điều này gây ra các vấn đề:

- Ngữ nghĩa `stop` không phù hợp với mọi event;
- `payload rewrite` chỉ phù hợp với một số stage;
- Tác giả plugin khó hiểu event nào được phép làm gì.

### B. Thiếu governance cho thực thi async

Điểm đau của async execution hiện tại:

- Queue không có giới hạn;
- Thiếu timeout;
- Thiếu metric dropped;
- Tất cả async handler dùng chung một luồng consumer tuần tự;
- Thiếu cách cô lập plugin chậm.

### C. Cách đăng ký plugin còn hard-code

Hiện tại chủ yếu đăng ký bằng `RegisterBuiltinPlugins` trong code. Cách này đơn giản nhưng không thuận lợi cho:

- Xem hiện đang load plugin nào;
- Bật/tắt plugin;
- Cấu hình theo môi trường;
- Debug theo từng plugin.

### D. Contract chưa rõ

Dù đã có `ASROutputData` / `LLMInputData` / `LLMOutputData` / `TTSInputData` / `MetricData`, vẫn còn thiếu các ràng buộc sau:

- Field nào được phép sửa;
- Field nào không được phép set rỗng;
- Ngữ nghĩa nghiệp vụ của `stop` là gì;
- `Err` có được phép overwrite hay không;
- Thời gian tối đa plugin được phép chạy là bao lâu.

---

## 4. Định vị và nguyên tắc thiết kế V2

## 4.1 Định vị kiến trúc

V2 nên chính thức đặt tên hệ thống là:

> **Chat Interceptor & Observer Framework**

Thay vì gọi ngay là:

> Nền tảng plugin

Tên này sát hơn với năng lực thực tế ở giai đoạn hiện tại và giúp kiểm soát kỳ vọng trong team tốt hơn.

## 4.2 Nguyên tắc thiết kế

V2 nên tuân theo các nguyên tắc sau:

1. **Giữ ổn định luồng nghiệp vụ chính**
   - Không sửa lớn logic ASR / LLM / TTS hiện có;
   - Ưu tiên tăng cường ở tầng Hook Runtime và Domain Facade.

2. **Làm governance trước, platform hóa sau**
   - Giải quyết ngữ nghĩa, ranh giới, monitoring, ổn định trước;
   - Sau đó mới cân nhắc hệ sinh thái plugin phức tạp hơn.

3. **Phân biệt Interceptor và Observer trước**
   - Mọi hành vi có thể rewrite luồng chính phải được phân loại rõ là Interceptor;
   - Mọi hành vi quan sát read-only phải được phân loại rõ là Observer.

4. **Làm rõ contract trước, mở rộng số lượng plugin sau**
   - Khi không có contract, càng nhiều plugin thì chi phí bảo trì càng cao.

5. **Tiến hóa tăng dần, không rewrite một lần**
   - Ưu tiên thiết kế tương thích với interface emit hiện tại;
   - Migration cần triển khai theo giai đoạn.

---

## 5. Kiến trúc tổng thể V2

## 5.1 Cấu trúc ba tầng

V2 tiếp tục dùng cấu trúc ba tầng hiện tại nhưng tăng cường ranh giới trách nhiệm.

### Tầng 1: Luồng nghiệp vụ chính

Trách nhiệm:

- Emit tại các điểm quan trọng của ASR / LLM / TTS / Session Metric;
- Không trực tiếp nhận biết đăng ký plugin, chiến lược scheduling, lifecycle.

Không chịu trách nhiệm:

- Đăng ký plugin;
- Governance thực thi plugin;
- Quản lý metadata plugin.

### Tầng 2: Chat Domain Hook

Trách nhiệm:

- Định nghĩa event miền chat, typed payload, domain context;
- Cung cấp điểm vào thống nhất và ổn định cho code nghiệp vụ;
- Ràng buộc contract field và ngữ nghĩa stop/error.

### Tầng 3: Generic Runtime

Trách nhiệm:

- Đăng ký plugin;
- Sắp xếp và thực thi;
- Scheduling async;
- Timeout / drop / metrics;
- Quản lý lifecycle.

## 5.2 Vai trò Hook trong request path

```text
ASR final text
  -> ASR Output Interceptors
  -> LLM Input Interceptors
  -> LLM execution
  -> LLM Output Interceptors
  -> TTS Input Interceptors
  -> TTS Output Observers

Đồng thời:
  Metric Observers quan sát ở các stage turn_start / asr_first / asr_final /
  llm_start / llm_first / llm_end / tts_start / tts_first / tts_stop, v.v.
```

Cốt lõi của cách tách này là:

- Luồng nghiệp vụ chính chỉ quan tâm “emit lúc nào”;
- Interceptor chịu trách nhiệm rewrite;
- Observer chịu trách nhiệm quan sát;
- Runtime chịu trách nhiệm governance.

---

## 6. Thiết kế event model

## 6.1 Phân tầng event

### A. Event loại Interceptor

Dùng cho rewrite đồng bộ và điều khiển luồng.

Đề xuất giữ:

- `chat.asr.output`
- `chat.llm.input`
- `chat.llm.output`
- `chat.tts.input`

Loại event này nên có:

- Thực thi có thứ tự theo priority;
- Có thể sửa payload;
- Có thể `stop`;
- Có thể trả error;
- Bắt buộc return nhanh.

### B. Event loại Observer

Dùng cho quan sát, instrumentation, log, trace, audit, v.v.

Đề xuất phân loại là Observer:

- `chat.metric`
- `chat.tts.output.start`
- `chat.tts.output.stop`
- Audit / trace / debug event mở rộng sau này

Loại event này nên có:

- Mặc định read-only;
- Không được phép stop luồng chính;
- Không tham gia thay đổi payload của luồng chính;
- Có thể thực thi async;
- Lỗi chỉ ảnh hưởng chuỗi quan sát.

## 6.2 Đề xuất đặt tên event

Tiếp tục dùng naming phân tầng hiện có, không nên đổi lớn hệ thống tên ngay. Khuyến nghị giữ:

- `chat.asr.output`
- `chat.llm.input`
- `chat.llm.output`
- `chat.tts.input`
- `chat.tts.output.start`
- `chat.tts.output.stop`
- `chat.metric`

Lý do:

- Tên hiện tại đã trực quan;
- Tương thích implementation hiện có;
- Chi phí migration thấp nhất.

---

## 7. Thiết kế runtime

## 7.1 PluginMeta

V2 thêm metadata thống nhất để phục vụ hiển thị, bật/tắt, chẩn đoán và sắp xếp.

```go
package hooks

type PluginKind string

const (
    PluginKindInterceptor PluginKind = "interceptor"
    PluginKindObserver    PluginKind = "observer"
)

type PluginMeta struct {
    Name        string
    Version     string
    Description string
    Priority    int
    Enabled     bool
    Kind        PluginKind
    Stage       string
}
```

### Giải thích thiết kế

- `Name`: định danh duy nhất toàn cục;
- `Version`: phục vụ tương thích và rollout sau này;
- `Priority`: căn cứ sắp xếp;
- `Enabled`: công tắc runtime;
- `Kind`: phân biệt interceptor / observer;
- `Stage`: khai báo stage được mount.

## 7.2 Registry

V2 khuyến nghị thêm registry rõ ràng, thay vì để Runtime tự quyết “có plugin nào”.

```go
package hooks

type Registration struct {
    Meta     PluginMeta
    Register func(*Hub)
}

type Registry interface {
    Add(reg Registration)
    List() []Registration
}
```

### Registry chịu trách nhiệm gì

- Lưu định nghĩa plugin;
- Expose danh sách đăng ký có thể enumerate;
- Hỗ trợ lọc bật/tắt theo cấu hình;
- Cung cấp dữ liệu nền cho debug và quan sát.

### Registry không chịu trách nhiệm gì

- Không trực tiếp thực thi plugin;
- Không trực tiếp chứa trạng thái nghiệp vụ;
- Không thay thế logic thực thi của Runtime.

## 7.3 Lifecycle

Cung cấp lifecycle tối thiểu cho plugin có trạng thái.

```go
package hooks

type Lifecycle interface {
    Init(context.Context) error
    Close() error
}
```

Khuyến nghị:

- Plugin stateless không cần implement;
- Plugin có cache, background task, connection pool thì implement interface này;
- Runtime quản lý thống nhất thời điểm gọi.

## 7.4 Interface Interceptor

```go
package hooks

type Interceptor[T any] interface {
    Meta() PluginMeta
    Handle(Context, T) (T, bool, error)
}
```

Ý đồ thiết kế:

- Giữ ba năng lực “rewrite + stop + error”;
- Dùng generic để tăng ràng buộc ở compile time;
- Giảm rủi ro dùng sai do `any`.

## 7.5 Interface Observer

```go
package hooks

type Observer[T any] interface {
    Meta() PluginMeta
    Handle(Context, T)
}
```

Ý đồ thiết kế:

- Làm rõ “chỉ quan sát, không rewrite”;
- Loại bỏ việc lạm dụng `stop` ở mức ngữ nghĩa;
- Thuận tiện để sau này áp dụng scheduling strategy riêng cho observer.

## 7.6 Async Runtime

### Vấn đề hiện tại

Vấn đề lớn nhất của mô hình async hiện tại không phải là “không chạy được”, mà là “thiếu ranh giới”.

### Mục tiêu thiết kế V2

Bổ sung các năng lực sau cho async observer:

- bounded queue;
- timeout;
- thống kê dropped;
- metric thực thi theo plugin;
- sau này có thể mở rộng thành nhiều queue / nhiều worker.

### Cấu hình đề xuất

```go
package hooks

type AsyncConfig struct {
    QueueSize    int
    WorkerCount  int
    DropWhenFull bool
    Timeout      time.Duration
}
```

### Giá trị mặc định khuyến nghị

- `QueueSize = 1024`
- `WorkerCount = 1`
- `DropWhenFull = true`
- `Timeout = 200ms`

### Chiến lược khuyến nghị

1. Mặc định giữ một worker để đảm bảo ngữ nghĩa thứ tự;
2. Khi queue đầy, ưu tiên drop observer event thay vì làm chậm luồng chính;
3. Ghi nhận dropped count và timeout count;
4. Nếu sau này xuất hiện observer tải cao, tách queue theo event hoặc plugin.

---

## 8. Contract miền nghiệp vụ

V2 bắt buộc viết rõ contract payload.

## 8.1 ASROutputData

Mục đích: rewrite text cuối của ASR và kết quả speaker.

| Field | Có thể sửa | Ghi chú |
| --- | --- | --- |
| `Text` | Có | Có thể clean, normalize, filter |
| `SpeakerResult` | Có | Có thể chỉnh hoặc enrich speaker |

Ràng buộc:

- Plugin không được block lâu;
- `stop=true` nghĩa là text lượt này không đi tiếp vào LLM;
- Nếu trả text rỗng, plugin tự chịu hậu quả.

## 8.2 LLMInputData

Mục đích: rewrite message và tool trước khi gửi request LLM.

| Field | Có thể sửa | Ghi chú |
| --- | --- | --- |
| `UserMessage` | Có | Không được set rỗng |
| `RequestMessages` | Có | Có thể cắt, reorder, inject system prompt |
| `Tools` | Có | Có thể filter hoặc append |

Ràng buộc:

- `UserMessage` không được là `nil`;
- Plugin phải đảm bảo output vẫn thỏa yêu cầu input tối thiểu của LLM Provider phía dưới;
- `stop=true` nghĩa là request LLM lần này bị intercept và kết thúc.

## 8.3 LLMOutputData

Mục đích: rewrite text hiển thị hoặc bổ sung ngữ nghĩa lỗi sau khi LLM output hoàn tất.

| Field | Có thể sửa | Ghi chú |
| --- | --- | --- |
| `FullText` | Có | Có thể rewrite an toàn, format lại, tối ưu cho giọng nói |
| `Err` | Thận trọng | Nên bổ sung context, không nên nuốt trực tiếp lỗi tầng dưới |

Ràng buộc:

- Không khuyến nghị plugin âm thầm overwrite lỗi thật tầng dưới;
- `stop=true` nghĩa là không tiếp tục đi vào TTS hoặc cập nhật message;
- Sau này có thể tách `Err` thành `OriginErr` / `DisplayErr`.

## 8.4 TTSInputData

Mục đích: xử lý text trước khi vào TTS để dễ đọc thành tiếng hơn.

| Field | Có thể sửa | Ghi chú |
| --- | --- | --- |
| `Text` | Có | Có thể chuyển số, dấu câu, emoji sang dạng dễ đọc |
| `IsStart` | Mặc định không | Xem là field ranh giới protocol |
| `IsEnd` | Mặc định không | Xem là field ranh giới protocol |

Ràng buộc:

- Plugin thông thường chỉ nên sửa `Text`;
- `IsStart` / `IsEnd` nên dành cho plugin quyền cao hơn hoặc plugin chuyên dụng;
- `stop=true` nghĩa là fragment hiện tại không đi vào TTS.

## 8.5 MetricData

Mục đích: quan sát chuỗi xử lý.

| Field | Có thể sửa | Ghi chú |
| --- | --- | --- |
| `Stage` | Không | Read-only |
| `Ts` | Không | Read-only |
| `Err` | Không | Read-only, chỉ dùng để quan sát |

Ràng buộc:

- Không được stop luồng chính;
- Không được rewrite rồi feedback lại luồng chính;
- Chỉ dùng cho log, metric, tracing, debug.

---

## 9. Mô hình cấu hình

Khuyến nghị thêm model cấu hình tối thiểu cho hệ thống Hook:

```yaml
chat_hooks:
  enabled: true
  async:
    queue_size: 1024
    worker_count: 1
    drop_when_full: true
    timeout_ms: 200
  plugins:
    statistic_plugin:
      enabled: true
      priority: 100
```

Mục tiêu thiết kế cấu hình:

- Hỗ trợ bật/tắt plugin;
- Hỗ trợ override priority;
- Hỗ trợ kiểm soát tham số runtime async;
- Chừa không gian cho plugin-level config schema trong tương lai.

---

## 10. Yêu cầu quan sát

Runtime tối thiểu nên thu thập các metric sau:

- Số lần gọi plugin;
- Thời gian chạy plugin;
- Số lần error;
- Số lần stop (interceptor);
- Số lần dropped (observer async);
- Số lần timeout;
- Độ dài async queue hiện tại.

Khuyến nghị log/metric chứa:

- `plugin_name`
- `plugin_kind`
- `stage`
- `priority`
- `duration_ms`
- `result`

Nếu sau này tích hợp tracing, có thể ghi thêm:

- session_id
- device_id
- turn_id
- correlation_id

---

## 11. Kế hoạch migration

## 11.1 Giai đoạn 1: Tăng cường Runtime (P0)

Mục tiêu: tăng độ ổn định mà không sửa điểm tích hợp nghiệp vụ.

Hạng mục:

- Thêm bounded queue cho async observer;
- Thêm timeout và thống kê dropped;
- Thêm metric runtime cơ bản;
- Giữ tương thích interface `Emit` / `RegisterSync` / `RegisterAsync` hiện có.

Kết quả:

- Độ ổn định tăng;
- Có ranh giới cho việc mở rộng observer sau này.

## 11.2 Giai đoạn 2: Phân tầng ngữ nghĩa (P0)

Mục tiêu: phân biệt rõ Interceptor và Observer.

Hạng mục:

- Thêm façade rõ ràng ở tầng Domain Hook;
- Chuyển event `Metric` sang ngữ nghĩa observer tiêu chuẩn;
- Làm rõ tập event không cho phép stop.

Kết quả:

- Ngữ nghĩa rõ hơn;
- Tác giả plugin ít dễ dùng sai hơn.

## 11.3 Giai đoạn 3: Registry + Meta (P1)

Mục tiêu: tách “có những plugin nào” khỏi logic hard-code.

Hạng mục:

- Thêm `PluginMeta`;
- Thêm `Registration` / `Registry`;
- Hỗ trợ bật/tắt plugin theo cấu hình;
- Hỗ trợ liệt kê plugin đang được load.

Kết quả:

- Đăng ký minh bạch;
- Thuận tiện debug, quan sát và quản trị cấu hình.

## 11.4 Giai đoạn 4: Contract và lifecycle (P1)

Mục tiêu: chính thức hóa ranh giới plugin.

Hạng mục:

- Cố định contract payload;
- Thêm `Lifecycle`;
- Thêm quy trình init/close cho plugin có trạng thái.

Kết quả:

- Phù hợp hơn để chứa plugin built-in phức tạp;
- Dễ tiến hóa thành hệ thống plugin hoàn chỉnh hơn.

## 11.5 Giai đoạn 5: Năng lực nâng cao (P2)

Mục tiêu: hỗ trợ hệ sinh thái plugin phức tạp hơn.

Hạng mục:

- Tách queue theo event;
- Runtime observer nhiều worker;
- Tích hợp sâu tracing / metrics;
- Chiến lược thực thi cô lập cho plugin nặng.

---

## 12. Không phải mục tiêu

V2 hiện tại **không theo đuổi**:

- Sandbox cho plugin bên thứ ba không tin cậy;
- Hệ thống plugin RPC ngoài tiến trình;
- Hệ sinh thái hot-load plugin phức tạp;
- Plugin market hoàn chỉnh.

Các năng lực này nên được đánh giá ở version cao hơn trong tương lai, không nên đưa độ phức tạp vào quá sớm.

---

## 13. Khuyến nghị triển khai

Nếu iteration hiện tại chỉ được làm 3 việc, khuyến nghị thứ tự như sau:

1. **Làm governance Async Runtime trước**
   - Đây là bước có lợi ích ổn định cao nhất.

2. **Sau đó tách ngữ nghĩa Interceptor / Observer**
   - Đây là bước hiệu quả nhất để giảm rủi ro dùng sai.

3. **Hoàn thiện contract và Meta/Registry**
   - Đây là bước then chốt để đẩy hệ thống từ “dùng được” thành “quản lý được”.

---

## 14. Tổng kết

Mục tiêu của V2 không phải là đóng gói hệ thống Hook hiện tại thành một “nền tảng plugin” nghe có vẻ lớn hơn, mà là tiến hóa nó thành một framework mở rộng thật sự:

- Rõ ngữ nghĩa;
- Kiểm soát được thực thi;
- Quan sát được;
- Mở rộng tăng dần được;
- Tương thích với luồng chat chính hiện tại.

Vì vậy, điều quan trọng nhất của V2 không phải là “thêm nhiều plugin hơn”, mà là hoàn tất ba việc sau trước:

- Phân rõ **Interceptor** và **Observer**;
- Bổ sung đầy đủ **ranh giới async runtime**;
- Xây dựng **contract payload và metadata plugin**.

Sau khi hoàn tất ba bước này, hệ thống Hook của repo hiện tại mới thật sự có nền tảng để tiến hóa lâu dài.
