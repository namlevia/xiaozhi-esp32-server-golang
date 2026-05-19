package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server ServerConfig `json:"server"`
	Edge   EdgeConfig   `json:"edge"`
	Piper  PiperConfig  `json:"piper"`
}

type ServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Path         string `json:"path"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
}

type EdgeConfig struct {
	Voice          string `json:"voice"`
	Rate           string `json:"rate"`
	Volume         string `json:"volume"`
	Pitch          string `json:"pitch"`
	ConnectTimeout int    `json:"connect_timeout_seconds"`
	ReceiveTimeout int    `json:"receive_timeout_seconds"`
}

type PiperConfig struct {
	ModelDir        string  `json:"model_dir"`
	DefaultVoice    string  `json:"default_voice"`
	EspeakDataDir   string  `json:"espeak_data_dir"`
	Provider        string  `json:"provider"`
	ResponseFormat  string  `json:"response_format"`
	NumThreads      int     `json:"num_threads"`
	MaxNumSentences int     `json:"max_num_sentences"`
	SilenceScale    float32 `json:"silence_scale"`
	LengthScale     float32 `json:"length_scale"`
	NoiseScale      float32 `json:"noise_scale"`
	NoiseW          float32 `json:"noise_w"`
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("đọc cấu hình TTS server thất bại: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse cấu hình TTS server thất bại: %w", err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func defaultConfig() Config {
	cfg := Config{}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 9001
	}
	if c.Server.Path == "" {
		c.Server.Path = "/tts"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 60
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 60
	}
	if c.Edge.Voice == "" {
		c.Edge.Voice = "vi-VN-HoaiMyNeural"
	}
	if c.Edge.Rate == "" {
		c.Edge.Rate = "+0%"
	}
	if c.Edge.Volume == "" {
		c.Edge.Volume = "+0%"
	}
	if c.Edge.Pitch == "" {
		c.Edge.Pitch = "+0Hz"
	}
	if c.Edge.ConnectTimeout == 0 {
		c.Edge.ConnectTimeout = 10
	}
	if c.Edge.ReceiveTimeout == 0 {
		c.Edge.ReceiveTimeout = 60
	}
	if c.Piper.ModelDir == "" {
		c.Piper.ModelDir = "tts_server/tts-model"
	}
	if c.Piper.DefaultVoice == "" {
		c.Piper.DefaultVoice = "banmai"
	}
	if c.Piper.Provider == "" {
		c.Piper.Provider = "cpu"
	}
	if c.Piper.ResponseFormat == "" {
		c.Piper.ResponseFormat = "wav"
	}
	if c.Piper.NumThreads == 0 {
		c.Piper.NumThreads = 2
	}
	if c.Piper.MaxNumSentences == 0 {
		c.Piper.MaxNumSentences = 1
	}
	if c.Piper.SilenceScale == 0 {
		c.Piper.SilenceScale = 0.2
	}
	if c.Piper.LengthScale == 0 {
		c.Piper.LengthScale = 1.0
	}
	if c.Piper.NoiseScale == 0 {
		c.Piper.NoiseScale = 0.667
	}
	if c.Piper.NoiseW == 0 {
		c.Piper.NoiseW = 0.8
	}
}

func (c Config) addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c Config) readTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

func (c Config) writeTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}
