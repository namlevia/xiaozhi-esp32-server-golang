package memory

import (
	"context"
	"fmt"

	"xiaozhi-esp32-server-golang/internal/domain/memory/mem0"
	"xiaozhi-esp32-server-golang/internal/domain/memory/memobase"
	"xiaozhi-esp32-server-golang/internal/domain/memory/memos"
	"xiaozhi-esp32-server-golang/internal/domain/memory/nomemo"

	"github.com/cloudwego/eino/schema"
)

// MemoryProvider interface memory provider
// Định nghĩa các method lõi mà mọi memory provider cần triển khai
type MemoryProvider interface {
	// AddMessage Thêm một message vào memory
	AddMessage(ctx context.Context, agentID string, msg schema.Message) error

	// GetMessages Lấy lịch sử message của người dùng
	GetMessages(ctx context.Context, agentId string, count int) ([]*schema.Message, error)

	// GetContext Lấy context của người dùng để tăng cường LLM prompt
	GetContext(ctx context.Context, agentId string, maxToken int) (string, error)

	// Search Tìm kiếm memory của người dùng
	Search(ctx context.Context, agentId string, query string, topK int, timeRangeDays int64) (string, error)

	// Flush Refresh memory của người dùng
	Flush(ctx context.Context, agentId string) error

	// ResetMemory Reset memory của người dùng
	ResetMemory(ctx context.Context, agentId string) error
}

// MemoryType Loại memory
type MemoryType string

const (
	MemoryTypeNone     MemoryType = "nomemo"
	MemoryTypeMemobase MemoryType = "memobase" // Memobase memory dài hạn
	MemoryTypeMem0     MemoryType = "mem0"     // Mem0 dịch vụ memory
	MemoryTypeMemOS    MemoryType = "memos"    // MemOS（tương thích Mem0 API）
)

// GetProvider Lấy memory provider theo loại chỉ định
func GetProvider(memoryType MemoryType, config map[string]interface{}) (MemoryProvider, error) {
	return GetProviderByType(memoryType, config)
}

// GetProviderByType Lấy memory provider theo type
func GetProviderByType(memoryType MemoryType, config map[string]interface{}) (MemoryProvider, error) {
	if memoryType == "" {
		memoryType = MemoryTypeNone
	}
	switch memoryType {
	case MemoryTypeNone:
		return nomemo.Get(), nil
	case MemoryTypeMemobase:
		return memobase.GetWithConfig(config)
	case MemoryTypeMem0:
		return mem0.GetMem0ClientWithConfig(config)
	case MemoryTypeMemOS:
		return memos.GetWithConfig(config)
	default:
		return nil, fmt.Errorf("unsupported memory type: %v", memoryType)
	}
}
