package sqlite

import (
	"fmt"
	"path/filepath"
	"xiaozhi/manager/backend/config"
)

// Config cấu hình SQLite
type Config struct {
	// FilePath đường dẫn file cơ sở dữ liệu (ví dụ: ./data/xiaozhi.db hoặc /path/to/xiaozhi.db)
	FilePath string `json:"file_path"`

	// Cấu hình connection pool (SQLite thường chỉ cần một kết nối)
	MaxIdleConns    int `json:"max_idle_conns"`
	MaxOpenConns    int `json:"max_open_conns"`
	ConnMaxLifetime int `json:"conn_max_lifetime"`
}

// NewConfigFromDatabase tạo cấu hình SQLite từ cấu hình cơ sở dữ liệu
func NewConfigFromDatabase(cfg *config.SQLiteConfig) *Config {
	filePath := cfg.FilePath
	if filePath == "" {
		filePath = "./data/xiaozhi.db"
	}

	return &Config{
		FilePath:        filePath,
		MaxIdleConns:    1,
		MaxOpenConns:    1,
		ConnMaxLifetime: 3600,
	}
}

// DSN tạo tên nguồn dữ liệu (định dạng GORM SQLite)
func (c *Config) DSN() string {
	// Đảm bảo dùng tiền tố file: để hỗ trợ nhiều tùy chọn hơn
	return "file:" + c.FilePath + "?_foreign_keys=on&_journal_mode=WAL"
}

// Validate xác thực cấu hình
func (c *Config) Validate() error {
	if c.FilePath == "" {
		return fmt.Errorf("SQLite file path is required")
	}

	// Kiểm tra phần mở rộng file
	ext := filepath.Ext(c.FilePath)
	if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
		return fmt.Errorf("SQLite file must have .db, .sqlite or .sqlite3 extension")
	}

	return nil
}

// ValidateConfig xác thực cấu hình SQLite
func ValidateConfig(cfg *config.SQLiteConfig) error {
	if cfg == nil {
		return fmt.Errorf("SQLite config is required")
	}
	if cfg.FilePath == "" {
		return fmt.Errorf("SQLite file path is required")
	}

	// Kiểm tra phần mở rộng file
	ext := filepath.Ext(cfg.FilePath)
	if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
		return fmt.Errorf("SQLite file must have .db, .sqlite or .sqlite3 extension")
	}

	return nil
}
