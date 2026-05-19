package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/difyz9/edge-tts-go/pkg/communicate"
	"github.com/gorilla/websocket"
)

type Server struct {
	cfg      Config
	upgrader websocket.Upgrader
}

func Setup(configPath string) (http.Handler, string, time.Duration, time.Duration, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, "", 0, 0, err
	}

	s := &Server{
		cfg: cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc(cfg.Server.Path, s.handleTTS)
	mux.HandleFunc("/piper/voices", s.handlePiperVoices)
	mux.HandleFunc("/piper/tts", s.handlePiperTTS)
	return mux, cfg.addr(), cfg.readTimeout(), cfg.writeTimeout(), nil
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok","service":"tts_server"}`))
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("Nâng cấp WebSocket TTS thất bại: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			log.Debugf("Đọc request TTS WebSocket kết thúc: %v", err)
			return
		}
		if messageType != websocket.TextMessage {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "text message required"), time.Now().Add(time.Second))
			return
		}

		text := strings.TrimSpace(string(data))
		if text == "" {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "empty text"), time.Now().Add(time.Second))
			return
		}

		if err := s.synthesize(r.Context(), conn, text); err != nil {
			log.Warnf("Tổng hợp Edge TTS thất bại: %v", err)
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()), time.Now().Add(time.Second))
			return
		}
		return
	}
}

func (s *Server) synthesize(ctx context.Context, conn *websocket.Conn, text string) error {
	comm, err := communicate.NewCommunicate(
		text,
		s.cfg.Edge.Voice,
		s.cfg.Edge.Rate,
		s.cfg.Edge.Volume,
		s.cfg.Edge.Pitch,
		"",
		s.cfg.Edge.ConnectTimeout,
		s.cfg.Edge.ReceiveTimeout,
	)
	if err != nil {
		return fmt.Errorf("tạo Edge communicator thất bại: %w", err)
	}

	chunkChan, errChan := comm.Stream(ctx)
	for chunk := range chunkChan {
		if chunk.Type != "audio" || len(chunk.Data) == 0 {
			continue
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, chunk.Data); err != nil {
			return fmt.Errorf("gửi audio TTS qua WebSocket thất bại: %w", err)
		}
	}
	if err := <-errChan; err != nil {
		return fmt.Errorf("stream Edge TTS thất bại: %w", err)
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"), time.Now().Add(time.Second))
	return nil
}
