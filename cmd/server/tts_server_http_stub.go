//go:build !tts_server

package main

import log "xiaozhi-esp32-server-golang/logger"

func StartTtsServerHTTP(configPath string) {
	log.Warn("tts_server nhúng chưa được biên dịch vào binary này, hãy biên dịch lại với -tags tts_server để bật")
}

func StopTtsServerHTTP() {}
