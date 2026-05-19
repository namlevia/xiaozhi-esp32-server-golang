package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"xiaozhi-esp32-server-golang/internal/domain/asr/funasr"
)

// readWavFile đọc file WAV và chuyển thành dữ liệu PCM []float32.
func readWavFile(filePath string) ([]float32, error) {
	// Mở file WAV
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Mở file WAV thất bại: %v", err)
	}
	defer file.Close()

	// Tạo decoder WAV
	wavDecoder := wav.NewDecoder(file)
	if !wavDecoder.IsValidFile() {
		return nil, fmt.Errorf("File WAV không hợp lệ")
	}

	// Đọc thông tin file WAV
	wavDecoder.ReadInfo()
	format := wavDecoder.Format()

	fmt.Printf("Định dạng WAV: sample rate=%dHz, số kênh=%d\n",
		int(format.SampleRate), format.NumChannels)

	// Đọc toàn bộ dữ liệu PCM
	var allPcmData []float32

	// Dùng frame 20ms làm buffer
	perFrameDuration := 20
	frameSize := int(format.SampleRate) * perFrameDuration / 1000
	audioBuf := &audio.IntBuffer{
		Format:         format,
		SourceBitDepth: 16,
		Data:           make([]int, frameSize*format.NumChannels),
	}

	fmt.Printf("Dùng kích thước frame: %d sample (%.1fms)\n", frameSize, float64(perFrameDuration))
	fmt.Println("Bắt đầu đọc dữ liệu WAV...")

	for {
		// Đọc dữ liệu WAV
		n, err := wavDecoder.PCMBuffer(audioBuf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Đọc dữ liệu WAV thất bại: %v", err)
		}

		// Chuyển dữ liệu int thành float32 trong khoảng -1.0 đến 1.0
		for i := 0; i < n; i++ {
			// Chuyển int thành float32 từ khoảng [-32768, 32767] sang [-1.0, 1.0]
			floatSample := float32(audioBuf.Data[i]) / 32767.0
			allPcmData = append(allPcmData, floatSample)
		}
	}

	fmt.Printf("Đọc file WAV thành công, tổng sample: %d, thời lượng: %.2f giây\n",
		len(allPcmData), float64(len(allPcmData))/float64(format.SampleRate))

	return allPcmData, nil
}

func main() {
	// Định nghĩa tham số command line
	var (
		host = flag.String("host", "192.168.208.214", "Địa chỉ IP server FunASR")
		port = flag.String("port", "10096", "Port server FunASR")
		mode = flag.String("mode", "offline", "Mode nhận diện (online/offline)")
		file = flag.String("file", "test.wav", "Đường dẫn file WAV cần nhận diện")
	)

	// Parse tham số command line
	flag.Parse()

	// Hiển thị hướng dẫn sử dụng
	if len(os.Args) < 2 {
		fmt.Println("Cách dùng: ./streaming_example [tùy chọn]")
		fmt.Println("Tùy chọn:")
		flag.PrintDefaults()
		fmt.Println("\nVí dụ:")
		fmt.Println("  ./streaming_example -host=192.168.1.100 -port=10095 -file=audio.wav")
		fmt.Println("  ./streaming_example -mode=online -file=test.wav")
		return
	}

	config := funasr.FunasrConfig{
		Host:          *host,
		Port:          *port,
		Mode:          *mode,
		SampleRate:    16000,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: 10,
		Timeout:       30,
		AutoEnd:       false,
	}

	// Tạo instance ASR bằng config
	asr, err := funasr.NewFunasr(config)
	if err != nil {
		fmt.Printf("Tạo instance ASR thất bại: %v\n", err)
		return
	}

	fmt.Printf("Server đích: %s:%s, mode: %s\n", config.Host, config.Port, config.Mode)

	// Dùng đường dẫn file audio từ tham số command line
	audioFilePath := *file

	// Kiểm tra file audio có tồn tại hay không
	if _, err := os.Stat(audioFilePath); os.IsNotExist(err) {
		fmt.Printf("File audio %s không tồn tại\n", audioFilePath)
		fmt.Println("Vui lòng cung cấp đường dẫn file audio hợp lệ")
		return
	}

	// Đọc file WAV và chuyển thành dữ liệu PCM
	pcmData, err := readWavFile(audioFilePath)
	if err != nil {
		fmt.Printf("Đọc file WAV thất bại: %v\n", err)
		return
	}

	// Thực hiện nhận diện
	result, err := asr.Process(pcmData)
	if err != nil {
		fmt.Printf("Nhận diện thất bại: %v\n", err)
		return
	}

	// Format và in kết quả
	fmt.Println("Kết quả nhận diện:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println(result)
	fmt.Println(strings.Repeat("-", 40))
}
