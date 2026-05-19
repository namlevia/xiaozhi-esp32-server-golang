//go:build manager

package main

import (
	"context"
	"net/http"
	"time"

	log "xiaozhi-esp32-server-golang/logger"
	mbconfig "xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/database"
	"xiaozhi/manager/backend/router"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultManagerHTTPPort   = "9000"
	defaultManagerConfigPath = "manager.json"
)

var (
	managerHTTPServer *http.Server // Handle HTTP manager nhúng trong tiến trình này, dùng để shutdown mềm.
	managerDB         *gorm.DB     // DB manager đang dùng, đóng khi thoát.
)

// StartManagerHTTP khởi động dịch vụ HTTP manager trong tiến trình này. main quyết định có gọi hay không theo -manager-enable.
// configPath là đường dẫn file cấu hình manager; nếu rỗng sẽ dùng đường dẫn mặc định.
func StartManagerHTTP(configPath string) {
	if configPath == "" {
		configPath = defaultManagerConfigPath
	}
	log.Infof("Đang khởi động dịch vụ HTTP manager nhúng, file cấu hình: %s", configPath)

	cfg := mbconfig.LoadWithPath(configPath)
	port := cfg.Server.Port
	if port == "" {
		port = defaultManagerHTTPPort
	}
	cfg.Server.Port = port

	db := database.Init(cfg.Database)
	if db == nil {
		log.Warn("Khởi tạo database manager thất bại, bỏ qua khởi động manager HTTP")
		return
	}
	managerDB = db

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.Setup(db, cfg)

	managerHTTPServer = &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Infof("Dịch vụ HTTP manager đã khởi động trên cổng: %s", port)
		if err := managerHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Dịch vụ HTTP manager thoát bất thường: %v", err)
		}
	}()
}

// StopManagerHTTP shutdown mềm dịch vụ HTTP manager nhúng trong tiến trình này và đóng kết nối database.
func StopManagerHTTP() {
	if managerHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := managerHTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Đóng HTTP manager bị timeout hoặc lỗi: %v", err)
		}
		managerHTTPServer = nil
		log.Info("Dịch vụ HTTP manager đã đóng")
	}
	if managerDB != nil {
		database.Close(managerDB)
		managerDB = nil
	}
}
