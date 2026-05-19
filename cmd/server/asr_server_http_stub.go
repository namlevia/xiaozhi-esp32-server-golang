//go:build !asr_server

package main

import log "xiaozhi-esp32-server-golang/logger"

// StartAsrServerHTTP là bản rỗng khi chưa bật asr_server lúc biên dịch. Dùng -tags asr_server để bật asr_server nhúng.
func StartAsrServerHTTP(configPath string) {
	log.Warn("asr_server nhúng chưa được biên dịch vào binary này, hãy biên dịch lại với -tags asr_server để bật")
}

// StopAsrServerHTTP là bản rỗng khi chưa bật asr_server lúc biên dịch.
func StopAsrServerHTTP() {}
