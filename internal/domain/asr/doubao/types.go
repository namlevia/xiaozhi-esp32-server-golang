package doubao

import "strings"

const (
	legacyDoubaoNonstreamPath = "bigmodel_nostream"
	doubaoStreamingPath       = "bigmodel_async"
)

// DoubaoV2Config là cấu trúc config ASR Doubao.
type DoubaoV2Config struct {
	AppID             string // App ID
	AccessToken       string // Access token
	WsURL             string // WebSocket URL
	ResourceID        string // Resource ID
	ModelName         string // Tên model
	EndWindowSize     int    // Kích thước cửa sổ kết thúc
	EnablePunc        bool   // Có bật dấu câu hay không
	EnableITN         bool   // Có bật ITN hay không
	EnableDDC         bool   // Có bật DDC hay không
	ResultType        string // Mode trả kết quả
	ShowUtterances    bool   // Có trả thông tin tách câu hay không
	ForceToSpeechTime int    // Thời lượng tối thiểu trước khi ép chuyển sang speech
	EnableNonstream   bool   // Có bật bản tối ưu streaming hai chiều hay không
	ChunkDuration     int    // Thời lượng chunk (ms)
	Timeout           int    // Timeout (giây)
}

// DefaultConfig là config mặc định.
var DefaultConfig = DoubaoV2Config{
	WsURL:             "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async",
	ResourceID:        "volc.bigasr.sauc.duration",
	ModelName:         "bigmodel",
	EndWindowSize:     800,
	EnablePunc:        true,
	EnableITN:         true,
	EnableDDC:         false,
	ResultType:        "full",
	ShowUtterances:    true,
	ForceToSpeechTime: 1000,
	EnableNonstream:   false,
	ChunkDuration:     200,
	Timeout:           30,
}

func normalizeDoubaoWsURL(wsURL string) string {
	if wsURL == "" || !strings.Contains(wsURL, legacyDoubaoNonstreamPath) {
		return wsURL
	}
	return strings.ReplaceAll(wsURL, legacyDoubaoNonstreamPath, doubaoStreamingPath)
}

// DoubaoV2Request là cấu trúc request ASR Doubao.
type DoubaoV2Request struct {
	User struct {
		UID string `json:"uid"`
	} `json:"user"`
	Audio struct {
		Format   string `json:"format"`
		Rate     int    `json:"rate"`
		Bits     int    `json:"bits"`
		Channel  int    `json:"channel"`
		Language string `json:"language"`
	} `json:"audio"`
	Request struct {
		ModelName         string `json:"model_name"`
		EndWindowSize     int    `json:"end_window_size"`
		EnablePunc        bool   `json:"enable_punc"`
		EnableITN         bool   `json:"enable_itn"`
		EnableDDC         bool   `json:"enable_ddc"`
		ResultType        string `json:"result_type"`
		ShowUtterances    bool   `json:"show_utterances"`
		ForceToSpeechTime int    `json:"force_to_speech_time"`
		EnableNonstream   bool   `json:"enable_nonstream"`
	} `json:"request"`
}

// DoubaoV2Response là cấu trúc response ASR Doubao.
type DoubaoV2Response struct {
	Code   int `json:"code"`
	Result struct {
		Text string `json:"text"`
	} `json:"result,omitempty"`
	Error string `json:"error,omitempty"`
}
