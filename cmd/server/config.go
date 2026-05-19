package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"xiaozhi-esp32-server-golang/internal/app/server/auth"
	redisdb "xiaozhi-esp32-server-golang/internal/db/redis"
	user_config "xiaozhi-esp32-server-golang/internal/domain/config"

	log "xiaozhi-esp32-server-golang/logger"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/mitchellh/hashstructure/v2"
	logrus "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

// Biến toàn cục dùng để điều khiển cập nhật định kỳ.
var (
	configUpdateTicker *time.Ticker
	configUpdateStop   chan struct{}
	configUpdateWg     sync.WaitGroup
)

func Init(configFile string) error {
	//init config
	err := initConfig(configFile)
	if err != nil {
		fmt.Printf("Khởi tạo cấu hình lỗi: %+v", err)
		os.Exit(1)
		return err
	}

	//init log
	initLog()

	// Khởi tạo hệ thống cấu hình, bao gồm kết nối WebSocket.
	// Không đăng ký riêng ApplySystemConfigToViper ở đây, vì nó sẽ chạy trước callback trong main và khiến cấu hình hiện tại đọc trong main đã là cấu hình mới sau merge; việc merge phải nằm trong callback của main, sau khi đọc current và so sánh.
	ctx := context.Background()
	if err := user_config.InitConfigSystem(ctx); err != nil {
		fmt.Printf("Khởi tạo hệ thống cấu hình thất bại: %v\n", err)
	}

	// Lấy cấu hình từ API và cập nhật.
	if err := updateConfigFromAPI(); err != nil {
		fmt.Printf("Lấy cấu hình từ API thất bại, dùng cấu hình cục bộ: %v\n", err)
	}

	// Khởi động cập nhật cấu hình định kỳ.
	startPeriodicConfigUpdate()

	//init vad
	initVad()

	//init redis
	initRedis()

	// Module memory dùng lazy loading, tự khởi tạo khi sử dụng nên không cần khởi tạo tường minh.

	//init auth
	err = initAuthManager()
	if err != nil {
		fmt.Printf("Khởi tạo auth manager lỗi: %+v", err)
		os.Exit(1)
		return err
	}

	return nil
}

// startPeriodicConfigUpdate khởi động cập nhật cấu hình định kỳ.
func startPeriodicConfigUpdate() {
	// Lấy khoảng thời gian cập nhật từ cấu hình, mặc định 5 phút.
	updateInterval := viper.GetDuration("config_provider.update_interval")
	if updateInterval <= 0 {
		updateInterval = 30 * time.Second
	}

	// Kiểm tra có bật cập nhật định kỳ hay không.
	if !viper.GetBool("config_provider.enable_periodic_update") {
		log.Info("Cập nhật cấu hình định kỳ đã tắt")
		return
	}

	configUpdateStop = make(chan struct{})
	configUpdateTicker = time.NewTicker(updateInterval)

	configUpdateWg.Add(1)
	go func() {
		defer configUpdateWg.Done()
		defer configUpdateTicker.Stop()

		for {
			select {
			case <-configUpdateTicker.C:
				if err := updateConfigFromAPI(); err != nil {
					log.Warnf("Cập nhật cấu hình định kỳ thất bại: %v", err)
				} else {
					//log.Debug("Cập nhật cấu hình định kỳ thành công")
				}
			case <-configUpdateStop:
				log.Info("Cập nhật cấu hình định kỳ đã dừng")
				return
			}
		}
	}()

	log.Infof("Cập nhật cấu hình định kỳ đã khởi động, khoảng cập nhật: %v", updateInterval)
}

// StopPeriodicConfigUpdate dừng cập nhật cấu hình định kỳ.
func StopPeriodicConfigUpdate() {
	if configUpdateStop != nil {
		close(configUpdateStop)
		configUpdateWg.Wait()
		logrus.Info("Cập nhật cấu hình định kỳ đã dừng")
	}
}

func initConfig(configFile string) error {
	viper.SetConfigFile(configFile)

	// Đọc file cấu hình.
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

// ApplySystemConfigToViper merge cấu hình hệ thống vào viper, dùng cho cập nhật realtime system_config được đẩy qua WebSocket; callback không trả về giá trị.
func ApplySystemConfigToViper(data map[string]interface{}) {
	if err := viper.MergeConfigMap(data); err != nil {
		log.Warnf("Merge cấu hình được đẩy vào viper thất bại: %v", err)
		return
	}
	log.Info("Đã merge cấu hình hệ thống được đẩy qua WebSocket vào viper")
}

// SystemConfigEqual so sánh hai cấu hình hệ thống có tương đương về ngữ nghĩa hay không, dùng fingerprint hashstructure và không phụ thuộc thứ tự key trong map.
func SystemConfigEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		log.Debugf("[SystemConfigEqual] Kết quả: true (cả hai đều nil)")
		return true
	}
	if a == nil || b == nil {
		log.Debugf("[SystemConfigEqual] Kết quả: false (một bên là nil)")
		return false
	}
	ha, err1 := hashstructure.Hash(a, hashstructure.FormatV2, nil)
	hb, err2 := hashstructure.Hash(b, hashstructure.FormatV2, nil)
	if err1 != nil || err2 != nil {
		log.Debugf("[SystemConfigEqual] Kết quả: false (hash thất bại err1=%v err2=%v)", err1, err2)
		return false
	}
	equal := ha == hb
	log.Debugf("[SystemConfigEqual] Kết quả: %t (ha=%d hb=%d), a: %+v, b: %+v", equal, ha, hb, a, b)
	return equal
}

// updateConfigFromAPI lấy cấu hình từ API và cập nhật vào viper.
// Hàm sẽ tiếp tục retry cho đến khi thành công mới trả về.
func updateConfigFromAPI() error {
	configProviderType := viper.GetString("config_provider.type")
	retryInterval := 10 * time.Second // Khoảng cách giữa các lần retry.
	retryCount := 0

	for {
		// Lấy provider cấu hình theo loại được cấu hình.
		configProvider, err := user_config.GetProvider(configProviderType)
		if err != nil {
			retryCount++
			log.Warnf("Lấy provider cấu hình thất bại (retry lần %d): %v, retry sau %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// Tạo context.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Lấy chuỗi JSON cấu hình hệ thống.
		configJSON, err := configProvider.GetSystemConfig(ctx)
		cancel()

		if err != nil {
			retryCount++
			log.Warnf("Lấy cấu hình hệ thống thất bại (retry lần %d): %v, retry sau %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		if configJSON == "" {
			// Cấu hình rỗng được xem là thành công, vì service có thể trả về cấu hình rỗng.
			if retryCount > 0 {
				log.Infof("Lấy cấu hình thành công (cấu hình rỗng, sau %d lần retry)", retryCount)
			}
			return nil
		}

		// Parse JSON thành map.
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &configMap); err != nil {
			retryCount++
			log.Warnf("Parse JSON cấu hình thất bại (retry lần %d): %v, retry sau %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		//log.Debugf("Load config from API: %+v", configMap)

		// Dùng viper.MergeConfigMap để ghi vào viper.
		if err := viper.MergeConfigMap(configMap); err != nil {
			retryCount++
			log.Warnf("Merge cấu hình vào viper thất bại (retry lần %d): %v, retry sau %v", retryCount, err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}

		// Thành công.
		if retryCount > 0 {
			log.Infof("Lấy cấu hình thành công (sau %d lần retry)", retryCount)
		} else {
			log.Debug("Lấy cấu hình thành công")
		}
		return nil
	}
}

func initLog() error {
	// Ghi ra file.
	binPath, _ := os.Executable()
	baseDir := filepath.Dir(binPath)
	logPath := fmt.Sprintf("%s/%s%s", baseDir, viper.GetString("log.path"), viper.GetString("log.file"))
	/* Các hàm liên quan đến xoay vòng log:
	`WithLinkName` tạo liên kết mềm tới file log mới nhất.
	`WithRotationTime` đặt chu kỳ cắt log.
	WithMaxAge và WithRotationCount chỉ nên dùng một trong hai.
		`WithMaxAge` đặt thời gian lưu tối đa trước khi dọn file.
		`WithRotationCount` đặt số file lưu tối đa trước khi dọn.
	*/
	// Cấu hình bên dưới xoay log theo ngày và giữ số bản ghi theo log.max_age.
	writer, err := rotatelogs.New(
		logPath+".%Y%m%d",
		rotatelogs.WithLinkName(logPath),
		rotatelogs.WithRotationCount(uint(viper.GetInt("log.max_age"))),
		rotatelogs.WithRotationTime(time.Duration(86400)*time.Second),
	)
	if err != nil {
		fmt.Printf("Khởi tạo log lỗi: %v\n", err)
		os.Exit(1)
		return err
	}

	// Quyết định đầu ra theo cấu hình.
	if viper.GetBool("log.stdout") {
		// Ghi đồng thời ra file và stdout.
		multiWriter := io.MultiWriter(writer, os.Stdout)
		logrus.SetOutput(multiWriter)
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000", // Định dạng thời gian, kèm mili giây.
			ForceColors:     true,                      // Bật màu khi ghi stdout.
		})
	} else {
		// Chỉ ghi ra file.
		logrus.SetOutput(writer)
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000", // Định dạng thời gian, kèm mili giây.
			ForceColors:     false,                     // Không dùng màu khi ghi file.
		})
	}

	// Tắt báo cáo caller mặc định, dùng field caller tùy chỉnh.
	logrus.SetReportCaller(false)
	logLevel, _ := logrus.ParseLevel(viper.GetString("log.level"))
	logrus.SetLevel(logLevel)

	return nil
}

func initVad() error {
	log.Infof("Bắt đầu khởi tạo module VAD...")
	vadProvider := viper.GetString("vad.provider")
	log.Infof("Provider VAD: %s", vadProvider)

	// VAD dùng lazy loading và sẽ tự khởi tạo qua resource pool toàn cục khi được dùng lần đầu.
	log.Infof("Module VAD sẽ dùng lazy loading và tự khởi tạo khi được dùng lần đầu")
	return nil
}

func initRedis() error {
	// Khởi tạo module Redis thống nhất.
	redisConfig := &redisdb.Config{
		Host:     viper.GetString("redis.host"),
		Port:     viper.GetInt("redis.port"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}

	err := redisdb.Init(redisConfig)
	if err != nil {
		fmt.Printf("Khởi tạo Redis lỗi: %v\n", err)
		return err
	}

	return nil
}

func initAuthManager() error {
	return auth.Init()
}
