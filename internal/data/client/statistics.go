package client

import "time"

// Statistic struct đã deprecated, vui lòng dùng statistic_plugin để lấy thống kê tại MetricTtsStop
type Statistic struct {
	TurnStartTs        int64
	VoiceSilenceTs     int64
	AsrFirstTextTs     int64
	AsrFinalTextTs     int64
	LlmStartTs         int64
	LlmFirstTokenTs    int64
	LlmFirstSentenceTs int64
	LlmEndTs           int64
	TtsStartTs         int64
	TtsFirstFrameTs    int64
	TtsStopTs          int64
}

// MarkTurnStart ghi thời gian bắt đầu lượt
func (state *ClientState) MarkTurnStart() {
	state.Statistic.TurnStartTs = time.Now().UnixMilli()
	state.Statistic.VoiceSilenceTs = 0
	state.Statistic.AsrFirstTextTs = 0
	state.Statistic.AsrFinalTextTs = 0
}

// MarkVoiceSilenceAt ghi thời gian bắt đầu voice silence, trả về có phải lần ghi đầu của lượt này không
func (state *ClientState) MarkVoiceSilenceAt(ts int64) bool {
	if state.Statistic.VoiceSilenceTs != 0 {
		return false
	}
	state.Statistic.VoiceSilenceTs = ts
	return true
}

// MarkVoiceSilence ghi thời gian bắt đầu voice silence, trả về có phải lần ghi đầu của lượt này không
func (state *ClientState) MarkVoiceSilence() bool {
	return state.MarkVoiceSilenceAt(time.Now().UnixMilli())
}

// MarkAsrFirstText ghi thời điểm ASR trả text đầu tiên
func (state *ClientState) MarkAsrFirstText() {
	if state.Statistic.AsrFirstTextTs == 0 {
		state.Statistic.AsrFirstTextTs = time.Now().UnixMilli()
	}
}

// MarkAsrFinalText ghi thời điểm ASR trả text final
func (state *ClientState) MarkAsrFinalText() {
	state.MarkAsrFinalTextAt(time.Now().UnixMilli())
}

// MarkAsrFinalTextAt ghi thời điểm ASR trả text final, trả về có phải lần ghi đầu của lượt này không.
func (state *ClientState) MarkAsrFinalTextAt(ts int64) bool {
	if state.Statistic.AsrFinalTextTs != 0 {
		return false
	}
	state.Statistic.AsrFinalTextTs = ts
	return true
}

// MarkLlmStart ghi thời điểm LLM bắt đầu
func (state *ClientState) MarkLlmStart() {
	state.Statistic.LlmStartTs = time.Now().UnixMilli()
	state.Statistic.LlmFirstTokenTs = 0
	state.Statistic.LlmFirstSentenceTs = 0
	state.Statistic.LlmEndTs = 0
}

// MarkLlmFirstToken ghi thời điểm LLM trả token đầu tiên
func (state *ClientState) MarkLlmFirstToken() {
	state.Statistic.LlmFirstTokenTs = time.Now().UnixMilli()
}

// MarkLlmFirstSentenceAt ghi thời điểm LLM output câu đầu, trả về có phải lần ghi đầu của lượt này không
func (state *ClientState) MarkLlmFirstSentenceAt(ts int64) bool {
	if state.Statistic.LlmFirstSentenceTs != 0 {
		return false
	}
	state.Statistic.LlmFirstSentenceTs = ts
	return true
}

// MarkLlmFirstSentence ghi thời điểm LLM output câu đầu, trả về có phải lần ghi đầu của lượt này không
func (state *ClientState) MarkLlmFirstSentence() bool {
	return state.MarkLlmFirstSentenceAt(time.Now().UnixMilli())
}

// MarkLlmEnd ghi thời điểm LLM kết thúc
func (state *ClientState) MarkLlmEnd() {
	state.Statistic.LlmEndTs = time.Now().UnixMilli()
}

// MarkTtsStart ghi thời điểm TTS bắt đầu
func (state *ClientState) MarkTtsStart() {
	state.Statistic.TtsStartTs = time.Now().UnixMilli()
	state.Statistic.TtsFirstFrameTs = 0
	state.Statistic.TtsStopTs = 0
}

// MarkTtsFirstFrame ghi thời điểm frame TTS đầu tiên
func (state *ClientState) MarkTtsFirstFrame() {
	if state.Statistic.TtsFirstFrameTs == 0 {
		state.Statistic.TtsFirstFrameTs = time.Now().UnixMilli()
	}
}

// MarkTtsStop ghi thời điểm TTS kết thúc
func (state *ClientState) MarkTtsStop() {
	state.Statistic.TtsStopTs = time.Now().UnixMilli()
}

// SetStartAsrTs set thời điểm ASR bắt đầu (alias để tương thích)
func (state *ClientState) SetStartAsrTs() { state.MarkVoiceSilence() }

// SetStartLlmTs set thời điểm LLM bắt đầu (alias để tương thích)
func (state *ClientState) SetStartLlmTs() { state.MarkLlmStart() }

// SetStartTtsTs set thời điểm TTS bắt đầu (alias để tương thích)
func (state *ClientState) SetStartTtsTs() { state.MarkTtsStart() }

// GetAsrDuration lấy thời gian xử lý ASR (đã deprecated, chỉ giữ chữ ký method)
func (state *ClientState) GetAsrDuration() int64 {
	return calcStatisticDuration(state.Statistic.VoiceSilenceTs, state.Statistic.AsrFinalTextTs)
}

// GetAsrLlmTtsDuration lấy tổng thời gian (đã deprecated, chỉ giữ chữ ký method)
func (state *ClientState) GetAsrLlmTtsDuration() int64 {
	return calcStatisticDuration(state.Statistic.VoiceSilenceTs, state.Statistic.TtsFirstFrameTs)
}

// GetLlmDuration lấy thời gian LLM (đã deprecated, chỉ giữ chữ ký method)
func (state *ClientState) GetLlmDuration() int64 {
	return calcStatisticDuration(state.Statistic.LlmStartTs, state.Statistic.LlmEndTs)
}

// GetTtsDuration lấy thời gian TTS (đã deprecated, chỉ giữ chữ ký method)
func (state *ClientState) GetTtsDuration() int64 {
	return calcStatisticDuration(state.Statistic.TtsStartTs, state.Statistic.TtsStopTs)
}

func calcStatisticDuration(start, end int64) int64 {
	if start <= 0 || end <= 0 || end < start {
		return 0
	}
	return end - start
}

func (s *Statistic) Reset() {
	if s == nil {
		return
	}
	*s = Statistic{}
}
