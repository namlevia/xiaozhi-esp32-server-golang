package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "xiaozhi-esp32-server-golang/logger"
)

const (
	liteASRAssetsURL   = "https://github.com/namlevia/Xiaozhi-Esp32-Server-Go-Vi/releases/latest/download/xiaozhi_models_asr_vi.tar.gz"
	litePiperAssetsURL = "https://github.com/namlevia/Xiaozhi-Esp32-Server-Go-Vi/releases/latest/download/xiaozhi_models_piper_vi.tar.gz"
)

type liteASRConfig struct {
	Recognition struct {
		EncoderPath string `json:"encoder_path"`
		DecoderPath string `json:"decoder_path"`
		JoinerPath  string `json:"joiner_path"`
		TokensPath  string `json:"tokens_path"`
	} `json:"recognition"`
}

type liteTTSConfig struct {
	Piper struct {
		ModelDir      string `json:"model_dir"`
		EspeakDataDir string `json:"espeak_data_dir"`
	} `json:"piper"`
}

func EnsureAIOAssets(asrConfigPath, ttsConfigPath string, asrEnabled, ttsEnabled bool) {
	if asrEnabled {
		if asrConfigPath == "" {
			asrConfigPath = "asr_server.json"
		}
		if err := ensureASRAssets(asrConfigPath); err != nil {
			log.Warnf("Không thể chuẩn bị model ASR cho bản Lite: %v", err)
			log.Warn("Nếu máy không có mạng hoặc GitHub bị chặn, hãy tải bản Full Offline để chạy không cần tải thêm model")
		}
	}
	if ttsEnabled {
		if ttsConfigPath == "" {
			ttsConfigPath = "tts_server.json"
		}
		if err := ensurePiperAssets(ttsConfigPath); err != nil {
			log.Warnf("Không thể chuẩn bị model Piper/espeak cho bản Lite: %v", err)
			log.Warn("Edge Offline vẫn có thể dùng nếu không cần Piper; nếu cần chạy offline hoàn toàn hãy tải bản Full Offline")
		}
	}
}

func ensureASRAssets(configPath string) error {
	var cfg liteASRConfig
	if err := readJSONConfig(configPath, &cfg); err != nil {
		return err
	}
	paths := []string{
		cfg.Recognition.EncoderPath,
		cfg.Recognition.DecoderPath,
		cfg.Recognition.JoinerPath,
		cfg.Recognition.TokensPath,
	}
	if allFilesExist(paths) {
		return nil
	}
	log.Info("Bản Lite thiếu model ASR Vietnamese, đang tải model lần đầu. Quá trình này có thể mất vài phút tùy tốc độ mạng...")
	return downloadAndExtractTarGZ(liteASRAssetsURL, ".")
}

func ensurePiperAssets(configPath string) error {
	var cfg liteTTSConfig
	if err := readJSONConfig(configPath, &cfg); err != nil {
		return err
	}
	modelDir := strings.TrimSpace(cfg.Piper.ModelDir)
	if modelDir == "" {
		modelDir = "tts-model"
	}
	espeakDataDir := strings.TrimSpace(cfg.Piper.EspeakDataDir)
	if espeakDataDir == "" {
		espeakDataDir = "espeak-ng-data"
	}
	paths := []string{
		filepath.Join(modelDir, "ngochuyen.onnx"),
		filepath.Join(modelDir, "ngochuyen.onnx.json"),
		filepath.Join(modelDir, "adam1.onnx"),
		filepath.Join(modelDir, "adam1.onnx.json"),
		filepath.Join(espeakDataDir, "phontab"),
	}
	if allFilesExist(paths) {
		return nil
	}
	log.Info("Bản Lite thiếu model Piper hoặc dữ liệu espeak-ng, đang tải lần đầu. Quá trình này có thể mất vài phút tùy tốc độ mạng...")
	return downloadAndExtractTarGZ(litePiperAssetsURL, ".")
}

func readJSONConfig(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("đọc cấu hình %s thất bại: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse cấu hình %s thất bại: %w", path, err)
	}
	return nil
}

func allFilesExist(paths []string) bool {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			return false
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() == 0 {
			return false
		}
	}
	return true
}

func downloadAndExtractTarGZ(url, destDir string) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("tải %s thất bại: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("tải %s thất bại: HTTP %d", url, resp.StatusCode)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("đọc gzip thất bại: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	cleanDest, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("đọc tar thất bại: %w", err)
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("archive có đường dẫn không an toàn: %s", header.Name)
		}
		target := filepath.Join(cleanDest, name)
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) && target != cleanDest {
			return fmt.Errorf("archive có đường dẫn vượt thư mục đích: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, tr)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	log.Info("Đã tải và giải nén asset cho bản Lite")
	return nil
}
