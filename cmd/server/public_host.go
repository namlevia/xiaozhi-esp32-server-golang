package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

func applyPublicHostOverrides() {
	host := strings.TrimSpace(os.Getenv("PUBLIC_HOST"))
	if host == "" {
		host = detectLANIPv4()
	}
	if host == "" {
		return
	}

	websocketPort := viper.GetInt("websocket.port")
	if websocketPort == 0 {
		websocketPort = 1233
	}

	viper.Set("udp.external_host", host)
	viper.Set("vision.vision_url", fmt.Sprintf("http://%s:%d/xiaozhi/api/vision", host, websocketPort))
	viper.Set("ota.test.websocket.url", fmt.Sprintf("ws://%s:%d/xiaozhi/v1/", host, websocketPort))
	if viper.GetBool("ota.test.mqtt.enable") {
		viper.Set("ota.test.mqtt.endpoint", host)
	}

	log.Infof("Đã áp dụng địa chỉ LAN/public host cho thiết bị: %s", host)
}

func detectLANIPv4() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || !isPrivateIPv4(ip) {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func isPrivateIPv4(ip net.IP) bool {
	return ip[0] == 10 || ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 || ip[0] == 192 && ip[1] == 168
}
