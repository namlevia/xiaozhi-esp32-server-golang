package nomemo

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// NoMemoProvider triển khai memory provider rỗng
// Dùng khi người dùng không cần chức năng memory, mọi method đều là triển khai rỗng
type NoMemoProvider struct{}

// Get Lấy instance NoMemoProvider
func Get() *NoMemoProvider {
	return &NoMemoProvider{}
}

// AddMessage Thêm một message vào memory(triển khai rỗng)
func (n *NoMemoProvider) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	// Triển khai rỗng, không thực hiện thao tác nào
	return nil
}

// GetMessages Lấy lịch sử message của người dùng(triển khai rỗng)
func (n *NoMemoProvider) GetMessages(ctx context.Context, agentId string, count int) ([]*schema.Message, error) {
	// Trả về danh sách message rỗng
	return []*schema.Message{}, nil
}

// GetContext lấy context của người dùng (triển khai rỗng)
func (n *NoMemoProvider) GetContext(ctx context.Context, agentId string, maxToken int) (string, error) {
	// Trả về chuỗi rỗng
	return "", nil
}

// Search Tìm kiếm memory của người dùng(triển khai rỗng)
func (n *NoMemoProvider) Search(ctx context.Context, agentId string, query string, topK int, timeRangeDays int64) (string, error) {
	// Trả về chuỗi rỗng
	return "", nil
}

// Flush Refresh memory của người dùng(triển khai rỗng)
func (n *NoMemoProvider) Flush(ctx context.Context, agentId string) error {
	// Triển khai rỗng, không thực hiện thao tác nào
	return nil
}

// ResetMemory Reset memory của người dùng(triển khai rỗng)
func (n *NoMemoProvider) ResetMemory(ctx context.Context, agentId string) error {
	// Triển khai rỗng, không thực hiện thao tác nào
	return nil
}
