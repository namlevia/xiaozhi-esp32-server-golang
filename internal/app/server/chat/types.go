package chat

import (
	"context"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
)

// ChatSessionOperator định nghĩa interface thao tác ChatSession cần cho local MCP tool.
// Interface này dùng để tách LLMManager và ChatSession, tránh phụ thuộc vòng.
type ChatSessionOperator interface {
	// LocalMcpCloseChat đóng phiên chat.
	LocalMcpCloseChat() error

	// LocalMcpClearHistory xóa lịch sử hội thoại.
	LocalMcpClearHistory() error

	// LocalMcpPlayMusic phát nhạc.
	LocalMcpPlayMusic(ctx context.Context, params *PlayMusicParams) error

	// LocalMcpSwitchDeviceRole đổi vai trò thiết bị theo tên role, hỗ trợ khớp mờ.
	LocalMcpSwitchDeviceRole(ctx context.Context, roleName string) (string, error)

	// LocalMcpRestoreDeviceDefaultRole khôi phục role mặc định của thiết bị.
	LocalMcpRestoreDeviceDefaultRole(ctx context.Context) error

	// LocalMcpSearchKnowledge truy xuất knowledge base liên kết với agent hiện tại.
	LocalMcpSearchKnowledge(ctx context.Context, query string, topK int, knowledgeBaseIDs []uint) ([]config_types.KnowledgeSearchHit, error)

	// LocalMcpControlMusicPlayback điều khiển media playback cấp session hiện tại.
	LocalMcpControlMusicPlayback(ctx context.Context, params *MusicPlaybackControlParams) (*MusicPlaybackControlResult, error)

	// Có thể bổ sung thao tác khác khi cần.
	// GetDeviceID() string
	// IsActive() bool
}
