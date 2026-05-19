package msg

import (
	"encoding/json"

	types_audio "xiaozhi-esp32-server-golang/internal/data/audio"
)

const (
	MDeviceMockPubTopicPrefix = "device-server"
	MDeviceMockSubTopicPrefix = "null"
	MDeviceSubTopicPrefix     = "/p2p/device_sub/"
	MDevicePubTopicPrefix     = "/p2p/device_public/"
	MDeviceLifecycleTopic     = MDevicePubTopicPrefix + "_server/lifecycle"
	MServerSubTopicPrefix     = "/p2p/device_public/#"
	MServerPubTopicPrefix     = MDeviceSubTopicPrefix
)

const (
	MqttLifecycleType         = "mqtt_lifecycle"
	MqttLifecycleStateOnline  = "online"
	MqttLifecycleStateOffline = "offline"
)

// Hằng loại message
const (
	MessageTypeHello      = "hello"       // Message handshake
	MessageTypeAbort      = "abort"       // Message abort
	MessageTypeListen     = "listen"      // Message listen
	MessageTypeIot        = "iot"         // Message IoT
	MessageTypeMcp        = "mcp"         // Message MCP
	MessageTypeGoodBye    = "goodbye"     // Message goodbye
	MessageTypeSpeakReady = "speak_ready" // Thiết bị đã sẵn sàng nhận phát chủ động
)

// Hằng loại message server
const (
	ServerMessageTypeHello        = "hello"         // Message handshake
	ServerMessageTypeStt          = "stt"           // Speech to text
	ServerMessageTypeTts          = "tts"           // Text to speech
	ServerMessageTypeIot          = "iot"           // Message IoT
	ServerMessageTypeLlm          = "llm"           // Mô hình ngôn ngữ lớn
	ServerMessageTypeText         = "text"          // Message text
	ServerMessageTypeGoodBye      = "goodbye"       // Message goodbye
	ServerMessageTypeSpeakRequest = "speak_request" // Request phát chủ động
)

// Hằng trạng thái message
const (
	MessageStateStart         = "start"          // Trạng thái bắt đầu
	MessageStateSentenceStart = "sentence_start" // Trạng thái bắt đầu câu
	MessageStateSentenceEnd   = "sentence_end"   // Trạng thái kết thúc câu
	MessageStateStop          = "stop"           // Trạng thái dừng
	MessageStateDetect        = "detect"         // Trạng thái detect
	MessageStateAbort         = "abort"          // Trạng thái abort
	MessageStateSuccess       = "success"        // Trạng thái thành công
	MessageStateReady         = "ready"          // Thiết bị đã sẵn sàng
)

type UdpConfig struct {
	Server string `json:"server"`
	Port   int    `json:"port"`
	Key    string `json:"key"`
	Nonce  string `json:"nonce"`
}

type MqttLifecycleEvent struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	State    string `json:"state"`
	ClientID string `json:"client_id,omitempty"`
	Ts       int64  `json:"ts"`
}

// ServerMessage biểu diễn message server.
type ServerMessage struct {
	Type        string                   `json:"type"`
	Text        string                   `json:"text,omitempty"`
	SessionID   string                   `json:"session_id,omitempty"`
	Version     int                      `json:"version"`
	State       string                   `json:"state,omitempty"`
	Transport   string                   `json:"transport,omitempty"`
	AudioFormat *types_audio.AudioFormat `json:"audio_params,omitempty"`
	Emotion     string                   `json:"emotion,omitempty"`
	AutoListen  *bool                    `json:"auto_listen,omitempty"`
	Udp         *UdpConfig               `json:"udp,omitempty"`
	PayLoad     json.RawMessage          `json:"payload,omitempty"`
}
