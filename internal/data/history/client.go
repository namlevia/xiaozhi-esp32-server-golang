package history

import (
	"context"
	"fmt"
	"time"

	"xiaozhi-esp32-server-golang/internal/components/http"

	"github.com/cloudwego/eino/schema"
)

// MessageType là loại message.
type MessageType string

const (
	MessageTypeUser      MessageType = "user"
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeTool      MessageType = "tool"   // Kết quả gọi tool
	MessageTypeSystem    MessageType = "system" // Message system (nếu dùng)
)

// HistoryClientConfig là config client.
type HistoryClientConfig struct {
	BaseURL   string        // Địa chỉ backend Manager
	AuthToken string        // Auth token
	Timeout   time.Duration // Timeout request
	Enabled   bool          // Có bật hay không
}

// HistoryClient là HTTP client lịch sử chat.
type HistoryClient struct {
	client  *http.ManagerClient
	enabled bool
}

// NewHistoryClient tạo client lịch sử chat.
func NewHistoryClient(cfg HistoryClientConfig) *HistoryClient {
	managerClient := http.NewManagerClient(http.ManagerClientConfig{
		BaseURL:    cfg.BaseURL,
		AuthToken:  cfg.AuthToken,
		Timeout:    cfg.Timeout,
		MaxRetries: 3, // Retry mặc định 3 lần
	})

	return &HistoryClient{
		client:  managerClient,
		enabled: cfg.Enabled,
	}
}

// SaveMessageRequest là request lưu message.
type SaveMessageRequest struct {
	MessageID     string                 `json:"message_id"`
	DeviceID      string                 `json:"device_id"`
	AgentID       string                 `json:"agent_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	Role          MessageType            `json:"role"`
	Content       string                 `json:"content"`
	ToolCallID    string                 `json:"tool_call_id,omitempty"`    // Tool call ID (dùng cho role Tool)
	ToolCallsJSON *string                `json:"tool_calls_json,omitempty"` // JSON danh sách tool call (dùng cho role Assistant), nil nghĩa là NULL
	AudioData     string                 `json:"audio_data,omitempty"`      // Encode base64
	AudioFormat   string                 `json:"audio_format,omitempty"`
	AudioDuration int                    `json:"audio_duration,omitempty"`
	AudioSize     int                    `json:"audio_size,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SaveMessage lưu message.
func (c *HistoryClient) SaveMessage(ctx context.Context, req *SaveMessageRequest) error {
	if !c.enabled {
		return nil
	}
	return c.client.DoRequest(ctx, http.RequestOptions{
		Method: "POST",
		Path:   "/api/internal/history/messages",
		Body:   req,
	})
}

// UpdateMessageAudioRequest là request cập nhật audio message.
type UpdateMessageAudioRequest struct {
	MessageID   string                 `json:"message_id"`
	AudioData   string                 `json:"audio_data"` // Encode base64
	AudioFormat string                 `json:"audio_format"`
	AudioSize   int                    `json:"audio_size"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMessageAudio cập nhật audio message.
func (c *HistoryClient) UpdateMessageAudio(ctx context.Context, req *UpdateMessageAudioRequest) error {
	if !c.enabled {
		return nil
	}
	return c.client.DoRequest(ctx, http.RequestOptions{
		Method: "PUT",
		Path:   "/api/internal/history/messages/" + req.MessageID + "/audio",
		Body:   req,
	})
}

// GetMessagesRequest là request lấy message.
type GetMessagesRequest struct {
	DeviceID  string `json:"device_id"`
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id,omitempty"`
	Limit     int    `json:"limit"` // Giới hạn số lượng
}

// GetMessagesResponse là response lấy message.
type GetMessagesResponse struct {
	Messages []MessageItem `json:"messages"`
}

// MessageItem là item message (dùng để load khởi tạo, không gồm audio).
type MessageItem struct {
	MessageID  string            `json:"message_id"`
	Role       string            `json:"role"` // user/assistant/tool/system
	Content    string            `json:"content"`
	ToolCallID string            `json:"tool_call_id,omitempty"` // Dùng cho role Tool
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`   // Dùng cho role Assistant
	CreatedAt  string            `json:"created_at"`
}

// GetMessages lấy message từ DB Manager (dùng để load khởi tạo).
func (c *HistoryClient) GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error) {
	if !c.enabled {
		return nil, fmt.Errorf("history client is disabled")
	}

	// Tạo query parameter
	queryParams := map[string]string{
		"device_id": req.DeviceID,
		"agent_id":  req.AgentID,
		"limit":     fmt.Sprintf("%d", req.Limit),
	}
	if req.SessionID != "" {
		queryParams["session_id"] = req.SessionID
	}

	var resp GetMessagesResponse
	err := c.client.DoRequest(ctx, http.RequestOptions{
		Method:      "GET",
		Path:        "/api/internal/history/messages",
		QueryParams: queryParams,
		Response:    &resp,
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
