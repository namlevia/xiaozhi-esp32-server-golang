package mcp

import (
	"fmt"
	"net/url"
	"strings"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

// CheckMCPConfig kiểm tra config MCP và báo cáo vấn đề tiềm ẩn.
func CheckMCPConfig() {
	log.Info("=== Kiểm tra config MCP ===")

	// Kiểm tra trạng thái bật global
	globalEnabled := viper.GetBool("mcp.global.enabled")
	log.Infof("Trạng thái bật MCP global: %v", globalEnabled)

	if !globalEnabled {
		log.Info("MCP global đã tắt, kiểm tra config hoàn tất")
		return
	}

	// Kiểm tra config reconnect
	reconnectInterval := viper.GetInt("mcp.global.reconnect_interval")
	maxAttempts := viper.GetInt("mcp.global.max_reconnect_attempts")
	log.Infof("Config reconnect: interval=%d giây, số lần thử tối đa=%d", reconnectInterval, maxAttempts)

	// Kiểm tra config server
	var serverConfigs []MCPServerConfig
	if err := viper.UnmarshalKey("mcp.global.servers", &serverConfigs); err != nil {
		log.Errorf("❌ Parse config server MCP thất bại: %v", err)
		return
	}

	if len(serverConfigs) == 0 {
		log.Warn("⚠️  Chưa cấu hình server MCP nào")
		return
	}

	log.Infof("Đã cấu hình tổng cộng %d server MCP:", len(serverConfigs))

	enabledCount := 0
	problemCount := 0

	for i, config := range serverConfigs {
		status := "✅"
		issues := []string{}

		// Kiểm tra tên
		if config.Name == "" {
			status = "❌"
			issues = append(issues, "Tên rỗng")
			problemCount++
		}

		transportType, endpoint, err := endpointForConfig(config)
		if err != nil {
			status = "❌"
			issues = append(issues, err.Error())
			problemCount++
		} else {
			if _, parseErr := url.ParseRequestURI(endpoint); parseErr != nil {
				status = "❌"
				issues = append(issues, "Format URL không đúng")
				problemCount++
			}
			if transportType == "sse" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				status = "⚠️"
				issues = append(issues, "Format SSE URL có thể không đúng")
			}
		}

		// Kiểm tra trạng thái bật
		if config.Enabled {
			enabledCount++
		}

		// Xuất kết quả kiểm tra
		issueStr := ""
		if len(issues) > 0 {
			issueStr = fmt.Sprintf(" - Vấn đề: %s", strings.Join(issues, ", "))
		}

		log.Infof("  [%d] %s %s (URL: %s, bật: %v)%s",
			i+1, status, config.Name, endpointForLog(config), config.Enabled, issueStr)
	}

	// Tổng kết
	log.Infof("Kiểm tra cấu hình hoàn tất: %d server đã bật, %d server có vấn đề", enabledCount, problemCount)

	if problemCount > 0 {
		log.Warn("⚠️  Phát hiện vấn đề cấu hình, vui lòng kiểm tra lỗi phía trên và sửa")
	}

	log.Info("=== Kiểm tra config MCP hoàn tất ===")
}

func endpointForLog(config MCPServerConfig) string {
	_, endpoint, err := endpointForConfig(config)
	if err != nil {
		if strings.TrimSpace(config.Url) != "" {
			return config.Url
		}
		return config.SSEUrl
	}
	return endpoint
}
