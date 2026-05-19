package config

import "strings"

const DefaultInternalAuthToken = "xiaozhi_admin_secret_key"
const DefaultEndpointAuthToken = "xiaozhi_mcp_openclaw_secret_key"

// ResolveInternalAuthToken phân giải token dùng chung cho dịch vụ nội bộ của console.
// Thứ tự ưu tiên:
// 1. internal_auth_token trong file cấu hình
// 2. Giá trị mặc định (giữ nhất quán với chương trình chính)
func ResolveInternalAuthToken(cfg *Config) string {
	if cfg != nil {
		if token := strings.TrimSpace(cfg.InternalAuthToken); token != "" {
			return token
		}
	}
	return DefaultInternalAuthToken
}

// ResolveEndpointAuthToken phân giải token ký JWT cho endpoint MCP/OpenClaw.
// Thứ tự ưu tiên:
// 1. endpoint_auth_token trong file cấu hình
// 2. Giá trị mặc định (giữ nhất quán với chương trình chính)
func ResolveEndpointAuthToken(cfg *Config) string {
	if cfg != nil {
		if token := strings.TrimSpace(cfg.EndpointAuthToken); token != "" {
			return token
		}
	}
	return DefaultEndpointAuthToken
}
