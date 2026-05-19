package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	mqtt_server "xiaozhi-esp32-server-golang/internal/app/mqtt_server"
	log "xiaozhi-esp32-server-golang/logger"
)

// Khởi tạo ứng dụng.
func Init(configFile string) error {
	err := initConfig(configFile)
	if err != nil {
		return err
	}

	err = initLog()
	if err != nil {
		return err
	}

	return nil
}

func initLog() error {
	// Không kiểm tra cấu hình stdout nữa, thống nhất ghi ra file.
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
	logrus.SetOutput(writer)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000", // Định dạng thời gian, kèm mili giây.
		ForceColors:     false,                     // Không dùng màu khi ghi file.
	})

	// Tắt báo cáo caller mặc định, dùng field caller tùy chỉnh.
	logrus.SetReportCaller(false)
	logLevel, _ := logrus.ParseLevel(viper.GetString("log.level"))
	logrus.SetLevel(logLevel)

	return nil

}

func initConfig(configFile string) error {
	basePath, file := filepath.Split(configFile)

	// Lấy tên file và phần mở rộng.
	fileName, fileExt := func(file string) (string, string) {
		if pos := strings.LastIndex(file, "."); pos != -1 {
			return file[:pos], strings.ToLower(file[pos+1:])
		}
		return file, ""
	}(file)

	// Đặt tên file cấu hình, không gồm phần mở rộng.
	viper.SetConfigName(fileName)
	viper.AddConfigPath(basePath)

	// Đặt loại cấu hình theo phần mở rộng file.
	switch fileExt {
	case "json":
		viper.SetConfigType("json")
	case "yaml", "yml":
		viper.SetConfigType("yaml")
	default:
		return fmt.Errorf("loại file cấu hình không được hỗ trợ: %s", fileExt)
	}

	return viper.ReadInConfig()
}

func main() {
	// Phân tích tham số dòng lệnh.
	configFile := flag.String("c", "config/mqtt_config.json", "đường dẫn file cấu hình")
	flag.Parse()

	if *configFile == "" {
		fmt.Println("Đường dẫn file cấu hình không được để trống")
		return
	}

	// Khởi tạo cấu hình và log.
	err := Init(*configFile)
	if err != nil {
		fmt.Printf("Khởi tạo thất bại: %v\n", err)
		return
	}

	// Khởi động MQTT server.
	err = mqtt_server.StartMqttServer()
	if err != nil {
		log.Errorf("Khởi động MQTT server thất bại: %v", err)
		return
	}

	fmt.Println("MQTT server đã khởi động")

	// Chặn tiến trình để lắng nghe tín hiệu thoát.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Info("MQTT server đã khởi động, nhấn Ctrl+C để thoát")
	<-quit

	log.Info("Đang tắt MQTT server...")
	log.Info("MQTT server đã tắt")
}
