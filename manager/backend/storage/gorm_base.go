package storage

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GormBaseStorage lớp cơ sở lưu trữ GORM chung
// Chứa triển khai chung cho toàn bộ thao tác lưu trữ dựa trên GORM
type GormBaseStorage struct {
	DB *gorm.DB // Field export để lớp con có thể truy cập
}

// NewGormBaseStorage tạo instance lưu trữ GORM cơ sở
func NewGormBaseStorage(db *gorm.DB) *GormBaseStorage {
	return &GormBaseStorage{
		DB: db,
	}
}

// Ping kiểm tra kết nối cơ sở dữ liệu
func (s *GormBaseStorage) Ping() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Ping()
}

// Close đóng kết nối cơ sở dữ liệu
func (s *GormBaseStorage) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// BeginTx bắt đầu giao dịch
func (s *GormBaseStorage) BeginTx(ctx context.Context) (Transaction, error) {
	tx := s.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	transaction := &GormTransaction{
		DB: tx,
	}
	transaction.init()
	return transaction, nil
}

// GormTransaction triển khai giao dịch GORM chung
type GormTransaction struct {
	DB *gorm.DB
	*GormUserStorage
	*GormDeviceStorage
	*GormAgentStorage
	*GormConfigStorage
}

// init khởi tạo các thành phần lưu trữ trong giao dịch
func (t *GormTransaction) init() {
	t.GormUserStorage = &GormUserStorage{db: t.DB}
	t.GormDeviceStorage = &GormDeviceStorage{db: t.DB}
	t.GormAgentStorage = &GormAgentStorage{db: t.DB}
	t.GormConfigStorage = &GormConfigStorage{db: t.DB}
}

// Commit xác nhận giao dịch
func (t *GormTransaction) Commit() error {
	if err := t.DB.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback hoàn tác giao dịch
func (t *GormTransaction) Rollback() error {
	if err := t.DB.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
