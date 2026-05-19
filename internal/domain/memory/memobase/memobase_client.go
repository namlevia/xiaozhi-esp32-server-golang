package memobase

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/memodb-io/memobase/src/client/memobase-go/blob"
	"github.com/memodb-io/memobase/src/client/memobase-go/core"
)

var (
	clientInstance *MemobaseClient
	once           sync.Once
	configOnce     sync.Once
	// Dùng namespace UUID cố định để tạo UUID v5 cho device ID
	// Như vậy cùng một device ID luôn map tới cùng một UUID
	deviceNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // namespace DNS
)

// MemobaseClient manager Memobase client
type MemobaseClient struct {
	client *core.MemoBaseClient
	users  sync.Map // cache object user
	sync.RWMutex
	EnableSearch    bool
	SearchThreshold float64
	SearchTopk      int
}

// GetWithConfig Lấy instance Memobase client bằng config (singleton mode)
func GetWithConfig(config map[string]interface{}) (*MemobaseClient, error) {
	var initErr error
	configOnce.Do(func() {
		iClient := &MemobaseClient{
			users: sync.Map{},
		}
		// Đọc config liên quan memobase từ cấu hình		// Đọc các mục cấu hình bắt buộc
		projectUrlInterface, ok := config["base_url"]
		if !ok {
			initErr = fmt.Errorf("thiếu cấu hình memobase.base_url")
			return
		}
		baseUrl, ok := projectUrlInterface.(string)
		if !ok {
			initErr = fmt.Errorf("memobase.base_url phải là string")
			return
		}

		apiKeyInterface, ok := config["api_key"]
		if !ok {
			initErr = fmt.Errorf("thiếu cấu hình memobase.api_key")
			return
		}
		apiKey, ok := apiKeyInterface.(string)
		if !ok {
			initErr = fmt.Errorf("memobase.api_key phải là string")
			return
		}

		if baseUrl == "" || apiKey == "" {
			initErr = fmt.Errorf("Cấu hình Memobase chưa đầy đủ: base_url hoặc api_key rỗng")
			log.Log().Errorf("Khởi tạo Memobase thất bại: %v", initErr)
			return
		}

		// Đọc cấu hình search tùy chọn
		enableSearchInterface, ok := config["enable_search"]
		if ok {
			enableSearch, ok := enableSearchInterface.(bool)
			if ok {
				iClient.EnableSearch = enableSearch
			}
		}

		thresholdInterface, ok := config["search_threshold"]
		if ok {
			threshold, ok := thresholdInterface.(float64)
			if ok {
				iClient.SearchThreshold = threshold
			}
		}

		topKInterface, ok := config["search_topk"]
		if ok {
			topK, ok := topKInterface.(int)
			if ok {
				iClient.SearchTopk = topK
			}
		}

		// Tạo client
		client, err := core.NewMemoBaseClient(baseUrl, apiKey)
		if err != nil {
			initErr = fmt.Errorf("Tạo Memobase client thất bại: %v", err)
			log.Log().Errorf("Khởi tạo Memobase thất bại: %v", initErr)
			return
		}

		iClient.client = client
		clientInstance = iClient

		log.Log().Infof("Khởi tạo Memobase client thành công, project_url: %s", baseUrl)
	})

	if initErr != nil {
		return nil, initErr
	}
	return clientInstance, nil
}

// deviceIDToUUID Chuyển device ID sang định dạng UUID v5
// Dùng UUID v5 để đảm bảo cùng một device ID luôn sinh cùng UUID
func deviceIDToUUID(deviceID string) string {
	return uuid.NewSHA1(deviceNamespace, []byte(deviceID)).String()
}

func IsEnableSearch() bool {
	return clientInstance.EnableSearch
}

// AddMessage Thêm message vào Memobase
func (m *MemobaseClient) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	memobaseUserID := deviceIDToUUID(agentID)
	// Dựng message
	messages := []blob.OpenAICompatibleMessage{
		{
			Role:    string(msg.Role),
			Content: msg.Content,
		},
	}

	// Nếu có tool call, thêm vào message
	if len(msg.ToolCalls) > 0 {
		return nil
		/*for _, toolCall := range msg.ToolCalls {
			messages = append(messages, blob.OpenAICompatibleMessage{
				Role:    "tool",
				Content: fmt.Sprintf("Tool: %s, Args: %v", toolCall.Function.Name, toolCall.Function.Arguments),
			})
		}*/
	}

	// Tạo ChatBlob
	chatBlob := &blob.ChatBlob{
		BaseBlob: blob.BaseBlob{
			Type: blob.ChatType,
		},
		Messages: messages,
	}

	// Lấy hoặc tạo instance user (dùng userID dạng UUID)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Lấy hoặc tạo user thất bại, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return fmt.Errorf("Lấy hoặc tạo user thất bại: %v", err)
	}

	// Chèn message (async)
	blobID, err := user.Insert(chatBlob, false)
	if err != nil {
		log.Log().Errorf("Thêm message vào Memobase thất bại, deviceID: %s, error: %v", agentID, err)
		return fmt.Errorf("Thêm message vào Memobase thất bại: %v", err)
	}

	//user.Flush(blob.ChatType, false)

	log.Log().Debugf("Thêm message vào Memobase thành công, deviceID: %s, blobID: %s", agentID, blobID)
	return nil
}

func (m *MemobaseClient) Flush(ctx context.Context, agentID string) error {
	memobaseUserID := deviceIDToUUID(agentID)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Refresh memory người dùng thất bại, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return fmt.Errorf("Refresh memory người dùng thất bại: %v", err)
	}
	user.Flush(blob.ChatType, false)
	return nil
}

// GetContext Lấy context người dùng
func (m *MemobaseClient) GetContext(ctx context.Context, agentID string, maxToken int) (string, error) {

	// Chuyển device ID sang định dạng UUID (Memobase yêu cầu)
	memobaseUserID := deviceIDToUUID(agentID)

	// Lấy instance user (không thực hiện HTTP GET request, chỉ tạo instance)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Lấy instance user thất bại, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Lấy instance user thất bại: %v", err)
	}

	// Lấy context, dùng option mặc định
	context, err := user.Context(&core.ContextOptions{
		MaxTokenSize: maxToken,
	})
	if err != nil {
		log.Log().Errorf("Lấy context từ Memobase thất bại, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Lấy context từ Memobase thất bại: %v", err)
	}

	log.Log().Debugf("Lấy context từ Memobase thành công, agentID: %s, độ dài context: %d", agentID, len(context))
	return context, nil
}

func (m *MemobaseClient) Search(ctx context.Context, agentID string, query string, topK int, timeRangeDays int64) (string, error) {
	if !m.EnableSearch {
		return "", nil
	}
	topK = m.SearchTopk
	// Chuyển device ID sang định dạng UUID (Memobase yêu cầu)
	memobaseUserID := deviceIDToUUID(agentID)

	// Lấy instance user (không thực hiện HTTP GET request, chỉ tạo instance)
	user, err := m.getUser(memobaseUserID)
	if err != nil {
		log.Log().Errorf("Lấy instance user thất bại, agentID: %s, memobaseUserID: %s, error: %v", agentID, memobaseUserID, err)
		return "", fmt.Errorf("Lấy instance user thất bại: %v", err)
	}

	topK = 2

	// Search event
	userEventList, err := user.SearchEvent(query, topK, 0.2, int(timeRangeDays))
	if err != nil {
		log.Log().Errorf("Search event từ Memobase thất bại, agentID: %s, error: %v", agentID, err)
		return "", fmt.Errorf("Search event từ Memobase thất bại: %v", err)
	}

	var eventList []string
	for _, event := range userEventList {
		eventList = append(eventList, fmt.Sprintf("- %s: %s", event.CreatedAt, event.EventData.EventTip))
	}

	// Chuyển thành string
	userEventStr := strings.Join(eventList, "\n")

	log.Log().Debugf("Search event từ Memobase thành công, agentID: %s, số event: %d", agentID, len(eventList))
	return userEventStr, nil
}

// AddBatchMessages thêm message hàng loạt vào Memobase
func (m *MemobaseClient) AddBatchMessages(ctx context.Context, userID string, messages []schema.Message) error {
	m.Lock()
	defer m.Unlock()

	if len(messages) == 0 {
		return nil
	}

	// Chuyển định dạng message
	blobMessages := make([]blob.OpenAICompatibleMessage, 0, len(messages))
	for _, msg := range messages {
		blobMessages = append(blobMessages, blob.OpenAICompatibleMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Tạo ChatBlob
	chatBlob := &blob.ChatBlob{
		BaseBlob: blob.BaseBlob{
			Type: blob.ChatType,
		},
		Messages: blobMessages,
	}

	// Chuyển device ID sang định dạng UUID (Memobase yêu cầu)
	memobaseUserID := deviceIDToUUID(userID)

	// Lấy hoặc tạo instance user (dùng userID dạng UUID)
	user, err := m.getUser(userID)
	if err != nil {
		log.Log().Errorf("Thêm message hàng loạt: lấy hoặc tạo user thất bại, deviceID: %s, memobaseUserID: %s, error: %v", userID, memobaseUserID, err)
		return fmt.Errorf("Lấy hoặc tạo user thất bại: %v", err)
	}

	// Chèn message (async)
	blobID, err := user.Insert(chatBlob, false)
	if err != nil {
		log.Log().Errorf("Thêm message hàng loạt vào Memobase thất bại, deviceID: %s, error: %v", userID, err)
		return fmt.Errorf("Thêm message hàng loạt vào Memobase thất bại: %v", err)
	}

	log.Log().Debugf("Thêm hàng loạt %d message vào Memobase thành công, deviceID: %s, blobID: %s", len(messages), userID, blobID)
	return nil
}

// GetMessages Lấy lịch sử message của người dùng
// Triển khai interface BaseMemoryProvider
// Lưu ý: Memobase chủ yếu dùng cho memory dài hạn và tăng cường context, không cung cấp chức năng truy xuất lịch sử message
func (m *MemobaseClient) GetMessages(ctx context.Context, agentID string, count int) ([]*schema.Message, error) {
	return []*schema.Message{}, nil
}

// ResetMemory Reset memory của người dùng
// Triển khai interface MemoryProvider
// Lưu ý: reset memory của Memobase cần xóa dữ liệu người dùng qua API
func (m *MemobaseClient) ResetMemory(ctx context.Context, userID string) error {
	// TODO: Nếu Memobase SDK cung cấp interface xóa dữ liệu người dùng thì gọi tại đây
	// Hiện trả nil để biểu thị thao tác thành công (dù chưa xóa thực tế)
	log.Log().Infof("Yêu cầu reset memory Memobase: userID=%s (lưu ý: Memobase không hỗ trợ reset trực tiếp)", userID)
	return nil
}

// Close đóng client (nếu cần)
func (m *MemobaseClient) Close() error {
	log.Log().Info("Memobase client đã đóng")
	return nil
}

// todo thêm cache object user
func (m *MemobaseClient) getUser(userID string) (*core.User, error) {
	if user, ok := m.users.Load(userID); ok {
		return user.(*core.User), nil
	}

	memobaseUserID := deviceIDToUUID(userID)
	user, err := m.client.GetOrCreateUser(memobaseUserID)
	if err != nil {
		return nil, fmt.Errorf("Lấy instance user thất bại: %v", err)
	}

	m.users.Store(userID, user)
	return user, nil
}
