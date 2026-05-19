package main

import (
	"fmt"
	"os"

	"github.com/hraban/opus"
)

func main() {
	// Thiết lập tham số audio.
	channels := 1
	sampleRate := 16000 // 16kHz
	fmt.Printf("Số kênh: %d, sample rate: %d Hz\n", channels, sampleRate)

	// Tạo encoder, dùng loại ứng dụng VoIP cho giọng nói độ trễ thấp.
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		fmt.Printf("Tạo encoder thất bại: %v\n", err)
		os.Exit(1)
	}

	// Đặt bitrate 16kbps.
	if err = enc.SetBitrate(16000); err != nil {
		fmt.Printf("Đặt bitrate thất bại: %v\n", err)
		os.Exit(1)
	}

	// Đặt độ phức tạp trong khoảng 0-10, càng cao chất lượng càng tốt nhưng tốn CPU hơn.
	if err = enc.SetComplexity(5); err != nil {
		fmt.Printf("Đặt độ phức tạp thất bại: %v\n", err)
		os.Exit(1)
	}

	// Tạo dữ liệu PCM kiểm thử 20ms, với sample rate 16kHz tương ứng 320 mẫu mỗi frame.
	frameSize := 320
	pcm := make([]int16, frameSize*channels)

	// Tạo sóng sin đơn giản để kiểm thử.
	for i := 0; i < frameSize; i++ {
		// Sóng sin đơn giản, tần số khoảng 440Hz.
		value := int16(10000.0 * float64(i%36) / 36.0)
		pcm[i] = value
	}

	// Bộ nhớ chứa dữ liệu sau khi encode.
	data := make([]byte, 1000)

	// Encode dữ liệu PCM thành Opus.
	n, err := enc.Encode(pcm, data)
	if err != nil {
		fmt.Printf("Encode thất bại: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Đã encode %d mẫu thành %d byte dữ liệu Opus, tỷ lệ nén: %.2f%%\n",
		frameSize*channels, n, float64(n)/float64(frameSize*channels*2)*100)

	// Tạo decoder để kiểm thử decode.
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		fmt.Printf("Tạo decoder thất bại: %v\n", err)
		os.Exit(1)
	}

	// Bộ nhớ chứa dữ liệu PCM sau khi decode.
	decodedPCM := make([]int16, frameSize*channels)

	// Decode dữ liệu Opus về PCM.
	samplesDecoded, err := dec.Decode(data[:n], decodedPCM)
	if err != nil {
		fmt.Printf("Decode thất bại: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Đã decode %d byte dữ liệu Opus thành %d mẫu\n", n, samplesDecoded)

	// Tính sai khác giữa PCM gốc và PCM sau khi decode.
	var sumDiff int64
	for i := 0; i < frameSize; i++ {
		diff := int64(pcm[i]) - int64(decodedPCM[i])
		if diff < 0 {
			diff = -diff
		}
		sumDiff += diff
	}
	avgDiff := float64(sumDiff) / float64(frameSize)

	fmt.Printf("Sai khác trung bình giữa PCM gốc và PCM decode: %.2f\n", avgDiff)
	fmt.Println("Ví dụ encode/decode Opus đã hoàn tất!")
}
