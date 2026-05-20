package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"xiaozhi-esp32-server-golang/internal/app/server"
	user_config "xiaozhi-esp32-server-golang/internal/domain/config"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

func main() {
	// Phân tích tham số dòng lệnh.
	configFile := flag.String("c", defaultConfigFilePath, "đường dẫn file cấu hình")
	managerEnable := flag.Bool("manager-enable", defaultManagerEnable, "có bật manager nhúng hay không")
	managerConfig := flag.String("manager-config", "", "đường dẫn file cấu hình manager, tùy chọn khi bật; mặc định manager/backend/config/config.json")
	asrEnable := flag.Bool("asr-enable", defaultAsrEnable, "có bật asr_server nhúng hay không")
	asrConfig := flag.String("asr-config", "", "đường dẫn file cấu hình asr_server, tùy chọn khi bật; mặc định asr_server/config.json")
	ttsEnable := flag.Bool("tts-enable", defaultTtsEnable, "có bật tts_server nhúng hay không")
	ttsConfig := flag.String("tts-config", "", "đường dẫn file cấu hình tts_server, tùy chọn khi bật; mặc định tts_server.json")
	flag.Parse()

	if *configFile == "" {
		fmt.Println("Đường dẫn file cấu hình không được để trống")
		return
	}

	// Khởi động manager trước Init, nếu không updateConfigFromAPI trong Init sẽ không kết nối được manager và bị kẹt.
	if *managerEnable {
		StartManagerHTTP(*managerConfig)
	}
	EnsureAIOAssets(*asrConfig, *ttsConfig, *asrEnable, *ttsEnable)
	if *asrEnable {
		StartAsrServerHTTP(*asrConfig)
	}
	if *ttsEnable {
		StartTtsServerHTTP(*ttsConfig)
	}
	err := Init(*configFile)
	if err != nil {
		return
	}

	// Khởi động dịch vụ pprof theo cấu hình.
	if viper.GetBool("server.pprof.enable") {
		pprofPort := viper.GetInt("server.pprof.port")
		go func() {
			log.Infof("Khởi động dịch vụ pprof, cổng: %d", pprofPort)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil); err != nil {
				log.Errorf("Khởi động dịch vụ pprof thất bại: %v", err)
			}
		}()
		log.Infof("Địa chỉ pprof: http://localhost:%d/debug/pprof/", pprofPort)
	} else {
		log.Info("Dịch vụ pprof đã tắt")
	}

	// Tạo server.
	appInstance := server.NewApp()

	var lock sync.RWMutex
	// Đăng ký hot reload system_config: so sánh cấu hình hiện tại trong viper với cấu hình được đẩy, chỉ merge và reload khi nội dung thay đổi.
	user_config.RegisterManagerSystemConfigHandler(func(data map[string]interface{}) {
		lock.Lock()
		defer lock.Unlock()
		current := viper.AllSettings()
		oldMqttServer := current["mqtt_server"]
		oldMqtt := current["mqtt"]
		oldUdp := current["udp"]
		oldMcp := current["mcp"]
		oldLocalMcp := current["local_mcp"]

		var doMqttServer, doMqttReload, doUdpReload, doMcpReload bool
		if data["mqtt_server"] != nil {
			if !SystemConfigEqual(data["mqtt_server"], oldMqttServer) {
				doMqttServer = true
			}
		}
		if data["mqtt"] != nil {
			if !SystemConfigEqual(data["mqtt"], oldMqtt) {
				doMqttReload = true
			}
		}
		if data["udp"] != nil {
			if udpListenChanged(data["udp"], oldUdp) {
				doUdpReload = true
			}
		}
		if data["mcp"] != nil {
			if !SystemConfigEqual(data["mcp"], oldMcp) {
				doMcpReload = true
			}
		}
		if data["local_mcp"] != nil {
			if !SystemConfigEqual(data["local_mcp"], oldLocalMcp) {
				doMcpReload = true
			}
		}

		ApplySystemConfigToViper(data)

		var wg sync.WaitGroup
		if doMqttServer {
			wg.Add(1)
			go func() {
				defer wg.Done()
				appInstance.ReloadMqttServer()
			}()
		}
		if doMqttReload || doUdpReload {
			wg.Add(1)
			go func() {
				defer wg.Done()
				appInstance.ReloadMqttUdpWithFlags(doMqttReload, doUdpReload)
			}()
		}
		if doMcpReload {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := appInstance.ReloadMCP(); err != nil {
					log.Errorf("ReloadMCP thất bại: %v", err)
				}
			}()
		}
		wg.Wait()
	})
	appInstance.Run()

	// Chặn tiến trình để lắng nghe tín hiệu thoát.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Server đã khởi động, nhấn Ctrl+C để thoát")
	<-quit

	log.Info("Đang tắt server...")

	// Dừng dịch vụ cập nhật cấu hình định kỳ.
	StopPeriodicConfigUpdate()
	if *managerEnable {
		StopManagerHTTP()
	}
	if *asrEnable {
		StopAsrServerHTTP()
	}
	if *ttsEnable {
		StopTtsServerHTTP()
	}

	log.Info("Server đã tắt")
}

func udpListenChanged(newUdpCfg interface{}, oldUdpCfg interface{}) bool {
	newListenHost, newListenPort := udpListenHostPort(newUdpCfg)
	oldListenHost, oldListenPort := udpListenHostPort(oldUdpCfg)
	if newListenHost == "" && newListenPort == 0 {
		return false
	}
	return newListenHost != oldListenHost || newListenPort != oldListenPort
}

func udpListenHostPort(cfg interface{}) (string, int) {
	if cfg == nil {
		return "", 0
	}
	type udpListen struct {
		ListenHost string `json:"listen_host"`
		ListenPort int    `json:"listen_port"`
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", 0
	}
	var parsed udpListen
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", 0
	}
	return parsed.ListenHost, parsed.ListenPort
}
