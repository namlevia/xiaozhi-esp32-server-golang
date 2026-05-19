//go:build asr_server

package main

import (
	"context"
	"net/http"
	"time"

	"voice_server/server"
	log "xiaozhi-esp32-server-golang/logger"
)

const (
	defaultAsrServerConfigPath = "asr_server.json"
)

var (
	asrHTTPServer *http.Server // Handle HTTP asr_server nhúng trong tiến trình này, dùng để shutdown mềm.
)

// StartAsrServerHTTP khởi động dịch vụ HTTP asr_server trong tiến trình này trên một cổng riêng. main quyết định có gọi hay không theo -asr-enable.
// configPath là đường dẫn file cấu hình asr_server; nếu rỗng sẽ dùng đường dẫn mặc định asr_server/config.json.
func StartAsrServerHTTP(configPath string) {
	if configPath == "" {
		configPath = defaultAsrServerConfigPath
	}
	log.Infof("Đang khởi động dịch vụ HTTP asr_server nhúng, file cấu hình: %s", configPath)

	handler, addr, readTimeout, err := server.Setup(configPath)
	if err != nil {
		log.Warnf("Khởi tạo asr_server thất bại, bỏ qua khởi động: %v", err)
		return
	}

	asrHTTPServer = &http.Server{
		Addr:        addr,
		Handler:     handler,
		ReadTimeout: readTimeout,
	}

	go func() {
		log.Infof("Dịch vụ HTTP asr_server đã khởi động tại %s", addr)
		if err := asrHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Dịch vụ HTTP asr_server thoát bất thường: %v", err)
		}
	}()
}

// StopAsrServerHTTP shutdown mềm dịch vụ HTTP asr_server nhúng trong tiến trình này.
func StopAsrServerHTTP() {
	if asrHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := asrHTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Đóng HTTP asr_server bị timeout hoặc lỗi: %v", err)
		}
		asrHTTPServer = nil
		log.Info("Dịch vụ HTTP asr_server đã đóng")
	}
}
