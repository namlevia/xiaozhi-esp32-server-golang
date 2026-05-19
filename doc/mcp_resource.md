# Tài liệu kiểu nội dung trả về khi gọi công cụ MCP

## Tổng quan

Tài liệu này mô tả chi tiết các kiểu nội dung trả về khi gọi công cụ mà chương trình hỗ trợ. Chương trình dùng **hệ thống response có cấu trúc**, hỗ trợ xử lý và render nhiều kiểu nội dung.

## 🔧 Luồng xử lý cốt lõi

### Xử lý response khi gọi công cụ

Bộ xử lý cốt lõi của response gọi công cụ chịu trách nhiệm:

1. **Thực thi gọi công cụ**: duyệt toàn bộ request gọi công cụ
2. **Parse kết quả**: parse kết quả công cụ trả về
3. **Nhận diện kiểu nội dung**: xử lý khác nhau theo kiểu nội dung
4. **Render tài nguyên**: xử lý nhiều kiểu nội dung như audio, text, resource link

## 📋 Kiểu nội dung được hỗ trợ

### 1. Nội dung audio (AudioContent)

**Kiểu**: `mcp_go.AudioContent`

**Đặc điểm**:
- Chứa dữ liệu audio mã hóa Base64
- Hỗ trợ nhiều định dạng audio (MIME Type)
- Phát trực tiếp và chấm dứt xử lý LLM tiếp theo

**Luồng xử lý**:
```go
if audioContent, ok := content.(mcp_go.AudioContent); ok {
    // Decode dữ liệu audio Base64
    rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Data)
    // Dùng music_player để phát audio
    audioChan, err := play_music.PlayMusicFromAudioData(ctx, rawAudioData, ...)
    // Gửi message trạng thái phát
    l.serverTransport.SendSentenceStart(playText)
    // Phát audio qua TTS manager
    l.ttsManager.SendTTSAudio(ctx, audioChan, true)
}
```

**Kịch bản sử dụng**:
- Công cụ phát nhạc
- Công cụ tổng hợp giọng nói
- Phát file audio

### 2. Liên kết tài nguyên (ResourceLink)

**Kiểu**: `mcp_go.ResourceLink`

**Đặc điểm**:
- Chứa resource URI và metadata
- Hỗ trợ đọc phân trang tài nguyên lớn
- Xử lý streaming, phù hợp với file lớn
- Dùng cơ chế Pipe để phát audio stream realtime

**Luồng xử lý**:
```go
if resourceLink, ok := content.(mcp_go.ResourceLink); ok {
    // Tạo Pipe để truyền streaming
    pipeReader, pipeWriter = io.Pipe()
    
    // Khởi động goroutine đọc phân trang
    go func() {
        // Đọc tài nguyên theo phân trang
        resourceResult, err := client.ReadResource(readCtx, mcp_go.ReadResourceRequest{
            Params: mcp_go.ReadResourceParams{
                URI: resourceLink.URI,
                Arguments: map[string]any{
                    "url": resourceLink.Description, 
                    "start": start, 
                    "end": start + page,
                },
            },
        })
        
        // Xử lý BlobResourceContents
        for _, content := range resourceResult.Contents {
            if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
                // Decode và gửi vào channel audio stream
                rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
                streamChan <- rawAudioData
            }
        }
    }()
    
    // Dùng music_player để phát audio stream
    audioChan, err := play_music.PlayMusicFromPipe(ctx, pipeReader, ...)
}
```

**Chi tiết tham số đọc phân trang**:

#### Định dạng tham số request
```go
Arguments: map[string]any{
    "url": resourceLink.Description,  // URL tài nguyên thực tế
    "start": start,                   // Vị trí byte bắt đầu
    "end": start + page,              // Vị trí byte kết thúc
}
```

#### Mô tả tham số
- **url**: địa chỉ URL tài nguyên thực tế, lấy từ `resourceLink.Description`
- **start**: vị trí byte bắt đầu, đếm từ 0
- **end**: vị trí byte kết thúc (không bao gồm), tức phạm vi đọc [start, end)
- **Kích thước trang**: do hằng `McpReadResourcePageSize` định nghĩa, mặc định 100KB

#### Luồng đọc phân trang
```go
start := 0
page := McpReadResourcePageSize  // 100 * 1024
totalRead := 0
pageCount := 0

for {
    // Tạo context có timeout
    readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    
    // Gửi request đọc phân trang
    resourceResult, err := client.ReadResource(readCtx, mcp_go.ReadResourceRequest{
        Params: mcp_go.ReadResourceParams{
            URI: resourceLink.URI,
            Arguments: map[string]any{
                "url": resourceLink.Description, 
                "start": start, 
                "end": start + page,
            },
        },
    })
    cancel()
    
    // Xử lý BlobResourceContents trả về
    for _, content := range resourceResult.Contents {
        if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
            // Decode dữ liệu Base64
            rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
            
            // Kiểm tra có phải dấu kết thúc hay không
            if string(rawAudioData) == McpReadResourceStreamDoneFlag {
                return nil // Đọc xong
            }
            
            // Gửi vào channel audio stream
            streamChan <- rawAudioData
            totalRead += len(rawAudioData)
        }
    }
    
    // Kiểm tra điều kiện đọc xong
    if len(rawAudioData) < page || !hasData {
        return nil // Đọc xong
    }
    
    // Cập nhật vị trí bắt đầu
    start += page
    pageCount++
}
```

#### Cơ chế xử lý streaming

**Kiến trúc truyền Pipe**:
```go
// Tạo Pipe để truyền audio stream
pipeReader, pipeWriter = io.Pipe()

// Khởi động goroutine ghi dữ liệu
go func() {
    for {
        select {
        case audioData, ok := <-streamChan:
            if !ok {
                pipeWriter.Close()
                return
            }
            pipeWriter.Write(audioData)
        case <-ctx.Done():
            return
        }
    }
}()

// Dùng music_player để phát audio từ Pipe
audioChan, err := play_music.PlayMusicFromPipe(ctx, pipeReader, ...)
```

#### Cơ chế xử lý lỗi

**Retry khi timeout**:
```go
if err != nil {
    // Nếu là lỗi timeout thì thử retry
    if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
        log.Warnf("Đọc tài nguyên timeout, đang thử lại...")
        time.Sleep(1 * time.Second)
        continue
    }
    return fmt.Errorf("đọc tài nguyên thất bại: %v", err)
}
```

**Hủy context**:
```go
select {
case <-ctx.Done():
    log.Debugf("đã hủy đọc tài nguyên")
    return nil
case streamChan <- rawAudioData:
    // Gửi dữ liệu bình thường
}
```

#### Đặc điểm cơ chế phân trang
- **Tối ưu bộ nhớ**: đọc phân trang tránh tải file lớn vào bộ nhớ một lần
- **Xử lý streaming**: vừa đọc vừa phát, hỗ trợ audio stream realtime
- **Tự động kết thúc**: phát hiện cờ `McpReadResourceStreamDoneFlag` để xác định đã đọc xong
- **Khôi phục lỗi**: hỗ trợ retry timeout và hủy context
- **Phát realtime**: dùng cơ chế Pipe để vừa đọc vừa phát
- **Kiểm soát timeout**: mỗi lần đọc phân trang đều có giới hạn timeout 30 giây

#### Tham số cấu hình
- **McpReadResourcePageSize**: kích thước trang, mặc định 100KB (100 * 1024)
- **McpReadResourceStreamDoneFlag**: cờ kết thúc stream, là `"[DONE]"`
- **Timeout đọc**: thời gian timeout mỗi lần đọc phân trang, mặc định 30 giây
- **Cơ chế retry**: lỗi timeout tự động retry, cách nhau 1 giây

**Kịch bản sử dụng**:
- Phát file audio lớn
- Xử lý tài nguyên streaming media
- Truy cập tài nguyên mạng
- Phát audio stream realtime

### 3. Nội dung văn bản (TextContent)

**Kiểu**: `mcp_go.TextContent`

**Đặc điểm**:
- Nội dung text thuần
- Được cộng dồn vào response message
- Không chấm dứt xử lý tiếp theo

**Luồng xử lý**:
```go
if textContent, ok := content.(mcp_go.TextContent); ok {
    mcpContent += textContent.Text
}
```

**Kịch bản sử dụng**:
- Trả về kết quả truy vấn
- Hiển thị thông tin trạng thái
- Hiển thị thông báo lỗi

### 4. Nội dung tài nguyên Blob (BlobResourceContents)

**Kiểu**: `mcp_go.BlobResourceContents`

**Đặc điểm**:
- Nội dung dữ liệu nhị phân
- Mã hóa Base64
- Hỗ trợ xử lý streaming

**Luồng xử lý**:
```go
if audioContent, ok := content.(mcp_go.BlobResourceContents); ok {
    rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
    // Kiểm tra có phải dấu kết thúc hay không
    if string(rawAudioData) == McpReadResourceStreamDoneFlag {
        return nil
    }
    // Gửi vào channel audio stream
    streamChan <- rawAudioData
}
```

## 🏗️ Hệ thống response có cấu trúc

### Phân loại kiểu response

Chương trình hỗ trợ bốn kiểu response chính:

#### 1. Response loại hành động (MCPActionResponse)
- **Mục đích**: thực thi hành động cụ thể, như phát nhạc, thoát hội thoại
- **Tính kết thúc**: có thể cấu hình, thường chấm dứt xử lý LLM tiếp theo
- **Cờ điều khiển**: `FinalAction`, `NoFurtherResponse`, `SilenceLLM`

#### 2. Response loại audio (MCPAudioResponse)
- **Mục đích**: phát tài nguyên audio
- **Tính kết thúc**: thường chấm dứt xử lý tiếp theo
- **Đặc điểm**: chứa dữ liệu audio và thông tin phát

#### 3. Response loại nội dung (MCPContentResponse)
- **Mục đích**: trả về dữ liệu truy vấn, thông tin trạng thái
- **Tính kết thúc**: không chấm dứt xử lý tiếp theo
- **Đặc điểm**: chứa dữ liệu và gợi ý hiển thị

#### 4. Phản hồi loại lỗi (MCPErrorResponse)
- **Mục đích**: xử lý lỗi thống nhất
- **Tính kết thúc**: không chấm dứt xử lý tiếp theo
- **Đặc điểm**: chứa mã lỗi và gợi ý

### Interface xử lý response

```go
type MCPResponse interface {
    GetType() MCPResponseType
    GetSuccess() bool
    IsTerminal() bool // Quan trọng: xác định có chấm dứt xử lý LLM tiếp theo hay không
    ToJSON() (string, error)
    GetContent() []mcp_go.Content
}
```

## 🔄 Chi tiết luồng xử lý

### 1. Thực thi gọi công cụ
```go
fcResult, err := tool.InvokableRun(toolCtx, toolCall.Function.Arguments)
```

### 2. Parse kết quả
```go
// Thử parse kết quả công cụ cục bộ
if mcpResp, ok := l.handleLocalToolResult(fcResult); ok {
    contentList = mcpResp.GetContent()
} else if toolCallResult, ok := l.handleToolResult(fcResult); ok {
    contentList = toolCallResult.Content
}
```

> `handleToolResult` **không còn yêu cầu giá trị trả về của công cụ bắt buộc phải là JSON**.  
> - Nếu trả về JSON `CallToolResult` MCP tiêu chuẩn, hệ thống sẽ parse theo nội dung có cấu trúc.  
> - Nếu trả về chuỗi thông thường, hệ thống sẽ tự động bọc thành `TextContent` để tiếp tục luồng xử lý.  
> Nhờ vậy công cụ text thông thường và công cụ MCP có cấu trúc đều có thể được xử lý thống nhất.

### 3. Xử lý kiểu nội dung
```go
for _, content := range contentList {
    switch content.(type) {
    case mcp_go.AudioContent:
        // Xử lý nội dung audio
    case mcp_go.ResourceLink:
        // Xử lý resource link
    case mcp_go.TextContent:
        // Xử lý nội dung text
    }
}
```

### 4. Điều khiển xử lý tiếp theo
```go
if invokeToolSuccess && !shouldStopLLMProcessing {
    l.DoLlmRequest(ctx, nil, l.einoTools, true)
}
```

## 📊 Bảng so sánh kiểu nội dung

| Kiểu nội dung | Tính kết thúc | Cách xử lý | Kịch bản sử dụng | Công cụ ví dụ |
|----------|--------|----------|----------|----------|
| **AudioContent** | Kết thúc | Phát trực tiếp | File audio nhỏ | play_music |
| **ResourceLink** | Kết thúc | Đọc phân trang + phát streaming | File lớn/streaming media | music_player |
| **TextContent** | Không kết thúc | Cộng dồn text | Truy vấn thông tin | get_datetime |
| **BlobResourceContents** | Kết thúc | Xử lý streaming | Dữ liệu audio stream | audio_stream |

## 🎯 Best practice

### 1. Khuyến nghị triển khai công cụ
- **Công cụ audio**: trả về `AudioContent` hoặc `ResourceLink`
- **Công cụ truy vấn**: trả về `TextContent`
- **Công cụ hành động**: dùng hệ thống response có cấu trúc

### 2. Tối ưu hiệu năng
- File lớn dùng `ResourceLink` để xử lý phân trang, hỗ trợ phát streaming
- File audio nhỏ dùng trực tiếp `AudioContent` để giảm overhead mạng
- Tránh nội dung text quá dài vì ảnh hưởng tốc độ response
- Dùng cơ chế Pipe để vừa đọc vừa phát, cải thiện trải nghiệm người dùng

### 3. Xử lý lỗi
- Dùng `MCPErrorResponse` để thống nhất định dạng lỗi
- Cung cấp mã lỗi và gợi ý có ý nghĩa
- Giữ tương thích ngược

## 🔧 Tham số cấu hình

### Cấu hình phân trang
- `McpReadResourcePageSize`: kích thước trang đọc tài nguyên, mặc định 100KB (100 * 1024)
- `McpReadResourceStreamDoneFlag`: cờ kết thúc stream, là `"[DONE]"`
- **Timeout đọc**: thời gian timeout mỗi lần đọc phân trang, mặc định 30 giây
- **Cơ chế retry**: lỗi timeout tự động retry, cách nhau 1 giây

### Cấu hình audio
- `OutputAudioFormat.SampleRate`: sample rate audio đầu ra
- `OutputAudioFormat.FrameDuration`: thời lượng frame audio đầu ra
- **Định dạng audio**: tự động nhận diện theo `resourceLink.MIMEType`

## 📝 Hướng dẫn mở rộng

### Thêm kiểu nội dung mới
1. Định nghĩa kiểu nội dung mới trong package `mcp_go`
2. Thêm logic xử lý kiểu trong `handleToolCallResponse`
3. Implement hàm xử lý tương ứng
4. Cập nhật tài liệu và test

### Tùy chỉnh kiểu response
1. Kế thừa `MCPResponseBase`
2. Implement interface `MCPResponse`
3. Thêm logic parse trong `ParseMCPResponse`
4. Cung cấp hàm khởi tạo tiện dụng

## 🎵 Repo độc lập MCP Audio Server

### Tổng quan

MCP Audio Server đã được tách thành repo độc lập. Khuyến nghị chạy và debug MCP Server loại audio thông qua dự án độc lập. Phần này chủ yếu mô tả cách nó tương thích protocol với service chính.

### Chức năng cốt lõi

#### 1. Công cụ phát nhạc
- **Tên công cụ**: `musicPlayer`
- **Chức năng**: tìm kiếm và phát nhạc
- **Trả về**: resource link audio kiểu `ResourceLink`

#### 2. Template tài nguyên audio
- **Định dạng URI**: `resource://read_from_http`
- **Chức năng**: hỗ trợ đọc dữ liệu audio phân trang, truyền tham số qua Arguments
- **Tham số**: url (URL nhạc thực tế), start (vị trí bắt đầu), end (vị trí kết thúc)
- **Trả về**: dữ liệu audio kiểu `BlobResourceContents`

### Đặc điểm quan trọng

- **Đọc phân trang**: hỗ trợ xử lý streaming cho file lớn
- **HTTP Range request**: lấy dữ liệu audio theo đoạn
- **Xử lý lỗi**: xử lý tình huống bất thường như status code 416
- **Retry timeout**: tự động retry lỗi timeout, cách nhau 1 giây
- **Hủy context**: hỗ trợ hủy đọc tài nguyên một cách mềm mại
- **Mã hóa Base64**: truyền tham số URL nhạc an toàn
- **Hỗ trợ nhiều transport**: hai kiểu transport `stdio` và HTTP
- **Phát realtime**: dùng cơ chế Pipe để vừa đọc vừa phát

### Cách dùng

```bash
# Lấy và vào repo độc lập
git clone https://github.com/hackers365/mcp_audio_server.git
cd mcp_audio_server

# Khởi động server
go run .

# Gọi công cụ
{
  "name": "musicPlayer",
  "arguments": {"query": "nhạc thư giãn"}
}
```

Dự án độc lập này minh họa cách xây dựng công cụ MCP hỗ trợ xử lý tài nguyên audio, có thể dùng làm template tham khảo để phát triển các công cụ audio khác. Hướng dẫn sử dụng đầy đủ hơn xem `doc/mcp_audio_example.md`.

---

*Tài liệu này phản ánh toàn bộ kiểu nội dung trả về khi gọi công cụ mà chương trình hiện hỗ trợ.*
