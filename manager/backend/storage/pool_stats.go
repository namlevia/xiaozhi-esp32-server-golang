package storage

import (
	"sync"
	"time"
)

// PoolStatsData dữ liệu thống kê nhóm tài nguyên
type PoolStatsData struct {
	Timestamp time.Time              `json:"timestamp"`
	Stats     map[string]interface{} `json:"stats"`
}

// PoolStatsStorage lưu thống kê nhóm tài nguyên trong bộ nhớ, chỉ giữ dữ liệu mới nhất
type PoolStatsStorage struct {
	mu     sync.RWMutex
	latest *PoolStatsData // Chỉ giữ dữ liệu thống kê mới nhất
}

var (
	globalPoolStatsStorage *PoolStatsStorage
	once                   sync.Once
)

// GetPoolStatsStorage lấy bộ lưu thống kê nhóm tài nguyên toàn cục (singleton)
func GetPoolStatsStorage() *PoolStatsStorage {
	once.Do(func() {
		globalPoolStatsStorage = &PoolStatsStorage{
			latest: nil,
		}
	})
	return globalPoolStatsStorage
}

// AddStats thêm dữ liệu thống kê, chỉ giữ dữ liệu mới nhất và ghi đè dữ liệu cũ
func (s *PoolStatsStorage) AddStats(stats map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ghi đè trực tiếp dữ liệu mới nhất
	s.latest = &PoolStatsData{
		Timestamp: time.Now(),
		Stats:     stats,
	}
}

// GetLatestStats lấy dữ liệu thống kê mới nhất
func (s *PoolStatsStorage) GetLatestStats() *PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return nil
	}

	// Trả về bản sao dữ liệu mới nhất
	latest := *s.latest
	return &latest
}

// GetAllStats lấy toàn bộ dữ liệu thống kê, hiện chỉ trả về bản mới nhất
func (s *PoolStatsStorage) GetAllStats() []PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return []PoolStatsData{}
	}

	// Chỉ trả về một bản dữ liệu mới nhất
	return []PoolStatsData{*s.latest}
}

// GetStatsByTimeRange lấy dữ liệu thống kê theo khoảng thời gian, chỉ trả về bản mới nhất nếu nằm trong khoảng
func (s *PoolStatsStorage) GetStatsByTimeRange(start, end time.Time) []PoolStatsData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.latest == nil {
		return []PoolStatsData{}
	}

	// Kiểm tra dữ liệu mới nhất có nằm trong khoảng thời gian không
	if s.latest.Timestamp.After(start) && s.latest.Timestamp.Before(end) {
		return []PoolStatsData{*s.latest}
	}

	return []PoolStatsData{}
}

// GetStatsCount lấy số bản dữ liệu hiện đang lưu, chỉ giữ bản mới nhất nên trả về 0 hoặc 1
func (s *PoolStatsStorage) GetStatsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest == nil {
		return 0
	}
	return 1
}
