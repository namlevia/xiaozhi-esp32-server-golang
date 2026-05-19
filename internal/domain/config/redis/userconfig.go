package redis_config

import (
	"context"
	"encoding/json"
	"fmt"

	log "xiaozhi-esp32-server-golang/logger"

	i_redis "xiaozhi-esp32-server-golang/internal/db/redis"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// RedisUserConfigProvider là provider cấu hình người dùng Redis.
// Triển khai interface UserConfigProvider, hỗ trợ lưu trữ Redis.
type RedisUserConfigProvider struct {
	UserConfig
}

type UserConfig struct {
	redisInstance *redis.Client
	prefix        string
}

// NewRedisUserConfigProvider tạo provider cấu hình người dùng Redis.
// config: map tham số config, gồm host, port, password, db, prefix và các field khác.
func NewRedisUserConfigProvider(config interface{}) (*RedisUserConfigProvider, error) {
	provider := &RedisUserConfigProvider{
		UserConfig: UserConfig{
			redisInstance: i_redis.GetClient(),
			prefix:        viper.GetString("redis.key_prefix"),
		},
	}

	log.Log().Info("Khởi tạo provider cấu hình người dùng Redis thành công")
	return provider, nil
}

func (u *UserConfig) GetUserConfig(ctx context.Context, userID string) (types.UConfig, error) {
	redisConfig := map[string]string{}

	if u.redisInstance != nil {
		key := u.GetUserConfigKey(userID)

		// hgetall lấy toàn bộ field
		var err error
		redisConfig, err = u.redisInstance.HGetAll(ctx, key).Result()
		if err != nil {
			return types.UConfig{}, err
		}
	}

	ret := types.UConfig{
		SystemPrompt:    u.getSystemPrompt(ctx, userID),
		MemoryMode:      "short",
		SpeakerChatMode: "off",
	}
	// Chuyển UserConfig thành cấu trúc UConfig
	kv := map[string]string{
		"llm":    "",
		"asr":    "",
		"tts":    "",
		"memory": "",
	}
	for k, _ := range kv {
		var redisPerConfig map[string]interface{}
		if rv, ok := redisConfig[k]; ok {
			if rv != "" {
				err := json.Unmarshal([]byte(rv), &redisPerConfig)
				if err != nil {
					log.Log().Errorf("redis config unmarshal error: %+v", err)
				}
			}
		}

		if k == "llm" {
			config, err := u.getLlmConfig(ctx, redisPerConfig)
			if err != nil {
				return types.UConfig{}, err
			}
			ret.Llm = config
		} else if k == "tts" {
			config, err := u.getTtsConfig(ctx, redisPerConfig)
			if err != nil {
				return types.UConfig{}, err
			}
			ret.Tts = config
		} else if k == "asr" {
			config, err := u.getAsrConfig(ctx, redisPerConfig)
			if err != nil {
				return types.UConfig{}, err
			}
			ret.Asr = config
		} else if k == "memory" {
			config, err := u.getMemoryConfig(ctx, redisPerConfig)
			if err != nil {
				return types.UConfig{}, err
			}
			ret.Memory = config
		}
	}
	ret.Vad = u.getVadConfig(ctx)

	log.Log().Infof("userconfig: %+v", ret)
	return ret, nil
}

func (u *UserConfig) getVadConfig(ctx context.Context) types.VadConfig {
	provider := viper.GetString("vad.provider")
	return types.VadConfig{
		Provider: provider,
		Config:   viper.GetStringMap(fmt.Sprintf("vad.%s", provider)),
	}
}

func (u *UserConfig) getConfigByType(ctx context.Context, config map[string]interface{}, prefix string) (string, map[string]interface{}, error) {
	provider := viper.GetString(prefix + ".provider")
	if _, ok := config[provider]; !ok {
		if iProvider, ok := config["provider"].(string); ok {
			provider = iProvider
		}
	}

	commonConfig := viper.GetStringMap(prefix + "." + provider)

	for k, v := range config {
		if k == "provider" {
			continue
		}
		commonConfig[k] = v
	}
	return provider, commonConfig, nil
}

func (u *UserConfig) getLlmConfig(ctx context.Context, config map[string]interface{}) (types.LlmConfig, error) {
	provider, commonConfig, err := u.getConfigByType(ctx, config, "llm")
	if err != nil {
		return types.LlmConfig{}, err
	}
	return types.LlmConfig{
		Provider: provider,
		Config:   commonConfig,
	}, nil
}
func (u *UserConfig) getAsrConfig(ctx context.Context, config map[string]interface{}) (types.AsrConfig, error) {
	provider, commonConfig, err := u.getConfigByType(ctx, config, "asr")
	if err != nil {
		return types.AsrConfig{}, err
	}
	return types.AsrConfig{
		Provider: provider,
		Config:   commonConfig,
	}, nil
}
func (u *UserConfig) getTtsConfig(ctx context.Context, config map[string]interface{}) (types.TtsConfig, error) {
	provider, commonConfig, err := u.getConfigByType(ctx, config, "tts")
	if err != nil {
		return types.TtsConfig{}, err
	}
	return types.TtsConfig{
		Provider: provider,
		Config:   commonConfig,
	}, nil
}

func (u *UserConfig) getMemoryConfig(ctx context.Context, config map[string]interface{}) (types.MemoryConfig, error) {
	provider, commonConfig, err := u.getConfigByType(ctx, config, "memory")
	if err != nil {
		return types.MemoryConfig{}, err
	}
	return types.MemoryConfig{
		Provider: provider,
		Config:   commonConfig,
	}, nil
}

func (u *UserConfig) GetUserConfigKey(deviceId string) string {
	return fmt.Sprintf("%s:userconfig:%s", u.prefix, deviceId)
}

// getSystemPromptKey tạo Redis key system prompt tương ứng với thiết bị.
func (u *UserConfig) getSystemPrompt(ctx context.Context, deviceID string) string {
	key := fmt.Sprintf("%s:llm:system:%s", u.prefix, deviceID)

	configPrompt := viper.GetString("system_prompt")

	if u.redisInstance != nil {
		systemPrompt, err := u.redisInstance.Get(ctx, key).Result()
		if err != nil {
			return configPrompt
		}
		return systemPrompt
	}
	return configPrompt
}

// Lấy config mqtt, mqtt_server, udp, ota, vision.
func (u *UserConfig) GetSystemConfig(ctx context.Context) (string, error) {
	// Mặc định không override
	return "", nil
}

// SwitchDeviceRoleByName không hỗ trợ chuyển role thiết bị ở mode Redis.
func (u *UserConfig) SwitchDeviceRoleByName(ctx context.Context, deviceID string, roleName string) (string, error) {
	return "", fmt.Errorf("Provider cấu hình redis không hỗ trợ chuyển role thiết bị theo tên role")
}

// RestoreDeviceDefaultRole không hỗ trợ khôi phục role mặc định ở mode Redis.
func (u *UserConfig) RestoreDeviceDefaultRole(ctx context.Context, deviceID string) error {
	return fmt.Errorf("Provider cấu hình redis không hỗ trợ khôi phục role mặc định của thiết bị")
}

func (u *UserConfig) NotifyDeviceEvent(ctx context.Context, eventType string, eventData map[string]interface{}) {
	// Triển khai logic thông báo sự kiện thiết bị
	return
}

func (u *UserConfig) RegisterMessageEventHandler(ctx context.Context, deviceID string, handler types.EventHandler) {
	// Triển khai logic xử lý sự kiện message
	return
}

// Init khởi tạo provider cấu hình Redis.
func Init(ctx context.Context) error {
	log.Log().Info("Redis config provider initialized successfully")
	return nil
}

// Close đóng provider cấu hình Redis và cleanup tài nguyên.
func Close() error {
	log.Log().Info("Redis config provider closed")
	return nil
}

// IsConnected kiểm tra provider cấu hình Redis đã kết nối hay chưa.
func IsConnected() bool {
	// Trạng thái kết nối Redis do Redis client toàn cục quản lý
	return true
}
