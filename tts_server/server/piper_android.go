//go:build android

package server

import "net/http"

func (s *Server) handlePiperVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"Piper native TTS chưa hỗ trợ trên Android"}`))
}

func (s *Server) handlePiperTTS(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Piper native TTS chưa hỗ trợ trên Android", http.StatusNotImplemented)
}
