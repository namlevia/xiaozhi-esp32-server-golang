package http

import (
	"context"
	"time"
)

// ManagerClient là HTTP client chuyên dụng cho backend Manager.
type ManagerClient struct {
	client *Client
}

// ManagerClientConfig là config client Manager.
type ManagerClientConfig struct {
	BaseURL    string        // Địa chỉ backend Manager
	AuthToken  string        // Auth token (tùy chọn)
	Timeout    time.Duration // Thời gian timeout request
	MaxRetries int           // Số lần retry tối đa
}

// NewManagerClient tạo HTTP client backend Manager.
func NewManagerClient(cfg ManagerClientConfig) *ManagerClient {
	client := NewClient(ClientConfig{
		BaseURL:    cfg.BaseURL,
		AuthToken:  cfg.AuthToken,
		Timeout:    cfg.Timeout,
		MaxRetries: cfg.MaxRetries,
	})

	return &ManagerClient{
		client: client,
	}
}

// DoRequest thực thi HTTP request (bọc DoRequest của client dùng chung).
func (m *ManagerClient) DoRequest(ctx context.Context, opts RequestOptions) error {
	return m.client.DoRequest(ctx, opts)
}

// DoRequestRaw thực thi HTTP request và trả response raw.
func (m *ManagerClient) DoRequestRaw(ctx context.Context, opts RequestOptions) ([]byte, error) {
	return m.client.DoRequestRaw(ctx, opts)
}
