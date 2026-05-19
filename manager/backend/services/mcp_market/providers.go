package mcp_market

import "strings"

var providerPresets = []MarketProviderPreset{
	{
		ID:          ProviderGeneric,
		Name:        "Tùy chỉnh",
		AuthType:    AuthTypeNone,
		Description: "Nhập thủ công địa chỉ API danh mục/chi tiết, phù hợp mọi MCP market.",
	},
	{
		ID:                ProviderModelScope,
		Name:              "ModelScope",
		CatalogURL:        "https://www.modelscope.cn/openapi/v1/mcp/servers",
		DetailURLTemplate: "https://www.modelscope.cn/openapi/v1/mcp/servers/{raw_id}",
		AuthType:          AuthTypeBearer,
		Description:       "Luôn dùng Bearer Token để xác thực, chỉ kéo dịch vụ đã kích hoạt (/operational).",
	},
}

func ListProviderPresets() []MarketProviderPreset {
	out := make([]MarketProviderPreset, len(providerPresets))
	copy(out, providerPresets)
	return out
}

func NormalizeProviderID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return ProviderGeneric
	}
	for _, preset := range providerPresets {
		if id == preset.ID {
			return id
		}
	}
	return ProviderGeneric
}

func GetProviderPreset(id string) (MarketProviderPreset, bool) {
	id = NormalizeProviderID(id)
	for _, preset := range providerPresets {
		if preset.ID == id {
			return preset, true
		}
	}
	return MarketProviderPreset{}, false
}
