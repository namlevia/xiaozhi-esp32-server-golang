package main

import (
	"flag"
	"log"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/database"
	"xiaozhi/manager/backend/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// Định nghĩa tham số dòng lệnh
	var configFile string
	flag.StringVar(&configFile, "config", "config/config.json", "Đường dẫn file cấu hình")
	flag.StringVar(&configFile, "c", "config/config.json", "Đường dẫn file cấu hình (viết tắt)")
	flag.Parse()

	// Tải cấu hình
	cfg := config.LoadWithPath(configFile)

	// Khởi tạo cơ sở dữ liệu
	db := database.Init(cfg.Database)
	if db == nil {
		log.Fatal("Khởi tạo cơ sở dữ liệu thất bại, dịch vụ dừng")
	}
	defer database.Close(db)

	// Thiết lập chế độ Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Khởi tạo router
	r := router.Setup(db, cfg)

	// Khởi động server
	log.Printf("Đang dùng file cấu hình: %s", configFile)
	log.Printf("Server đang chạy trên cổng: %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Khởi động server thất bại:", err)
	}
}
