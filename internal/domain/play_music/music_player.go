package play_music

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"bytes"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

// HTTP client global, triển khai connection pool
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// Lấy HTTP client đã cấu hình connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			//Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// PlayMusicStream phát nhạc từ URL, trả về channel stream audio
// frameDuration: thời lượng mỗi frame (mili giây), mặc định 20ms
// audioFormat: định dạng audio, hỗ trợ "mp3"
func PlayMusicStream(ctx context.Context, url string, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	// Kiểm tra tham số và thiết lập giá trị mặc định
	if frameDuration <= 0 {
		frameDuration = 20 // thời lượng frame mặc định 20ms
	}
	if audioFormat == "" {
		audioFormat = "mp3" // định dạng MP3 mặc định
	}

	startTs := time.Now().UnixMilli()

	// Tạo HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Tạo request thất bại: %v", err)
	}

	req.Header.Set("Accept", "audio/*")
	req.Header.Set("User-Agent", "MusicPlayer/1.0")

	// Tạo client bằng connection pool
	client := getHTTPClient()

	// Tạo output channel
	outputChan = make(chan []byte, 100)

	// Khởi động goroutine xử lý response streaming
	go func() {
		// Gửi request
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Gửi request thất bại: %v", err)
			close(outputChan)
			return
		}
		defer func() {
			resp.Body.Close()
		}()

		// Kiểm tra status code của response
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("API request thất bại, status code: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		// Kiểm tra content type và content length của response
		contentLength := resp.ContentLength

		// Ghi độ dài response vào log
		log.Debugf("Nhận response stream nhạc, Content-Length: %d", contentLength)

		// Kiểm tra Content-Length có hợp lý không
		if contentLength == 0 {
			log.Errorf("Stream nhạc trả response rỗng, Content-Length bằng 0")
			close(outputChan)
			return
		}

		// Header file MP3 cần ít nhất 100 byte để parse bình thường
		// -1 biểu thị độ dài không xác định (ví dụ chunked transfer)
		if contentLength > 0 && contentLength < 100 {
			log.Errorf("Response stream nhạc quá nhỏ để parse thành MP3: %d byte", contentLength)
			close(outputChan)
			return
		}

		log.Infof("Bắt đầu phát nhạc: %s", url)

		// Xử lý response streaming theo định dạng audio
		if audioFormat == "mp3" {
			// Tạo MP3 decoder, truyền context thay vì done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, resp.Body, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Tạo MP3 decoder thất bại: %v", err)
				close(outputChan)
				return
			}

			// Khởi động quá trình decode
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("Decode MP3 thất bại: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Phát nhạc bị hủy, URL: %s", url)
				return
			default:
				log.Infof("Phát nhạc hoàn tất, thời gian: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Hiện chỉ hỗ trợ phát streaming định dạng MP3, định dạng truyền vào: %s", audioFormat)
			close(outputChan)
		}
	}()

	return outputChan, nil
}

func PlayMusicFromAudioData(ctx context.Context, audioData []byte, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	// Kiểm tra tham số và thiết lập giá trị mặc định
	if frameDuration <= 0 {
		frameDuration = 20 // thời lượng frame mặc định 20ms
	}
	if audioFormat == "" {
		audioFormat = "mp3" // định dạng MP3 mặc định
	}

	// Thêm thông tin debug
	log.Debugf("PlayMusicFromAudioData: độ dài audio data=%d byte, sample rate=%d, frame duration=%dms, format=%s",
		len(audioData), sampleRate, frameDuration, audioFormat)

	// Kiểm tra audio data có rỗng không
	if len(audioData) == 0 {
		log.Errorf("Audio data rỗng, không thể phát")
		return nil, fmt.Errorf("audio data rỗng")
	}

	startTs := time.Now().UnixMilli()

	// Tạo output channel
	outputChan = make(chan []byte, 100)

	// Khởi động goroutine xử lý response streaming
	go func() {
		// Tạo io.ReadCloser từ audioData
		audioReader := io.NopCloser(bytes.NewReader(audioData))

		// Xử lý response streaming theo định dạng audio
		if audioFormat == "mp3" {
			// Tạo MP3 decoder, truyền context thay vì done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, audioReader, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Tạo MP3 decoder thất bại: %v", err)
				return
			}

			// Khởi động quá trình decode
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("Decode MP3 thất bại: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Phát nhạc bị hủy")
				return
			default:
				log.Infof("Phát nhạc hoàn tất, thời gian: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Hiện chỉ hỗ trợ phát streaming định dạng MP3, định dạng truyền vào: %s", audioFormat)
		}
	}()

	return outputChan, nil
}

func PlayMusicFromPipe(ctx context.Context, pipeReader *io.PipeReader, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	// Kiểm tra tham số và thiết lập giá trị mặc định
	if frameDuration <= 0 {
		frameDuration = 20 // thời lượng frame mặc định 20ms
	}
	if audioFormat == "" {
		audioFormat = "mp3" // định dạng MP3 mặc định
	}

	// Thêm thông tin debug
	log.Debugf("PlayMusicFromPipe: sample rate=%d, frame duration=%dms, format=%s",
		sampleRate, frameDuration, audioFormat)

	startTs := time.Now().UnixMilli()

	// Tạo output channel
	outputChan = make(chan []byte, 100)

	// Khởi động goroutine xử lý response streaming
	go func() {
		// Xử lý response streaming theo định dạng audio
		if audioFormat == "mp3" {
			// Tạo MP3 decoder, truyền context thay vì done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Tạo MP3 decoder thất bại: %v", err)
				return
			}

			// Khởi động quá trình decode
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("Decode MP3 thất bại: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Phát nhạc bị hủy")
				return
			default:
				log.Infof("Phát nhạc hoàn tất, thời gian: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Hiện chỉ hỗ trợ phát streaming định dạng MP3, định dạng truyền vào: %s", audioFormat)
		}
	}()

	return outputChan, nil
}
