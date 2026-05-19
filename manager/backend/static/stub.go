//go:build !embed_ui

package static

import "embed"

// FS trống khi chưa bật embed_ui, giai đoạn phát triển không mount tài nguyên tĩnh frontend
var FS = embed.FS{}
