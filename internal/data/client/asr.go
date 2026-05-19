package client

import (
	"bytes"
	"context"
	"strings"
	"sync"
	asr_types "xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

type Asr struct {
	lock sync.RWMutex
	// Context và channel ASR
	Ctx              context.Context
	Cancel           context.CancelFunc
	AsrEnd           chan bool
	AsrAudioChannel  chan []float32                 // channel input audio streaming
	AsrResultChannel chan asr_types.StreamingResult // fragment kết quả ASR nhận diện streaming
	AsrResult        bytes.Buffer                   // lưu text final nhận diện lần này
	Statue           int                            // 0: khởi tạo 1: đang nhận diện 2: nhận diện kết thúc
	AutoEnd          bool                           // auto_end nghĩa là dùng ASR tự xác định kết thúc, không dùng module VAD nữa

	// Loại và mode ASR
	AsrType string // Loại ASR, ví dụ "funasr", "doubao", "wyoming_vietnamese_asr"
	Mode    string // Mode ASR, ví dụ "online", "offline"

	// Tham chiếu ClientState, dùng cho callback thông báo
	ClientState *ClientState

	// Cache audio lịch sử chat: liên tục tích lũy dữ liệu audio gửi tới ASR
	HistoryAudioBuffer []float32

	// Lượt ASR hiện tại đã nhận text không rỗng đầu tiên hay chưa
	ReceivedTextInTurn bool
}

func (a *Asr) Reset() {
	a.AsrResult.Reset()
}

func (a *Asr) CancelWithReason(reason string) {
	a.lock.RLock()
	cancel := a.Cancel
	a.lock.RUnlock()

	if cancel != nil {
		log.Debugf("Asr.CancelWithReason: reason=%s", reason)
		cancel()
	}
}

func (a *Asr) RetireAsrResult(ctx context.Context) (asr_types.StreamingResult, bool, error) {
	defer func() {
		a.Reset()
	}()

	log.Log().Debugf("asr type: %s, mode: %s", a.AsrType, a.Mode)

	// Dùng biến local theo dõi đã gửi event ký tự đầu tiên hay chưa
	firstTextSent := false
	var emptyResult asr_types.StreamingResult

	for {
		select {
		case <-ctx.Done():
			log.Debugf("RetireAsrResult: ctx done, exit")
			return emptyResult, false, nil
		default:
			// Tránh trường hợp ctx cancel nhưng select tình cờ chọn channel, dẫn tới dùng kết quả của context đã cancel
			select {
			case result, ok := <-a.AsrResultChannel:
				log.Debugf("asr result: %s, ok: %+v, isFinal: %+v, emptyReason: %s, error: %+v", result.Text, ok, result.IsFinal, result.EmptyReason, result.Error)
				if result.Error != nil {
					if result.RetryReason != "" {
						log.Warnf("ASR trả lỗi recoverable(%s)，giao lớp trên recover: %v", result.RetryReason, result.Error)
						return result, true, nil
					}
					return emptyResult, false, result.Error
				}

				// Detect ký tự trả về đầu tiên (text không rỗng và chưa gửi)
				if result.Text != "" && !firstTextSent && a.ClientState != nil && a.ClientState.OnAsrFirstTextCallback != nil {
					firstTextSent = true
					// Gọi callback thông báo ký tự đầu tiên
					a.ClientState.OnAsrFirstTextCallback(result.Text, result.IsFinal)
				}

				if a.AsrType == "funasr" &&
					strings.EqualFold(a.Mode, "2pass") &&
					strings.EqualFold(result.Mode, "2pass-online") {
					if result.IsFinal {
						log.Debugf("funasr 2pass-online kết quả bị đánh dấu final nhầm, tiếp tục chờ kết quả final 2pass-offline")
					}
					continue
				}

				if result.IsFinal {
					return result, true, nil
				}

				if !ok {
					log.Debugf("asr result channel closed")
					return emptyResult, true, nil
				}
			}
		}
	}
}

func (a *Asr) MarkTextReceived() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.ReceivedTextInTurn = true
}

func (a *Asr) HasReceivedText() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.ReceivedTextInTurn
}

func (a *Asr) ResetReceivedText() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.ReceivedTextInTurn = false
}

func (a *Asr) StopWithReason(reason string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.AsrAudioChannel != nil {
		log.Debugf("Asr.StopWithReason: reason=%s", reason)
		close(a.AsrAudioChannel) // đóng channel input audio ASR, báo ASR dừng và trả kết quả
		a.AsrAudioChannel = nil  // vì đã close nên cần set nil
	}
}

func (a *Asr) Stop() {
	a.StopWithReason("Asr.Stop")
}

func (a *Asr) HasOpenAudioInput() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.AsrAudioChannel != nil
}

func (a *Asr) AddAudioData(pcmFrameData []float32) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.AsrAudioChannel != nil {
		// Dùng select để gửi non-blocking, tránh deadlock khi channel đầy
		select {
		case a.AsrAudioChannel <- pcmFrameData:
			// Gửi thành công, cache đồng bộ dữ liệu audio cho lịch sử chat
			a.HistoryAudioBuffer = append(a.HistoryAudioBuffer, pcmFrameData...)
		default:
			// channel đã đầy, bỏ dữ liệu lần này để tránh block gây deadlock
			log.Warnf("AsrAudioChannel đã đầy, bỏ dữ liệu audio lần này")
		}
	}
	return nil
}

// GetHistoryAudio lấy cache audio lịch sử (trả bản sao, không xóa dữ liệu gốc)
func (a *Asr) GetHistoryAudio() []float32 {
	a.lock.Lock()
	defer a.lock.Unlock()
	if len(a.HistoryAudioBuffer) == 0 {
		return nil
	}
	// Trả bản sao để tránh bên ngoài sửa ảnh hưởng dữ liệu gốc
	result := make([]float32, len(a.HistoryAudioBuffer))
	copy(result, a.HistoryAudioBuffer)
	return result
}

// GetHistoryAudioLen lấy độ dài cache audio lịch sử (số sample)
func (a *Asr) GetHistoryAudioLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.HistoryAudioBuffer)
}

// ClearHistoryAudio xóa cache audio lịch sử
func (a *Asr) ClearHistoryAudio() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.HistoryAudioBuffer = nil
}

type AsrAudioBuffer struct {
	PcmData          []float32
	AudioBufferMutex sync.RWMutex
}

func (a *AsrAudioBuffer) AddAsrAudioData(pcmFrameData []float32) error {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	a.PcmData = append(a.PcmData, pcmFrameData...)
	return nil
}

func (a *AsrAudioBuffer) GetAsrDataSize() int {
	a.AudioBufferMutex.RLock()
	defer a.AudioBufferMutex.RUnlock()
	return len(a.PcmData)
}

// GetFrameCount lấy số frame (cần truyền kích thước frame để tính)
func (a *AsrAudioBuffer) GetFrameCount(frameSize int) int {
	a.AudioBufferMutex.RLock()
	defer a.AudioBufferMutex.RUnlock()
	if frameSize == 0 {
		return 0
	}
	return len(a.PcmData) / frameSize
}

func (a *AsrAudioBuffer) GetAndClearAllData() []float32 {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	pcmData := make([]float32, len(a.PcmData))
	copy(pcmData, a.PcmData)
	a.PcmData = []float32{}
	return pcmData
}

// GetAsrData lấy dữ liệu bằng sliding window (cần truyền kích thước frame để tính)
func (a *AsrAudioBuffer) GetAsrData(frameCount int, frameSize int) []float32 {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	pcmDataLen := len(a.PcmData)
	retSize := frameCount * frameSize
	if pcmDataLen < retSize {
		retSize = pcmDataLen
	}
	pcmData := make([]float32, retSize)
	copy(pcmData, a.PcmData[pcmDataLen-retSize:])
	return pcmData
}

// RemoveAsrAudioData xóa dữ liệu audio theo số frame chỉ định (cần truyền kích thước frame để tính)
func (a *AsrAudioBuffer) RemoveAsrAudioData(frameCount int, frameSize int) {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	removeSize := frameCount * frameSize
	if removeSize > len(a.PcmData) {
		removeSize = len(a.PcmData)
	}
	a.PcmData = a.PcmData[removeSize:]
}

func (a *AsrAudioBuffer) ClearAsrAudioData() {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	a.PcmData = nil
}
