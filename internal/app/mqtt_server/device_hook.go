package mqtt_server

import (
	"fmt"
	"strings"
	"time"

	mqttServer "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"

	client "xiaozhi-esp32-server-golang/internal/data/msg"
	log "xiaozhi-esp32-server-golang/logger"
)

// DeviceHook xử lý quyền thiết bị và tự động subscribe.
// Người dùng thường không được subscribe tùy ý, chỉ được publish topic chỉ định; khi kết nối sẽ tự subscribe /p2p/device_sub/{mac}.
type DeviceHook struct {
	mqttServer.HookBase
	server           *mqttServer.Server
	publishLifecycle func(event client.MqttLifecycleEvent) error
}

func (h *DeviceHook) ID() string {
	return "custom-device-hook"
}

func (h *DeviceHook) Provides(b byte) bool {
	return b == mqttServer.OnDisconnect || b == mqttServer.OnACLCheck || b == mqttServer.OnSessionEstablished || b == mqttServer.OnSubscribe || b == mqttServer.OnPublish
}

// OnACLCheck kiểm soát quyền publish/subscribe.
func (h *DeviceHook) OnACLCheck(cl *mqttServer.Client, topic string, write bool) bool {
	isAdmin := isAdminUser(cl)

	if isAdmin {
		return true // Super admin không bị giới hạn.
	}

	if write {
		// Chỉ cho phép người dùng thường publish tới "device-server".
		if topic == client.MDeviceMockPubTopicPrefix {
			return true
		}
		log.Warnf("Cấm người dùng thường publish tới %s", topic)
		return false
	}

	mac := parseMacFromClientId(cl.ID)
	if mac == "" {
		log.Warnf("Cấm người dùng thường subscribe %s: không thể parse MAC từ clientID, clientID=%s", topic, cl.ID)
		return false
	}

	allowedTopic := deviceSubTopic(mac)
	if topic == allowedTopic {
		return true
	}

	log.Warnf("Cấm người dùng thường subscribe %s: chỉ cho phép subscribe topic của chính thiết bị %s", topic, allowedTopic)
	return false
}

func (h *DeviceHook) OnConnect(cl *mqttServer.Client, pk packets.Packet) error {
	isAdmin := isAdminUser(cl)
	if isAdmin {
		return nil
	}
	pk.Connect.Clean = true
	return nil
}

func (h *DeviceHook) OnDisconnect(cl *mqttServer.Client, err error, ok bool) {
	if cl == nil {
		log.Warnf("OnDisconnect: client rỗng, err=%v, ok=%v", err, ok)
		return
	}
	isAdmin := isAdminUser(cl)
	mac := parseMacFromClientId(cl.ID)
	deviceID := deviceIDFromClientId(cl.ID)
	takenOver := cl.IsTakenOver()

	log.Infof("OnDisconnect: clientID=%s, deviceID=%s, mac=%s, ok=%v, err=%v, takenOver=%v, isAdmin=%v",
		cl.ID, deviceID, mac, ok, err, takenOver, isAdmin)

	if isAdmin {
		return
	}
	if takenOver {
		log.Infof("Client %s đã được kết nối mới cùng ID tiếp quản, bỏ qua unsubscribe và publish lifecycle offline", cl.ID)
		return
	}
	if mac == "" {
		log.Infof("OnDisconnect: không thể parse MAC address từ clientID, clientID=%s, err=%v, ok=%v", cl.ID, err, ok)
		return
	}

	log.Infof("OnDisconnect: chuẩn bị publish lifecycle offline, clientID=%s, deviceID=%s", cl.ID, deviceID)
	h.publishLifecycleEvent(cl.ID, client.MqttLifecycleStateOffline)
	topic := deviceSubTopic(mac)

	action := h.server.Topics.Unsubscribe(topic, cl.ID)
	log.Infof("OnDisconnect: unsubscribe client %s khỏi topic %s, action=%v", cl.ID, topic, action)

	return
}

// OnSessionEstablished tự động subscribe sau khi kết nối được thiết lập.
func (h *DeviceHook) OnSessionEstablished(cl *mqttServer.Client, pk packets.Packet) {
	isAdmin := isAdminUser(cl)
	mac := parseMacFromClientId(cl.ID)
	deviceID := deviceIDFromClientId(cl.ID)
	if isAdmin {
		return // Super admin không bị giới hạn.
	}
	if mac == "" {
		log.Info("Cảnh báo: không thể parse MAC address từ clientID:", cl.ID)
		return
	}
	log.Infof("OnSessionEstablished: clientID=%s, deviceID=%s, mac=%s, clean=%v", cl.ID, deviceID, mac, pk.Connect.Clean)
	h.publishLifecycleEvent(cl.ID, client.MqttLifecycleStateOnline)

	topic := deviceSubTopic(mac)

	// Subscribe trực tiếp bằng API của server thay vì inject packet.
	clientID := cl.ID
	exists := h.server.Topics.Subscribe(clientID, packets.Subscription{
		Filter: topic,
		Qos:    0,
	})

	log.Infof("Subscribe client %s vào topic %s, exists: %v", clientID, topic, exists)
}

// OnSubscribe in packet subscribe.
func (h *DeviceHook) OnSubscribe(cl *mqttServer.Client, pk packets.Packet) packets.Packet {
	log.Info("=== Nhận packet subscribe ===")
	log.Infof("Client ID: %s", cl.ID)
	log.Infof("Loại packet: %v", pk.FixedHeader.Type)
	log.Infof("Packet ID: %d", pk.PacketID)

	if len(pk.Filters) > 0 {
		log.Info("Thông tin subscribe:")
		for i, sub := range pk.Filters {
			log.Infof("  %d. Topic: %s, QoS: %d", i+1, sub.Filter, sub.Qos)
		}
	}

	log.Info("==================")
	return pk
}

// OnPublish in packet publish.
func (h *DeviceHook) OnPublish(cl *mqttServer.Client, pk packets.Packet) (packets.Packet, error) {
	if cl == nil {
		return pk, nil
	}

	log.Info("=== Nhận packet publish ===")
	log.Infof("Client ID: %s", cl.ID)
	log.Infof("Loại packet: %v", pk.FixedHeader.Type)
	log.Infof("Packet ID: %d", pk.PacketID)
	log.Infof("Topic: %s", pk.TopicName)

	if isAdminUser(cl) {
		return pk, nil
	}

	if len(pk.Payload) > 0 {
		if len(pk.Payload) > 100 {
			log.Infof("Nội dung message (100 byte đầu): %s...", pk.Payload[:100])
		} else {
			log.Infof("Nội dung message: %s", pk.Payload)
		}
	} else {
		log.Info("Nội dung message: <rỗng>")
	}

	mac := parseMacFromClientId(cl.ID)
	if mac == "" {
		log.Info("Cảnh báo: không thể parse MAC address từ clientID:", cl.ID)
		return pk, nil
	}
	forwardTopic := fmt.Sprintf("%s%s", client.MDevicePubTopicPrefix, mac)

	pk.TopicName = forwardTopic

	log.Info("==================")
	return pk, nil
}

func isAdminUser(cl *mqttServer.Client) bool {
	if cl == nil {
		return false
	}
	return string(cl.Properties.Username) == configuredAdminUsername()
}

func parseMacFromClientId(clientId string) string {
	parts := strings.Split(clientId, "@@@")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

func deviceIDFromClientId(clientID string) string {
	mac := parseMacFromClientId(clientID)
	if mac == "" {
		return ""
	}
	return strings.ReplaceAll(mac, "_", ":")
}

func (h *DeviceHook) publishLifecycleEvent(clientID string, state string) {
	if h == nil || h.publishLifecycle == nil {
		return
	}
	deviceID := deviceIDFromClientId(clientID)
	if deviceID == "" {
		log.Warnf("Bỏ qua publish MQTT lifecycle event: không thể parse deviceID, clientID=%s, state=%s", clientID, state)
		return
	}
	event := client.MqttLifecycleEvent{
		Type:     client.MqttLifecycleType,
		DeviceID: deviceID,
		State:    state,
		ClientID: clientID,
		Ts:       time.Now().UnixMilli(),
	}
	log.Infof("Publish MQTT lifecycle event: device=%s, clientID=%s, state=%s, ts=%d", deviceID, clientID, state, event.Ts)
	if err := h.publishLifecycle(event); err != nil {
		log.Warnf("Publish MQTT lifecycle event thất bại: device=%s state=%s err=%v", deviceID, state, err)
	}
}

func deviceSubTopic(mac string) string {
	return fmt.Sprintf("%s%s", client.MDeviceSubTopicPrefix, mac)
}

// Khởi động task định kỳ in danh sách topic đã subscribe.
func (h *DeviceHook) StartPeriodicSubscriptionPrinter(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			h.PrintAllClientSubscriptions()
		}
	}()
}

// In tất cả topic đã subscribe của client.
func (h *DeviceHook) PrintAllClientSubscriptions() {
	log.Info("=== Danh sách topic subscribe của client ===")
	clients := h.server.Clients.GetAll()
	if len(clients) == 0 {
		log.Info("Hiện không có client đang kết nối")
		return
	}

	for clientID := range clients {
		log.Infof("Topic đã subscribe của client %s: ", clientID)

		// Dùng server.Topics.Subscribers("+") để lấy subscriber của mọi topic.
		// Sau đó lọc subscription khớp với clientID hiện tại.
		allSubs := h.server.Topics.Subscribers("+")
		foundTopics := false

		// Kiểm tra subscription của client.
		if subs, ok := allSubs.Subscriptions[clientID]; ok {
			log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
			foundTopics = true
		}

		// Kiểm tra thêm các topic subscription có thể tồn tại.
		allSubs = h.server.Topics.Subscribers("#")
		if subs, ok := allSubs.Subscriptions[clientID]; ok {
			log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
			foundTopics = true
		}

		// Kiểm tra thêm topic cụ thể.
		mac := parseMacFromClientId(clientID)
		if mac != "" {
			topic := deviceSubTopic(mac)
			topicSubs := h.server.Topics.Subscribers(topic)
			if subs, ok := topicSubs.Subscriptions[clientID]; ok {
				log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
				foundTopics = true
			}
		}

		if !foundTopics {
			log.Info("  Không có topic subscribe hoặc không thể lấy dữ liệu")
		}
	}
	log.Info("=====================")
}
