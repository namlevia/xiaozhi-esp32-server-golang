//go:build !manager

package main

import log "xiaozhi-esp32-server-golang/logger"

// StartManagerHTTP là bản rỗng khi chưa bật manager lúc biên dịch. Dùng -tags manager để bật manager HTTP nhúng.
func StartManagerHTTP(configPath string) {
	log.Warn("manager nhúng chưa được biên dịch vào binary này, hãy biên dịch lại với -tags manager để bật")
}

// StopManagerHTTP là bản rỗng khi chưa bật manager lúc biên dịch.
func StopManagerHTTP() {}
