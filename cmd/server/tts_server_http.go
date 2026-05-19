//go:build tts_server

package main

import (
	"context"
	"net/http"
	"time"

	log "xiaozhi-esp32-server-golang/logger"
	ttsserver "xiaozhi-esp32-server-golang/tts_server/server"
)

const (
	defaultTtsServerConfigPath = "tts_server.json"
)

var (
	ttsHTTPServer *http.Server
)

func StartTtsServerHTTP(configPath string) {
	if configPath == "" {
		configPath = defaultTtsServerConfigPath
	}
	log.Infof("Đang khởi động dịch vụ HTTP tts_server nhúng, file cấu hình: %s", configPath)

	handler, addr, readTimeout, writeTimeout, err := ttsserver.Setup(configPath)
	if err != nil {
		log.Warnf("Khởi tạo tts_server thất bại, bỏ qua khởi động: %v", err)
		return
	}

	ttsHTTPServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	go func() {
		log.Infof("Dịch vụ HTTP tts_server đã khởi động tại %s", addr)
		if err := ttsHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Dịch vụ HTTP tts_server thoát bất thường: %v", err)
		}
	}()
}

func StopTtsServerHTTP() {
	if ttsHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ttsHTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("Đóng HTTP tts_server bị timeout hoặc lỗi: %v", err)
		}
		ttsHTTPServer = nil
		log.Info("Dịch vụ HTTP tts_server đã đóng")
	}
}
