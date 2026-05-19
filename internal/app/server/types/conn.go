package types

import "context"

// IConn là interface kết nối độc lập protocol, do các adapter websocket/mqtt_udp triển khai.
// Có thể mở rộng method theo nhu cầu thực tế.

const (
	TransportTypeWebsocket = "websocket"
	TransportTypeMqttUdp   = "udp"
)

type IConn interface {
	// Gửi dữ liệu command/signaling
	SendCmd(msg []byte) error
	// Nhận dữ liệu command/signaling
	RecvCmd(ctx context.Context, timeout int) ([]byte, error)
	// Gửi dữ liệu voice
	SendAudio(audio []byte) error
	// Nhận dữ liệu voice
	RecvAudio(ctx context.Context, timeout int) ([]byte, error)

	GetDeviceID() string

	Close() error
	OnClose(func(deviceId string))

	CloseAudioChannel() error

	GetTransportType() string

	// Lấy dữ liệu private
	GetData(key string) (interface{}, error)
}

type OnNewConnection func(conn IConn)
