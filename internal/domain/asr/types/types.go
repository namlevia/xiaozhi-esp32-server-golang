package types

const (
	EmptyReasonNone               = ""
	EmptyReasonNoServerResponse   = "no_server_response"
	EmptyReasonProviderEmptyFinal = "provider_empty_final"

	RetryReasonNone                           = ""
	RetryReasonDoubaoResponseCode45000081     = "doubao_response_code_45000081"
	RetryReasonDoubaoWaitingNextPacketTimeout = "doubao_waiting_next_packet_timeout"
	RetryReasonXunfeiServiceInstanceInvalid   = "xunfei_service_instance_invalid"
	RetryReasonAliyunQwen3ConnectionClosed    = "aliyun_qwen3_connection_closed"
)

// StreamingResult là kết quả nhận diện streaming.
type StreamingResult struct {
	Text        string // Text nhận diện được
	IsFinal     bool   // Có phải kết quả cuối hay không
	Error       error  // Thông tin lỗi
	AsrType     string // Loại ASR
	Mode        string // Mode
	EmptyReason string // Lý do kết quả rỗng; chỉ dùng khi Text rỗng để phân biệt upstream trả rỗng/chạy không tải
	RetryReason string // Lý do lỗi recoverable; chỉ dùng khi cần giải phóng tài nguyên hiện tại và retry
}
