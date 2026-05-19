package http

import "time"

// ClientConfig là config HTTP client.
type ClientConfig struct {
	BaseURL    string        // Base URL
	AuthToken  string        // Auth token (tùy chọn)
	Timeout    time.Duration // Thời gian timeout request
	MaxRetries int           // Số lần retry tối đa (mặc định 3 lần)
}

// RequestOptions là option cho request.
type RequestOptions struct {
	Method      string            // HTTP method
	Path        string            // Request path
	QueryParams map[string]string // Query parameter
	Headers     map[string]string // Custom request header
	Body        interface{}       // Request body (tự động serialize thành JSON)
	Response    interface{}       // Response body (tự động deserialize)
}
