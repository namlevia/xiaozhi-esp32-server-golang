package database

import (
	"fmt"
	"log"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitWithReset khởi tạo cơ sở dữ liệu và đặt lại toàn bộ bảng (chỉ dùng cho môi trường phát triển)
func InitWithReset(cfg config.DatabaseConfig) *gorm.DB {
	storageType := cfg.GetStorageType()
	var db *gorm.DB
	var err error

	if storageType == "sqlite" {
		if cfg.SQLite == nil {
			log.Fatal("Cấu hình SQLite trống")
		}
		db, err = gorm.Open(sqlite.Open(cfg.SQLite.FilePath), &gorm.Config{})
	} else {
		if cfg.MySQL == nil {
			log.Fatal("Cấu hình MySQL trống")
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.MySQL.Username, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		log.Fatal("Kết nối cơ sở dữ liệu thất bại:", err)
	}

	log.Println("Cảnh báo: đang đặt lại bảng cơ sở dữ liệu, toàn bộ dữ liệu sẽ bị xóa!")

	// Xóa toàn bộ bảng
	err = db.Migrator().DropTable(
		&models.User{},
		&models.Device{},
		&models.Agent{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.Role{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.VoiceClone{},
		&models.VoiceCloneAudio{},
	)
	if err != nil {
		log.Printf("Lỗi khi xóa bảng (có thể bảng không tồn tại): %v", err)
	}

	log.Println("Xóa bảng cơ sở dữ liệu hoàn tất!")
	return db
}
