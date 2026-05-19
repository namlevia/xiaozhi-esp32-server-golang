package chat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/asr"
	asr_types "xiaozhi-esp32-server-golang/internal/domain/asr/types"
	"xiaozhi-esp32-server-golang/internal/domain/audio"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/domain/vad/inter"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

type ASRManagerOption func(*ASRManager)

const maxFirstSpeechPreAudioMs = 200

// AsrMessageSaveCallback kiểu callback lưu message
type AsrMessageSaveCallback func(userMsg *schema.Message, messageID string, audioData []float32)

type ASRManager struct {
	clientState     *ClientState
	serverTransport *ServerTransport
	session         *ChatSession // dùng để truy cập speakerManager

	// Quản lý resource ASR bằng private field
	asrResource *pool.ResourceWrapper[asr.AsrProvider]
	resourceMu  sync.RWMutex // bảo vệ truy cập resource
}

func NewASRManager(clientState *ClientState, serverTransport *ServerTransport, opts ...ASRManagerOption) *ASRManager {
	asr := &ASRManager{
		clientState:     clientState,
		serverTransport: serverTransport,
		session:         nil, // set sau qua SetSession
	}
	for _, opt := range opts {
		opt(asr)
	}
	return asr
}

func (a *ASRManager) runAudioIdleTimeoutWatchdog(ctx context.Context) {
	state := a.clientState
	if state == nil {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !state.UsesAudioIdleClock() || !state.AudioIdleStarted() || state.AudioIdlePaused() {
				continue
			}
			if !state.ShouldCountAudioIdleTimeout() || state.Asr.HasReceivedText() {
				continue
			}
			if state.GetClientVoiceStop() || state.AudioIdleTimeoutPending() {
				continue
			}

			elapsed := state.GetAudioIdleElapsed(time.Now())
			threshold := time.Duration(state.GetMaxIdleDuration()) * time.Millisecond
			if elapsed < threshold {
				continue
			}
			if !state.MarkAudioIdleTimeoutPending() {
				continue
			}

			if !state.Asr.HasOpenAudioInput() {
				log.Infof(
					"Audio idle timeout, hiện không có ASR stream active, đóng session trực tiếp: device=%s, mode=%s, elapsed=%dms, threshold=%dms",
					state.DeviceID,
					state.ListenMode,
					elapsed.Milliseconds(),
					state.GetMaxIdleDuration(),
				)
				if a.session != nil {
					a.session.CloseWithReason(chatSessionCloseReasonAudioIdleTimeout)
				} else {
					state.ClearAudioIdleTimeoutPending()
				}
				continue
			}

			log.Infof(
				"Audio idle timeout, kích hoạt chốt ASR: device=%s, mode=%s, elapsed=%dms, threshold=%dms",
				state.DeviceID,
				state.ListenMode,
				elapsed.Milliseconds(),
				state.GetMaxIdleDuration(),
			)
			state.OnVoiceSilence()
		}
	}
}

// ProcessVadAudio khởi động xử lý audio VAD
func (a *ASRManager) ProcessVadAudio(ctx context.Context) {
	state := a.clientState
	go func() {
		hasTriggeredCancel := true // cờ ghi nhận đã trigger thao tác cancel hay chưa (khi voiceDuration > 120)
		hasLoggedFirstTextExtendedWait := false
		speakerInterruptTriggered := atomic.Bool{}
		speakerPeekInFlight := atomic.Bool{}
		lastSpeakerPeekDoneAt := atomic.Int64{}
		var speakerPeekAudioMs int64
		var speakerPeekRequestSeq uint64
		const speakerPeekInterval = 200 * time.Millisecond
		const firstSpeakerPeekAudioThresholdMs int64 = 400
		audioFormat := state.InputAudioFormat
		// Dùng buffer đủ lớn để decode (giả định frame dài tối đa 120ms)
		maxFrameSize := audioFormat.SampleRate * audioFormat.Channels * 120 / 1000
		audioProcesser, err := audio.GetAudioProcesser(audioFormat.SampleRate, audioFormat.Channels, 20) // truyền giá trị mặc định để tạo decoder
		if err != nil {
			log.Errorf("Lấy decoder thất bại: %v", err)
			return
		}

		// Lấy kích thước frame và duration từ dữ liệu thực tế của frame đầu
		var frameSize int
		var frameDurationMs int
		var vadNeedGetCount int // số frame VAD cần, tính sau frame đầu

		// Resource VAD chuyển sang lazy load + release khi idle, tránh chiếm instance pool quá lâu.
		var vadWrapper *pool.ResourceWrapper[inter.VAD]
		var vadProvider inter.VAD
		var vadLastUseAt time.Time
		const vadIdleReleaseTimeout = 2 * time.Second
		vadIdleTicker := time.NewTicker(time.Second)
		defer vadIdleTicker.Stop()
		needVad := !(state.Asr.AutoEnd || state.ListenMode == "manual")
		vadProviderName := state.DeviceConfig.Vad.Provider
		vadProviderConfig := state.DeviceConfig.Vad.Config
		releaseVad := func(reason string) {
			if vadWrapper == nil {
				return
			}
			pool.Release(vadWrapper)
			vadWrapper = nil
			vadProvider = nil
			vadLastUseAt = time.Time{}
			log.Debugf("Release resource VAD: device=%s, reason=%s", state.DeviceID, reason)
		}
		defer releaseVad("process_exit")
		ensureVad := func() bool {
			if !needVad {
				return false
			}
			if vadProvider != nil {
				return true
			}

			// Kiểm tra provider có rỗng không; nếu rỗng thì ghi warn.
			if vadProviderName == "" {
				log.Warnf("VAD provider rỗng, thử lấy từ config")
			} else {
				log.Debugf("Lấy resource VAD: provider=%s", vadProviderName)
			}

			wrapper, err := pool.Acquire[inter.VAD](
				"vad",
				vadProviderName,
				vadProviderConfig,
			)
			if err != nil {
				log.Errorf("Lấy resource VAD thất bại: provider=%s, config=%+v, error=%v", vadProviderName, vadProviderConfig, err)
				return false
			}
			vadWrapper = wrapper
			vadProvider = wrapper.GetProvider()
			vadLastUseAt = time.Now()
			return true
		}
		for {
			// Dùng kích thước frame tối đa làm buffer, sau decode sẽ có kích thước frame thực tế
			pcmFrame := make([]float32, maxFrameSize)

			select {
			case <-vadIdleTicker.C:
				if vadWrapper != nil && !vadLastUseAt.IsZero() && time.Since(vadLastUseAt) >= vadIdleReleaseTimeout {
					releaseVad("idle_timeout")
				}
				continue
			case opusFrame, ok := <-state.OpusAudioBuffer:
				//log.Debugf("processAsrAudio nhận dữ liệu audio, len: %d", len(opusFrame))
				if !ok {
					log.Debugf("processAsrAudio channel audio đã đóng")
					return
				}

				var skipVad bool
				var haveVoice bool
				clientHaveVoice := state.GetClientHaveVoice()
				if state.ListenMode == "manual" {
					skipVad = true         //bỏ qua VAD
					clientHaveVoice = true //trước đó có âm thanh
					haveVoice = true       //lần này có âm thanh
				} else if state.Asr.AutoEnd {
					skipVad = true   // vẫn để provider kiểm soát stop, nhưng không đổi ngữ nghĩa idle
					haveVoice = true // audio lần này đi thẳng vào ASR
				}

				if state.GetClientVoiceStop() { //đã dừng nói thì không nhận dữ liệu audio
					//log.Infof("Client đã dừng nói, bỏ qua dữ liệu audio")
					continue
				}

				//log.Debugf("clientVoiceStop: %+v, asrDataSize: %d, listenMode: %s, isSkipVad: %v\n", state.GetClientVoiceStop(), state.AsrAudioBuffer.GetAsrDataSize(), state.ListenMode, skipVad)

				n, err := audioProcesser.DecoderFloat32(opusFrame, pcmFrame)
				if err != nil {
					log.Errorf("Decode thất bại: %v", err)
					continue
				}

				// Tính động kích thước frame và duration từ dữ liệu đã decode thực tế
				if frameSize == 0 {
					// Frame đầu: tính thông tin frame từ dữ liệu decode thực tế
					frameSize = n
					samplesPerChannel := n / audioFormat.Channels
					frameDurationMs = samplesPerChannel * 1000 / audioFormat.SampleRate
					audioFormat.FrameDuration = frameDurationMs

					// Tính số frame VAD cần
					vadNeedGetCount = 1
					if state.DeviceConfig.Vad.Provider == "silero_vad" {
						// silero_vad cần ít nhất 60ms dữ liệu audio
						vadNeedGetCount = 60 / frameDurationMs
						if vadNeedGetCount < 1 {
							vadNeedGetCount = 1
						}
					}
					log.Debugf("Tính thông tin frame từ dữ liệu audio thực tế: frameSize=%d, frameDurationMs=%d, vadNeedGetCount=%d", frameSize, frameDurationMs, vadNeedGetCount)
				}

				var vadPcmData []float32
				pcmData := pcmFrame[:n]
				speakerPcmData := pcmFrame[:n]

				// Kiểm tra kích thước frame có nhất quán không (bình thường phải nhất quán; nếu không thì dùng giá trị thực tế)
				if n != frameSize {
					log.Debugf("Kích thước frame không nhất quán: mong đợi=%d, thực tế=%d, dùng giá trị thực tế", frameSize, n)
					// Tính lại duration của frame này
					samplesPerChannel := n / audioFormat.Channels
					currentFrameDurationMs := samplesPerChannel * 1000 / audioFormat.SampleRate
					frameSize = n
					frameDurationMs = currentFrameDurationMs
					audioFormat.FrameDuration = frameDurationMs
				}

				if !skipVad && needVad {
					if !ensureVad() {
						continue
					}
					//decode opus to pcm
					state.AsrAudioBuffer.AddAsrAudioData(pcmData)

					// Tính lượng dữ liệu tối thiểu VAD cần（60ms for silero_vad）
					vadNeedMinSize := frameSize
					if state.DeviceConfig.Vad.Provider == "silero_vad" {
						vadNeedMinSize = vadNeedGetCount * frameSize
					}

					if state.AsrAudioBuffer.GetAsrDataSize() >= vadNeedMinSize {
						//Nếu chạy VAD, cần lấy ít nhất 60ms dữ liệu audio
						vadPcmData = state.AsrAudioBuffer.GetAsrData(vadNeedGetCount, frameSize)

						//Nếu đã phát hiện giọng nói, không chạy VAD nữa, truyền thẳng pcmData cho ASR
						// Dùng resource VAD lấy ngoài vòng lặp để detect
						// Reset trạng thái VAD
						vadLastUseAt = time.Now()
						if err := vadProvider.Reset(); err != nil {
							log.Errorf("Reset VAD thất bại: %v", err)
							continue
						}

						// Chạy detect VAD
						vadLastUseAt = time.Now()
						haveVoice, err = vadProvider.IsVADExt(vadPcmData, audioFormat.SampleRate, frameSize)
						if err != nil {
							log.Errorf("processAsrAudio detect VAD thất bại: %v", err)
							continue
						}

						//Khi lần đầu detect có giọng nói, gán vadPcmData cho pcmData để giữ đủ dữ liệu giọng nói; các audio sau đó đều vào ASR
						if haveVoice && !clientHaveVoice {
							//Khi detect giọng nói lần đầu, chỉ giữ tối đa 200ms silence phía trước
							currentFrameSamples := len(pcmData)
							allData := state.AsrAudioBuffer.GetAndClearAllData()
							pcmData = trimFirstSpeechAudio(allData, currentFrameSamples, audioFormat.SampleRate, audioFormat.Channels)
						}
					}
					//log.Debugf("isVad, pcmData len: %d, vadPcmData len: %d, haveVoice: %v", len(pcmData), len(vadPcmData), haveVoice)
				}

				if haveVoice {
					hasLoggedFirstTextExtendedWait = false
					//log.Infof("Detect có giọng nói, len: %d", len(pcmData))
					state.SetClientHaveVoice(true)
					state.SetClientHaveVoiceLastTime(time.Now().UnixMilli())
					state.Vad.ResetIdleDuration()
					// Tích lũy duration detect có âm thanh (đồng thời cập nhật duration trong quá trình)
					state.Vad.AddVoiceDuration(int64(frameDurationMs))

					continuousVoiceDuration := state.Vad.GetVoiceContinuousDuration()
					if state.IsRealTime() && viper.GetInt("chat.realtime_mode") == 1 && continuousVoiceDuration > 360 {
						// Chỉ thực thi khi chưa trigger, đảm bảo chỉ chạy một lần
						if !hasTriggeredCancel {
							if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
								log.Debugf("Thiết bị %s realtime media gate active, bỏ qua interrupt VAD", state.DeviceID)
								hasTriggeredCancel = true
							} else {
								// Ở mode realtime, nếu lúc này đang có LLM và TTS thì cancel.
								log.Debugf("mode realtime VAD interrupt && voice duration vượt %d ms, nếu đang có LLM và TTS thì cancel", continuousVoiceDuration)
								if a.session != nil {
									a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.ProcessVadAudio realtime_mode=1 VAD interrupt")
								} else {
									state.AfterAsrSessionCtx.CancelWithReason("ASRManager.ProcessVadAudio: realtime_mode=1 VAD interrupt")
								}
								hasTriggeredCancel = true // đánh dấu đã trigger
							}
						}
					}
				} else {
					state.Vad.AddIdleDuration(int64(frameDurationMs))
					state.Vad.ResetVoiceContinuousDuration()

					// Khi không có âm thanh, nếu trước đó cũng không có giọng nói thì reset duration tích lũy
					// Nếu trước đó có giọng nói nhưng lần này không có, giữ duration để logic sau quyết định có reset không
					if !clientHaveVoice {
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						//giữ gần 10 frame
						/*
							if state.AsrAudioBuffer.GetFrameCount(frameSize) > vadNeedGetCount*3 {
								state.AsrAudioBuffer.RemoveAsrAudioData(1, frameSize)
							}*/
						continue
					}
				}

				if clientHaveVoice || haveVoice {
					// Khi lần đầu hit giọng nói cũng cần forward ngay frame cache hiện tại, tránh toàn bộ đoạn nói rất ngắn không được đưa vào ASR.

					//VAD detect thành công, gửi dữ liệu vào channel audio ASR
					//log.Infof("VAD detect thành công, gửi dữ liệu vào channel audio ASR, len: %d", len(pcmData))
					state.Asr.AddAudioData(pcmData)

					// Voiceprint chỉ nhận các frame hiện được xác định là có âm, tránh đưa silence đầu/cuối vào stream nhận diện.
					if haveVoice &&
						state.IsSpeakerEnabled() && state.HasSpeakerGroups() &&
						a.session != nil && a.session.speakerManager != nil {
						// Khi detect giọng nói lần đầu, khởi động nhận diện streaming.
						if !a.session.speakerManager.IsActive() {
							sampleRate := audioFormat.SampleRate
							agentId := a.session.clientState.AgentID
							if err := a.session.speakerManager.StartStreaming(ctx, sampleRate, agentId); err != nil {
								log.Warnf("Khởi động stream nhận diện voiceprint thất bại: %v", err)
							} else {
								speakerInterruptTriggered.Store(false)
								lastSpeakerPeekDoneAt.Store(0)
								speakerPeekAudioMs = 0
							}
						}

						// Gửi audio chunk
						if err := a.session.speakerManager.SendAudioChunk(ctx, speakerPcmData); err != nil {
							log.Warnf("Gửi audio chunk tới dịch vụ nhận diện voiceprint thất bại: %v", err)
						} else if a.session.speakerManager.IsActive() {
							if audioFormat.Channels > 0 && audioFormat.SampleRate > 0 {
								speakerPeekAudioMs += int64(len(speakerPcmData)/audioFormat.Channels) * 1000 / int64(audioFormat.SampleRate)
							}

							if state.IsRealTime() &&
								viper.GetInt("chat.realtime_mode") == 3 &&
								!speakerInterruptTriggered.Load() &&
								speakerPeekAudioMs >= firstSpeakerPeekAudioThresholdMs {
								now := time.Now()
								lastDoneAt := lastSpeakerPeekDoneAt.Load()
								if (lastDoneAt <= 0 || now.Sub(time.Unix(0, lastDoneAt)) >= speakerPeekInterval) &&
									speakerPeekInFlight.CompareAndSwap(false, true) {
									reqSeq := atomic.AddUint64(&speakerPeekRequestSeq, 1)
									requestID := fmt.Sprintf("peek_%d_%d", now.UnixMilli(), reqSeq)

									go func(reqID string) {
										defer func() {
											lastSpeakerPeekDoneAt.Store(time.Now().UnixNano())
											speakerPeekInFlight.Store(false)
										}()

										if a.session == nil || a.session.speakerManager == nil || !a.session.speakerManager.IsActive() {
											return
										}

										peekCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
										defer cancel()

										peekResult, throttled, err := a.session.speakerManager.PeekAndIdentify(peekCtx, reqID)
										if err != nil {
											if ctx.Err() == nil {
												log.Debugf("speaker peek thất bại: device=%s, request_id=%s, err=%v", state.DeviceID, reqID, err)
											}
											return
										}
										if throttled {
											return
										}
										if peekResult == nil || !peekResult.Identified {
											return
										}
										if !speakerInterruptTriggered.CompareAndSwap(false, true) {
											return
										}

										log.Infof(
											"mode realtime speaker peek hit, interrupt ngay: device=%s, speaker=%s, confidence=%.4f, threshold=%.4f",
											state.DeviceID,
											peekResult.SpeakerName,
											peekResult.Confidence,
											peekResult.Threshold,
										)
										if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
											log.Debugf("Thiết bị %s realtime media gate active, bỏ qua interrupt speaker peek", state.DeviceID)
											return
										}
										a.session.MarkTurnSpeakerInterrupted()
										if a.session != nil {
											a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.ProcessVadAudio realtime_mode=3 speaker peek interrupt")
										} else {
											state.AfterAsrSessionCtx.CancelWithReason("ASRManager.ProcessVadAudio: realtime_mode=3 speaker peek interrupt")
										}
									}(requestID)
								}
							}
						}
					}
				}

				// Đã có giọng nói nhưng lần này không detect thấy giọng nói, cần xét đã dừng nói chưa.
				lastHaveVoiceTime := state.GetClientHaveVoiceLastTime()

				if clientHaveVoice && lastHaveVoiceTime > 0 && !haveVoice {
					// Xét duration giọng nói có audio; nếu nhỏ hơn 300ms thì reset clientHaveVoice để tránh nhận nhầm do âm thanh quá ngắn
					voiceDurationInSession := state.Vad.GetVoiceDurationInSession()
					if voiceDurationInSession < 100 {
						log.Debugf("Voice duration quá ngắn (%dms < 300ms), reset clientHaveVoice", voiceDurationInSession)
						state.SetClientHaveVoice(false)
						state.Vad.ResetVoiceDuration()
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						continue
					}

					idleDuration := state.Vad.GetIdleDuration()
					if state.IsRealTime() && !state.Asr.HasReceivedText() {
						preTextSilenceDuration := state.GetPreAsrTextSilenceDuration()
						if idleDuration <= preTextSilenceDuration {
							log.Debugf(
								"mode realtime chưa nhận text ASR đầu tiên, trì hoãn chốt theo ngưỡng silence: status=%s, idle=%dms, pre_text_timeout=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d",
								state.Status,
								idleDuration,
								preTextSilenceDuration,
								state.Vad.GetVoiceDuration(),
								voiceDurationInSession,
								state.Asr.GetHistoryAudioLen(),
							)
							continue
						}

						if !hasLoggedFirstTextExtendedWait {
							log.Debugf(
								"mode realtime silence timeout nhưng vẫn chưa nhận text ASR, tiếp tục giữ ASR stream hiện tại và forward audio: status=%s, idle=%dms, pre_text_timeout=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d",
								state.Status,
								idleDuration,
								preTextSilenceDuration,
								state.Vad.GetVoiceDuration(),
								voiceDurationInSession,
								state.Asr.GetHistoryAudioLen(),
							)
							hasLoggedFirstTextExtendedWait = true
						}
						continue
					}

					if state.IsSilence(idleDuration) { //xét chuyển từ có âm thanh sang silence
						log.Debugf(
							"Xác định giọng nói kết thúc, chuẩn bị dừng ASR: status=%s, idle=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d, pending_restart=%v",
							state.Status,
							idleDuration,
							state.Vad.GetVoiceDuration(),
							state.Vad.GetVoiceDurationInSession(),
							state.Asr.GetHistoryAudioLen(),
							state.AudioIdleTimeoutPending(),
						)
						// Reset cờ trước OnVoiceSilence để lần sau có thể trigger lại
						hasTriggeredCancel = false
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						state.OnVoiceSilence()
						//state.VoiceStatus.Reset()
						continue
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}

// releaseResource release resource ASR (method nội bộ)
func (a *ASRManager) releaseResource() {
	a.resourceMu.Lock()
	defer a.resourceMu.Unlock()
	if a.asrResource != nil {
		pool.Release(a.asrResource)
		a.asrResource = nil
		log.Debugf("Resource ASR đã trả lại")
	}
}

// Cleanup dọn resource ASR (cho bên ngoài gọi)
func (a *ASRManager) Cleanup() {
	a.releaseResource()
}

// restartAsrRecognition restart nhận diện ASR
func (a *ASRManager) RestartAsrRecognition(ctx context.Context) error {
	state := a.clientState
	log.Debugf("Bắt đầu restart nhận diện ASR")
	if a.session != nil {
		a.session.ResetTurnSpeakerInterrupted()
	}

	// Cancel context ASR hiện tại
	state.Asr.CancelWithReason("ASRManager.RestartAsrRecognition: cancel previous ASR context before restart")

	state.Asr.ResetReceivedText()
	state.VoiceStatus.Reset()
	state.AsrAudioBuffer.ClearAsrAudioData()
	state.Asr.ClearHistoryAudio() // Xóa cache audio lịch sử

	// Kiểm tra đã có resource chưa; nếu chưa thì lấy
	a.resourceMu.Lock()
	var asrProvider asr.AsrProvider
	if a.asrResource == nil {
		// Cần lấy resource mới
		a.resourceMu.Unlock()

		asrWrapper, err := pool.Acquire[asr.AsrProvider](
			"asr",
			state.DeviceConfig.Asr.Provider,
			state.DeviceConfig.Asr.Config,
		)
		if err != nil {
			log.Errorf("Lấy resource ASR thất bại: %v", err)
			return fmt.Errorf("Lấy resource ASR thất bại: %w", err)
		}

		// Lưu tham chiếu resource vào private field
		a.resourceMu.Lock()
		a.asrResource = asrWrapper
		asrProvider = asrWrapper.GetProvider()
		a.resourceMu.Unlock()
		log.Debugf("Lấy resource ASR mới")
	} else {
		// Tái dùng resource hiện có
		asrProvider = a.asrResource.GetProvider()
		a.resourceMu.Unlock()
		log.Debugf("Tái dùng resource ASR hiện có")
	}

	// Tạo lại context và channel ASR
	state.Asr.Ctx, state.Asr.Cancel = context.WithCancel(ctx)
	state.Asr.AsrAudioChannel = make(chan []float32, 100)

	// Khởi động lại nhận diện streaming
	asrResultChannel, err := asrProvider.StreamingRecognize(state.Asr.Ctx, state.Asr.AsrAudioChannel)
	if err != nil {
		// Nhận diện thất bại, trả resource (vì resource có thể đã hỏng)
		a.releaseResource()
		log.Errorf("Restart nhận diện streaming ASR thất bại: %v", err)
		return fmt.Errorf("Restart nhận diện streaming ASR thất bại: %w", err)
	}

	state.AsrResultChannel = asrResultChannel
	// Reset thời gian thống kê để tính tổng thời gian lượt hội thoại này
	state.MarkTurnStart()
	if a.session != nil {
		a.session.TraceTurnStart(state.Asr.Ctx, state.Statistic.TurnStartTs)
	}
	log.Debugf("Restart nhận diện ASR thành công")
	return nil
}

// StartAsrRecognitionLoop khởi động vòng xử lý kết quả nhận diện ASR
// onMessageSave: callback lưu message
// onError: callback xử lý lỗi (như đóng session)
func (a *ASRManager) StartAsrRecognitionLoop(
	ctx context.Context,
	onMessageSave AsrMessageSaveCallback,
	onError func(error),
) {
	state := a.clientState

	// Khởi động goroutine xử lý kết quả ASR
	go func() {
		// Dùng defer đảm bảo release resource ASR khi goroutine thoát
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("goroutine xử lý kết quả ASR panic: %v, stack: %s", r, string(debug.Stack()))
			}
			// Dù thoát bình thường hay panic đều release resource
			a.releaseResource()
		}()

		//Idle tối đa 60s
		var startIdleTime, maxIdleTime int64
		startIdleTime = time.Now().Unix()
		maxIdleTime = 60

		// Bộ đếm chờ khi trạng thái không cho restart (tránh loop vô hạn)
		var invalidStatusWaitCount int64
		maxInvalidStatusWaitCount := int64(10) // chờ tối đa 10 lần (khoảng 1 giây)

		// Bảo vệ ngắn hạn với kết quả rỗng: tránh dịch vụ ASR lỗi liên tục trả chuỗi rỗng khiến flow chính loop chết
		const emptyResultProtectWindow = 3 * time.Second
		const maxEmptyResultInWindow = 3
		emptyResultWindowStart := time.Now()
		emptyResultCount := 0

		// Bảo vệ ngắn hạn lỗi recoverable: tránh upstream liên tục trả instance invalid dẫn tới reconnect vô hạn
		const recoverableErrorProtectWindow = 10 * time.Second
		const maxRecoverableErrorInWindow = 3
		recoverableErrorWindowStart := time.Now()
		recoverableErrorCount := 0

		isAllowedToRestart := func() bool {
			allowed := state.Status == ClientStatusListening || state.Status == ClientStatusListenStop
			if state.IsRealTime() {
				allowed = state.Status != ClientStatusInit
			}
			return allowed
		}
		resumeAudioIdle := func() {
			state.ResumeAudioIdleWindow(time.Now())
		}
		startAudioIdle := func() {
			state.StartAudioIdleWindow(time.Now())
		}
		closeAudioIdleTimeout := func(reason string) {
			if !state.AudioIdleTimeoutPending() {
				return
			}

			state.ClearAudioIdleTimeoutPending()
			log.Infof("Hoàn tất chốt do audio idle timeout: device=%s, reason=%s", state.DeviceID, reason)
			if a.session != nil {
				a.session.CloseWithReason(chatSessionCloseReasonAudioIdleTimeout)
				return
			}
			if onError != nil {
				onError(fmt.Errorf("audio idle timeout: %s", reason))
			}
		}

		for {
			select {
			case <-ctx.Done():
				log.Debugf("asr ctx done")
				return
			default:
			}

			result, isRetry, err := state.RetireAsrResult(ctx)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, context.Canceled) {
					log.Debugf("Xử lý kết quả ASR thất bại, ASR đã cancel: %v", err)
				} else {
					log.Errorf("Xử lý kết quả ASR thất bại: %v", err)
				}
				if onError != nil {
					onError(err)
				}
				return
			}
			if !isRetry {
				log.Debugf("asrResult is not retry, return")
				return
			}
			text := result.Text

			if result.RetryReason != "" {
				if state.AudioIdleTimeoutPending() {
					closeAudioIdleTimeout(result.RetryReason)
					return
				}

				now := time.Now()
				if now.Sub(recoverableErrorWindowStart) > recoverableErrorProtectWindow {
					recoverableErrorWindowStart = now
					recoverableErrorCount = 0
				}
				recoverableErrorCount++
				log.Warnf(
					"Lỗi ASR recoverable: reason=%s, count=%d/%d, status=%s",
					result.RetryReason,
					recoverableErrorCount,
					maxRecoverableErrorInWindow,
					state.Status,
				)

				if recoverableErrorCount >= maxRecoverableErrorInWindow {
					err := fmt.Errorf("ASR liên tục trigger lỗi recoverable trong thời gian ngắn(%d lần/%s), dừng retry và ngắt kết nối", recoverableErrorCount, recoverableErrorProtectWindow)
					log.Errorf("%v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				switch result.RetryReason {
				case asr_types.RetryReasonDoubaoResponseCode45000081, asr_types.RetryReasonXunfeiServiceInstanceInvalid, asr_types.RetryReasonAliyunQwen3ConnectionClosed:
					a.releaseResource()
					if isAllowedToRestart() {
						invalidStatusWaitCount = 0
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("Restart nhận diện sau lỗi ASR recoverable thất bại: reason=%s, err=%v", result.RetryReason, restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						resumeAudioIdle()
						continue
					}

					log.Warnf("Khi lỗi ASR recoverable xảy ra, trạng thái hiện tại không cho restart ngay: reason=%s, status=%s, realtime=%v", result.RetryReason, state.Status, state.IsRealTime())
					state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: recoverable error but restart not allowed yet")
					resumeAudioIdle()
					continue
				case asr_types.RetryReasonDoubaoWaitingNextPacketTimeout:
					log.Warnf("Doubao ASR session idle timeout, suspend stream hiện tại và chờ giọng nói lần sau để dựng lại")
					state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: doubao waiting next packet timeout")
					resumeAudioIdle()
					continue
				}
			}

			if text != "" {
				asrFinalTs := time.Now().UnixMilli()
				state.MarkAsrFinalTextAt(asrFinalTs)
				if a.session != nil {
					a.session.TraceAsrFinalText(ctx, asrFinalTs)
				}
				log.Debugf("Xử lý kết quả ASR: %s, thời gian: %d ms", text, state.GetAsrDuration())

				state.ClearAudioIdleTimeoutPending()
				// Reset bộ đếm kết quả rỗng sau khi nhận diện thành công
				emptyResultWindowStart = time.Now()
				emptyResultCount = 0
				recoverableErrorWindowStart = time.Now()
				recoverableErrorCount = 0

				//Nếu đang ở mode realtime, cần dừng LLM và TTS hiện tại
				if state.IsRealTime() && viper.GetInt("chat.realtime_mode") == 2 {
					shouldInterrupt := true
					if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
						shouldInterrupt = false
						log.Debugf("Thiết bị %s realtime media gate active, trì hoãn tới khi gate ASR final quyết định, bỏ qua interrupt bằng kết quả ASR", state.DeviceID)
					}
					if shouldInterrupt {
						log.Debugf("OnListenStart ở mode realtime, dừng LLM và TTS hiện tại")
						if a.session != nil {
							a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.StartAsrRecognitionLoop realtime_mode=2 ASR result interrupt")
						} else {
							state.AfterAsrSessionCtx.CancelWithReason("ASRManager.StartAsrRecognitionLoop: realtime_mode=2 ASR result interrupt")
						}
					}
				}

				// Reset bộ đếm retry
				startIdleTime = time.Now().Unix()

				//Khi nhận được kết quả ASR, kết thúc input giọng nói (OnVoiceSilence sẽ lấy kết quả voiceprint bất đồng bộ)
				state.OnVoiceSilence()

				// Lấy kết quả voiceprint tạm lưu (có timeout)
				speakerResult := a.getSpeakerResult()
				speakerInterrupted := false
				if a.session != nil {
					speakerInterrupted = a.session.ConsumeTurnSpeakerInterrupted()
				}

				if a.session != nil {
					payload, stop, hookErr := a.session.hookHub.EmitASROutput(a.session.hookContext(ctx), chathooks.ASROutputData{Text: text, SpeakerResult: speakerResult})
					if hookErr != nil {
						log.Warnf("ASR_OUTPUT hook thực thi thất bại: %v", hookErr)
					}
					text = payload.Text
					speakerResult = payload.SpeakerResult
					if stop {
						log.Infof("ASR_OUTPUT hook yêu cầu dừng flow hiện tại")
						state.Asr.ClearHistoryAudio()
						if state.UsesAudioIdleClock() {
							startAudioIdle()
						} else {
							state.ResetAudioIdleWindow()
						}
						continue
					}
				}

				if a.session != nil {
					allowChat, denyReason := a.session.ShouldAllowSpeakerChat(speakerResult, speakerInterrupted)
					if !allowChat {
						log.Infof(
							"Bỏ kết quả ASR và skip STT/LLM: device=%s, reason=%s, speaker_interrupted=%v, speaker_result=%+v, text=%q",
							state.DeviceID,
							denyReason,
							speakerInterrupted,
							speakerResult,
							text,
						)
						state.Asr.ClearHistoryAudio()

						if !state.IsRealTime() {
							startAudioIdle()
							return
						}
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("Restart nhận diện sau khi bỏ kết quả ASR thất bại: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						startAudioIdle()
						continue
					}
				}

				// Tạo user message, dùng text đã được hook rewrite để đi vào chuỗi side effect tiếp theo
				userMsg := &schema.Message{
					Role:    schema.User,
					Content: text,
				}

				// Tạo MessageID (dùng MD5 hash để rút ngắn, tránh vượt giới hạn varchar(64) trong DB)
				// Format gốc：{SessionID}-{Role}-{Timestamp}
				rawMessageID := fmt.Sprintf("%s-%s-%d",
					state.SessionID,
					userMsg.Role,
					time.Now().UnixMilli())
				// Dùng MD5 hash tạo chuỗi hex cố định 32 ký tự
				hash := md5.Sum([]byte(rawMessageID))
				messageID := hex.EncodeToString(hash[:])

				// Thêm đồng bộ vào memory (dùng cho context LLM)
				state.AddMessage(userMsg)

				// Lấy dữ liệu audio (audio lịch sử ASR)
				audioData := state.Asr.GetHistoryAudio()
				state.Asr.ClearHistoryAudio()

				// Lưu message qua callback
				if onMessageSave != nil {
					onMessageSave(userMsg, messageID, audioData)
				}

				// Kết quả ASR gửi cho client cũng dùng text đã được hook rewrite
				err = a.serverTransport.SendAsrResult(text)
				if err != nil {
					log.Errorf("Gửi message ASR thất bại: %v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				if a.session != nil {
					handledByRealtimeGate, gateErr := a.session.tryHandleRealtimeMcpAudioASR(ctx, text)
					if gateErr != nil {
						log.Warnf("Điều khiển nhanh realtime media playback thất bại: device=%s text=%q err=%v", state.DeviceID, text, gateErr)
					}
					if handledByRealtimeGate {
						if !state.IsRealTime() {
							return
						}
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("Restart nhận diện ASR sau điều khiển realtime media thất bại: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						startAudioIdle()
						continue
					}
				}

				// Thêm vào queue (chuyển sang xử lý trong ASRManager)
				if err := a.addAsrResultToQueue(text, speakerResult); err != nil {
					log.Errorf("Bắt đầu hội thoại thất bại: %v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				// Ở mode không realtime, nhận diện ASR hoàn tất thì trả resource
				// Ở mode realtime, resource được quản lý tự động trong RestartAsrRecognition (trả resource cũ trước rồi lấy resource mới)
				if !state.IsRealTime() {
					return
				}

				// Ở mode realtime, restart nhận diện ASR (RestartAsrRecognition sẽ trả resource cũ trước rồi lấy resource mới)
				if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
					log.Errorf("Restart nhận diện ASR thất bại: %v", restartErr)
					if onError != nil {
						onError(restartErr)
					}
					return
				}
				// Ở mode realtime, tiếp tục vòng lặp xử lý kết quả ASR tiếp theo
				continue
			} else {
				log.Debugf(
					"Chi tiết kết quả ASR rỗng: status=%s, emptyReason=%s, client_voice_stop=%v, history_audio_samples=%d, voice_duration=%dms, voice_duration_in_session=%dms, idle_duration=%dms, realtime=%v",
					state.Status,
					result.EmptyReason,
					state.GetClientVoiceStop(),
					state.Asr.GetHistoryAudioLen(),
					state.Vad.GetVoiceDuration(),
					state.Vad.GetVoiceDurationInSession(),
					state.Vad.GetIdleDuration(),
					state.IsRealTime(),
				)
				if state.AudioIdleTimeoutPending() {
					closeAudioIdleTimeout(result.EmptyReason)
					return
				}
				if result.EmptyReason != "" {
					log.Debugf("Kết quả ASR rỗng đã phân loại: reason=%s, status=%s", result.EmptyReason, state.Status)
					emptyResultWindowStart = time.Now()
					emptyResultCount = 0

					if result.EmptyReason == asr_types.EmptyReasonNoServerResponse ||
						result.EmptyReason == asr_types.EmptyReasonProviderEmptyFinal {
						state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: empty final result from provider")
						resumeAudioIdle()
						continue
					}
				}

				now := time.Now()
				if now.Sub(emptyResultWindowStart) > emptyResultProtectWindow {
					emptyResultWindowStart = now
					emptyResultCount = 0
				}
				emptyResultCount++
				if emptyResultCount >= maxEmptyResultInWindow {
					err := fmt.Errorf("ASR liên tục trả kết quả rỗng trong thời gian ngắn(%d lần/%s), trigger bảo vệ và ngắt kết nối", emptyResultCount, emptyResultProtectWindow)
					log.Errorf("%v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				// Trường hợp text rỗng
				select {
				case <-ctx.Done():
					log.Debugf("asr ctx done")
					return
				default:
				}

				log.Debugf("ready Restart Asr, state.Status: %s", state.Status)
				// Ở mode realtime, ngay cả khi trạng thái là LLMStart hoặc TTSStart vẫn nên tiếp tục listen (cho phép restart ASR)
				// Ở mode không realtime, chỉ trạng thái Listening hoặc ListenStop mới cho phép restart ASR
				if isAllowedToRestart() {
					// Trạng thái cho phép restart, reset bộ đếm chờ
					invalidStatusWaitCount = 0
					// Text rỗng, kiểm tra có cần restart ASR không
					diffTs := time.Now().Unix() - startIdleTime
					if startIdleTime > 0 && diffTs <= maxIdleTime {
						log.Warnf("Kết quả nhận diện ASR rỗng, thử restart nhận diện ASR, diff ts: %d", diffTs)
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("Restart nhận diện ASR thất bại: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						resumeAudioIdle()
						continue
					} else {
						log.Warnf("Kết quả nhận diện ASR rỗng, đã đạt thời gian idle tối đa: %d", maxIdleTime)
						if onError != nil {
							onError(fmt.Errorf("Kết quả nhận diện ASR rỗng, đã đạt thời gian idle tối đa: %d", maxIdleTime))
						}
						return
					}
				} else {
					// Trường hợp trạng thái không cho restart, chờ ngắn rồi tiếp tục loop để trạng thái có cơ hội phục hồi
					invalidStatusWaitCount++
					if invalidStatusWaitCount >= maxInvalidStatusWaitCount {
						// Chờ timeout, thoát vòng lặp
						log.Debugf("Trạng thái là %s, realtime: %v, chờ%dlần vẫn không đổi, thoát vòng nhận diện ASR", state.Status, state.IsRealTime(), maxInvalidStatusWaitCount)
						return
					}
					// Chờ ngắn rồi tiếp tục loop, chờ trạng thái phục hồi.
					log.Debugf("Trạng thái là %s, realtime: %v, không cho restart, chờ trạng thái phục hồi (số lần chờ: %d/%d)", state.Status, state.IsRealTime(), invalidStatusWaitCount, maxInvalidStatusWaitCount)
					time.Sleep(200 * time.Millisecond) // chờ100ms
					continue
				}
			}
		}
	}()
}

func trimFirstSpeechAudio(allData []float32, currentFrameSamples, sampleRate, channels int) []float32 {
	if len(allData) == 0 {
		return nil
	}
	if currentFrameSamples <= 0 || currentFrameSamples > len(allData) || sampleRate <= 0 || channels <= 0 {
		return allData
	}

	maxPreSpeechSamples := sampleRate * channels * maxFirstSpeechPreAudioMs / 1000
	keepSamples := currentFrameSamples + maxPreSpeechSamples
	if keepSamples >= len(allData) {
		return allData
	}

	audio := make([]float32, keepSamples)
	copy(audio, allData[len(allData)-keepSamples:])
	return audio
}

// getSpeakerResult Lấy kết quả voiceprint tạm lưu (có timeout)
func (a *ASRManager) getSpeakerResult() *speaker.IdentifyResult {
	if a.session == nil || a.session.speakerManager == nil {
		return nil
	}

	log.Debugf("speakerManager: %+v, IsActive: %+v", a.session.speakerManager, a.session.speakerManager.IsActive())

	timeout := time.NewTimer(200 * time.Millisecond)
	defer timeout.Stop()

	var speakerResult *speaker.IdentifyResult
	select {
	case <-a.session.speakerResultReady:
		a.session.speakerResultMu.RLock()
		speakerResult = a.session.pendingSpeakerResult
		a.session.speakerResultMu.RUnlock()
	case <-timeout.C:
		// Sau timeout đọc kết quả hiện tại (có thể nil)
		a.session.speakerResultMu.RLock()
		speakerResult = a.session.pendingSpeakerResult
		a.session.speakerResultMu.RUnlock()
		log.Debugf("Lấy kết quả nhận diện voiceprint timeout, dùng kết quả hiện tại")
	}
	log.Debugf("Lấy kết quả nhận diện voiceprint: %+v", speakerResult)
	return speakerResult
}

// addAsrResultToQueue Thêm kết quả ASR vào queue (chuyển sang xử lý trong ASRManager)
func (a *ASRManager) addAsrResultToQueue(text string, speakerResult *speaker.IdentifyResult) error {
	if a.session == nil {
		return fmt.Errorf("session is nil")
	}
	return a.session.AddAsrResultToQueue(text, speakerResult)
}
