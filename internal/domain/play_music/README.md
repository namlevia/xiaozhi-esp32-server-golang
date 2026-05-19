# Chức năng phát nhạc

Module này cung cấp khả năng phát nhạc streaming từ URL, hỗ trợ tải file audio qua mạng và decode theo thời gian thực thành stream frame audio.

## Tính năng

- Hỗ trợ phát streaming từ URL.
- Chủ yếu hỗ trợ MP3, tự động decode thành frame audio Opus.
- Dùng audio decoder ổn định và hiệu quả.
- Hỗ trợ hủy và timeout qua `context`.
- Dùng HTTP connection pool để cải thiện hiệu năng mạng.
- Cho phép cấu hình frame duration và audio format.
- Cung cấp thống kê phát và theo dõi trạng thái.

## Bắt đầu nhanh

### 1. Sử dụng cơ bản

```go
package main

import (
    "context"
    "fmt"

    "xiaozhi-esp32-server-golang/internal/domain/play_music"
)

func main() {
    ctx := context.Background()

    audioChan, err := play_music.PlayMusicStream(ctx, "https://example.com/music.mp3", 16000, 20, "mp3")
    if err != nil {
        panic(err)
    }

    for audioFrame := range audioChan {
        fmt.Printf("Nhận frame audio: %d byte\n", len(audioFrame))
    }
}
```

### 2. Cấu hình tùy chỉnh

```go
config := play_music.MusicPlayerConfig{
    FrameDuration: 20,
    AudioFormat:   "mp3",
}

configMap := config.ToMap()
_ = configMap
```

### 3. Ví dụ có thống kê

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

audioChan, err := play_music.PlayMusicStream(ctx, musicURL, 16000, 20, "mp3")
if err != nil {
    return err
}

frameCount := 0
for frame := range audioChan {
    frameCount++
    _ = frame
}

fmt.Printf("Phát hoàn tất, tổng số frame: %d\n", frameCount)
```

## API tham khảo

### `PlayMusicStream`

Phát nhạc streaming từ URL.

Tham số:

- `ctx`: context dùng để hủy hoặc timeout.
- `url`: URL file nhạc.
- `sampleRate`: sample rate output mong muốn.
- `frameDuration`: thời lượng frame tính bằng mili giây, mặc định 20.
- `audioFormat`: định dạng audio, hiện hỗ trợ `mp3`.

Trả về:

- `chan []byte`: channel dữ liệu frame audio.
- `error`: thông tin lỗi.

### `PlayMusicFromAudioData`

Decode và phát từ buffer audio có sẵn.

### `PlayMusicFromPipe`

Decode và phát từ `io.PipeReader`, phù hợp cho nguồn audio streaming nội bộ.

## Kiểu cấu hình

```go
type MusicPlayerConfig struct {
    FrameDuration int    `json:"frame_duration"`
    AudioFormat   string `json:"audio_format"`
}
```

## Định dạng audio hỗ trợ

- MP3: hỗ trợ đầy đủ, khuyến nghị sử dụng.
- WAV: chỉ hỗ trợ một phần qua decoder chung nếu pipeline liên quan hỗ trợ.

## Xử lý lỗi

Player xử lý các lỗi chính:

1. Response HTTP không thành công.
2. Stream rỗng hoặc quá nhỏ để parse.
3. Audio format không được hỗ trợ.
4. Decode MP3 thất bại.
5. Context bị hủy trong lúc phát.

## Gợi ý tối ưu hiệu năng

1. Giữ `frame_duration` mặc định 20ms cho đa số trường hợp.
2. Dùng URL audio có mạng ổn định; HTTP connection pool đã được tối ưu sẵn.
3. Xử lý frame audio kịp thời để tránh channel bị nghẽn.
4. Tránh phát quá nhiều stream đồng thời nếu tài nguyên hạn chế.

## Tích hợp WebSocket

```go
for frame := range audioChan {
    if err := websocket.WriteMessage(websocket.BinaryMessage, frame); err != nil {
        log.Errorf("Gửi message WebSocket thất bại: %v", err)
        break
    }
}
```

## Lưu ý

1. Đảm bảo URL audio truy cập được và trả về file audio hợp lệ.
2. Với playback dài, cần chú ý memory usage.
3. Dùng kết nối mạng ổn định để có trải nghiệm phát tốt nhất.
4. Hủy context kịp thời với các tác vụ phát không còn cần thiết.
