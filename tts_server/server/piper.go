package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

type PiperVoice struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	ModelPath       string  `json:"model_path"`
	ModelConfigPath string  `json:"model_config_path"`
	SampleRate      int     `json:"sample_rate"`
	Language        string  `json:"language"`
	NumSpeakers     int     `json:"num_speakers"`
	LengthScale     float32 `json:"length_scale"`
	NoiseScale      float32 `json:"noise_scale"`
	NoiseW          float32 `json:"noise_w"`
}

type piperMetadata struct {
	Audio struct {
		SampleRate int `json:"sample_rate"`
	} `json:"audio"`
	Espeak struct {
		Voice string `json:"voice"`
	} `json:"espeak"`
	NumSpeakers int `json:"num_speakers"`
	Inference   struct {
		NoiseScale  float32 `json:"noise_scale"`
		LengthScale float32 `json:"length_scale"`
		NoiseW      float32 `json:"noise_w"`
	} `json:"inference"`
	PhonemeIDMap map[string][]int `json:"phoneme_id_map"`
}

type piperRequest struct {
	Text            string  `json:"text"`
	Voice           string  `json:"voice"`
	ModelPath       string  `json:"model_path"`
	ModelConfigPath string  `json:"model_config_path"`
	ResponseFormat  string  `json:"response_format"`
	SampleRate      int     `json:"sample_rate"`
	SpeakerID       int     `json:"speaker_id"`
	Speed           float32 `json:"speed"`
	LengthScale     float32 `json:"length_scale"`
	NoiseScale      float32 `json:"noise_scale"`
	NoiseW          float32 `json:"noise_w"`
}

type piperRuntime struct {
	tts      *sherpa.OfflineTts
	voice    PiperVoice
	tokens   string
	sid      int
	speed    float32
	response string
}

var (
	piperMu       sync.Mutex
	piperRuntimes = map[string]*piperRuntime{}
)

func (s *Server) handlePiperVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	voices, err := s.discoverPiperVoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"voices": voices})
}

func (s *Server) handlePiperTTS(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Errorf("Piper TTS panic: %v", recovered)
			http.Error(w, fmt.Sprintf("Piper TTS lỗi runtime: %v", recovered), http.StatusInternalServerError)
		}
	}()
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req piperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	runtime, err := s.getPiperRuntime(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	speed := req.Speed
	if speed == 0 {
		speed = runtime.speed
	}
	if speed == 0 {
		speed = 1
	}
	audio := runtime.tts.Generate(req.Text, runtime.sid, speed)
	if audio == nil || len(audio.Samples) == 0 {
		http.Error(w, "piper generated empty audio", http.StatusInternalServerError)
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.ResponseFormat))
	if format == "" {
		format = runtime.response
	}
	if format == "pcm" {
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(float32ToPCM16(audio.Samples))
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	_, _ = w.Write(wavBytes(audio.Samples, audio.SampleRate))
}

func (s *Server) discoverPiperVoices() ([]PiperVoice, error) {
	modelDir := s.cfg.Piper.ModelDir
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return nil, fmt.Errorf("đọc thư mục model Piper thất bại: %w", err)
	}
	voices := make([]PiperVoice, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".onnx") {
			continue
		}
		modelPath := filepath.Join(modelDir, entry.Name())
		configPath := modelPath + ".json"
		voice, err := readPiperVoice(modelPath, configPath)
		if err != nil {
			continue
		}
		voices = append(voices, voice)
	}
	sort.Slice(voices, func(i, j int) bool { return voices[i].ID < voices[j].ID })
	return voices, nil
}

func (s *Server) getPiperRuntime(req piperRequest) (*piperRuntime, error) {
	voice, err := s.resolvePiperVoice(req)
	if err != nil {
		return nil, err
	}
	voice.LengthScale = firstFloat32(req.LengthScale, voice.LengthScale, s.cfg.Piper.LengthScale, 1.0)
	voice.NoiseScale = firstFloat32(req.NoiseScale, voice.NoiseScale, s.cfg.Piper.NoiseScale, 0.667)
	voice.NoiseW = firstFloat32(req.NoiseW, voice.NoiseW, s.cfg.Piper.NoiseW, 0.8)

	key := fmt.Sprintf("%s|%.4f|%.4f|%.4f", voice.ModelPath, voice.LengthScale, voice.NoiseScale, voice.NoiseW)
	piperMu.Lock()
	defer piperMu.Unlock()
	if runtime := piperRuntimes[key]; runtime != nil {
		return runtime, nil
	}

	tokens, err := ensurePiperTokens(voice.ModelConfigPath)
	if err != nil {
		return nil, fmt.Errorf("chuẩn bị tokens Piper thất bại, kiểm tra quyền ghi thư mục model hoặc mount tts-model: %w", err)
	}
	modelPath, err := ensurePiperModelMetadata(voice)
	if err != nil {
		return nil, fmt.Errorf("chuẩn bị metadata ONNX cho Piper thất bại, kiểm tra quyền ghi thư mục model hoặc mount tts-model: %w", err)
	}
	cfg := &sherpa.OfflineTtsConfig{
		Model: sherpa.OfflineTtsModelConfig{
			Vits: sherpa.OfflineTtsVitsModelConfig{
				Model:       modelPath,
				Tokens:      tokens,
				DataDir:     s.cfg.Piper.EspeakDataDir,
				NoiseScale:  voice.NoiseScale,
				NoiseScaleW: voice.NoiseW,
				LengthScale: voice.LengthScale,
			},
			NumThreads: s.cfg.Piper.NumThreads,
			Provider:   s.cfg.Piper.Provider,
		},
		MaxNumSentences: s.cfg.Piper.MaxNumSentences,
		SilenceScale:    s.cfg.Piper.SilenceScale,
	}
	if cfg.Model.NumThreads == 0 {
		cfg.Model.NumThreads = 2
	}
	if cfg.MaxNumSentences == 0 {
		cfg.MaxNumSentences = 1
	}
	if cfg.SilenceScale == 0 {
		cfg.SilenceScale = 0.2
	}
	tts := sherpa.NewOfflineTts(cfg)
	if tts == nil {
		return nil, fmt.Errorf("khởi tạo Piper model thất bại: %s", voice.ModelPath)
	}
	runtime := &piperRuntime{
		tts:      tts,
		voice:    voice,
		tokens:   tokens,
		sid:      req.SpeakerID,
		speed:    req.Speed,
		response: strings.ToLower(strings.TrimSpace(s.cfg.Piper.ResponseFormat)),
	}
	if runtime.response == "" {
		runtime.response = "wav"
	}
	piperRuntimes[key] = runtime
	return runtime, nil
}

func (s *Server) resolvePiperVoice(req piperRequest) (PiperVoice, error) {
	modelPath := strings.TrimSpace(req.ModelPath)
	configPath := strings.TrimSpace(req.ModelConfigPath)
	if modelPath != "" {
		if configPath == "" {
			configPath = modelPath + ".json"
		}
		return readPiperVoice(modelPath, configPath)
	}
	voiceID := strings.TrimSpace(req.Voice)
	if voiceID == "" {
		voiceID = s.cfg.Piper.DefaultVoice
	}
	voices, err := s.discoverPiperVoices()
	if err != nil {
		return PiperVoice{}, err
	}
	for _, voice := range voices {
		if voice.ID == voiceID || voice.Name == voiceID {
			return voice, nil
		}
	}
	return PiperVoice{}, fmt.Errorf("không tìm thấy giọng Piper: %s", voiceID)
}

func readPiperVoice(modelPath, configPath string) (PiperVoice, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return PiperVoice{}, err
	}
	var meta piperMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return PiperVoice{}, err
	}
	id := strings.TrimSuffix(filepath.Base(modelPath), ".onnx")
	voice := PiperVoice{
		ID:              id,
		Name:            id,
		ModelPath:       modelPath,
		ModelConfigPath: configPath,
		SampleRate:      meta.Audio.SampleRate,
		Language:        meta.Espeak.Voice,
		NumSpeakers:     meta.NumSpeakers,
		LengthScale:     meta.Inference.LengthScale,
		NoiseScale:      meta.Inference.NoiseScale,
		NoiseW:          meta.Inference.NoiseW,
	}
	if voice.SampleRate == 0 {
		voice.SampleRate = 22050
	}
	if voice.NumSpeakers == 0 {
		voice.NumSpeakers = 1
	}
	return voice, nil
}

func ensurePiperModelMetadata(voice PiperVoice) (string, error) {
	modelPath := voice.ModelPath
	cachePath := strings.TrimSuffix(modelPath, ".onnx") + ".sherpa.onnx"
	modelInfo, err := os.Stat(modelPath)
	if err != nil {
		return "", err
	}
	if cacheInfo, err := os.Stat(cachePath); err == nil && cacheInfo.ModTime().After(modelInfo.ModTime()) {
		return cachePath, nil
	}
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return "", err
	}
	metadata := map[string]string{
		"sample_rate":    fmt.Sprintf("%d", voice.SampleRate),
		"num_speakers":   fmt.Sprintf("%d", voice.NumSpeakers),
		"n_speakers":     fmt.Sprintf("%d", voice.NumSpeakers),
		"noise_scale":    fmt.Sprintf("%g", voice.NoiseScale),
		"noise_scale_w":  fmt.Sprintf("%g", voice.NoiseW),
		"length_scale":   fmt.Sprintf("%g", voice.LengthScale),
		"model_type":     "vits",
		"comment":        "piper",
		"language":       voice.Language,
		"add_blank":      "1",
		"frontend":       "espeak",
		"voice":          voice.Language,
		"phoneme_type":   "espeak",
		"phoneme_id_map": voice.ModelConfigPath,
		"model_config":   voice.ModelConfigPath,
	}
	patched := appendModelMetadata(data, metadata)
	if err := os.WriteFile(cachePath, patched, 0644); err != nil {
		return "", err
	}
	return cachePath, nil
}

func appendModelMetadata(model []byte, metadata map[string]string) []byte {
	out := make([]byte, 0, len(model)+512)
	out = append(out, model...)
	for key, value := range metadata {
		entry := appendProtoString(nil, 1, key)
		entry = appendProtoString(entry, 2, value)
		out = appendProtoBytes(out, 14, entry)
	}
	return out
}

func appendProtoBytes(dst []byte, field int, value []byte) []byte {
	dst = appendProtoVarint(dst, uint64(field<<3|2))
	dst = appendProtoVarint(dst, uint64(len(value)))
	dst = append(dst, value...)
	return dst
}

func appendProtoString(dst []byte, field int, value string) []byte {
	return appendProtoBytes(dst, field, []byte(value))
}

func appendProtoVarint(dst []byte, value uint64) []byte {
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

func ensurePiperTokens(configPath string) (string, error) {
	tokensPath := strings.TrimSuffix(configPath, ".json") + ".tokens.txt"
	if _, err := os.Stat(tokensPath); err == nil {
		return tokensPath, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var meta piperMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}
	if len(meta.PhonemeIDMap) == 0 {
		return "", fmt.Errorf("metadata Piper thiếu phoneme_id_map: %s", configPath)
	}
	type token struct {
		symbol string
		id     int
	}
	tokens := make([]token, 0, len(meta.PhonemeIDMap))
	for symbol, ids := range meta.PhonemeIDMap {
		if len(ids) == 0 {
			continue
		}
		tokens = append(tokens, token{symbol: symbol, id: ids[0]})
	}
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].id < tokens[j].id })
	var b strings.Builder
	for _, token := range tokens {
		b.WriteString(token.symbol)
		b.WriteByte(' ')
		b.WriteString(fmt.Sprintf("%d\n", token.id))
	}
	if err := os.WriteFile(tokensPath, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return tokensPath, nil
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func firstFloat32(values ...float32) float32 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func float32ToPCM16(samples []float32) []byte {
	out := make([]byte, len(samples)*2)
	for i, sample := range samples {
		if sample > 1 {
			sample = 1
		} else if sample < -1 {
			sample = -1
		}
		value := int16(math.Round(float64(sample * 32767)))
		binary.LittleEndian.PutUint16(out[i*2:], uint16(value))
	}
	return out
}

func wavBytes(samples []float32, sampleRate int) []byte {
	pcm := float32ToPCM16(samples)
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+len(pcm)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	return buf.Bytes()
}
