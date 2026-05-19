package doubao

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/doubaoapi"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

var wsDialer = websocket.Dialer{
	ReadBufferSize:   16384,
	WriteBufferSize:  16384,
	HandshakeTimeout: 45 * time.Second,
}

var doubaoWSRequestHeader = []byte{0x11, 0x10, 0x11, 0x00}

type DoubaoWSProvider struct {
	AppID       string
	AccessToken string
	Model       string
	ResourceID  string
	Voice       string
	WSURL       string
}

type doubaoWSAttemptResult struct {
	outputChan chan []byte
	err        error
}

func NewDoubaoWSProvider(config map[string]interface{}) *DoubaoWSProvider {
	appID, _ := config["appid"].(string)
	accessToken, _ := config["access_token"].(string)
	model, _ := config["model"].(string)
	resourceID, _ := config["resource_id"].(string)
	voice, _ := config["voice"].(string)
	wsURL, _ := config["ws_url"].(string)
	if strings.TrimSpace(wsURL) == "" {
		if wsHost, ok := config["ws_host"].(string); ok && strings.TrimSpace(wsHost) != "" {
			host := strings.TrimSpace(wsHost)
			if !strings.Contains(host, "://") {
				wsURL = "wss://" + strings.TrimLeft(host, "/")
			} else {
				wsURL = host
			}
			wsURL = strings.TrimRight(wsURL, "/") + "/api/v3/tts/unidirectional/stream"
		}
	}
	return &DoubaoWSProvider{
		AppID:       appID,
		AccessToken: accessToken,
		Model:       normalizeDoubaoModel(model),
		ResourceID:  strings.TrimSpace(resourceID),
		Voice:       strings.TrimSpace(voice),
		WSURL:       normalizeDoubaoWSURL(wsURL),
	}
}

func normalizeDoubaoWSURL(raw string) string {
	wsURL := normalizeDoubaoTTSURL(raw, defaultDoubaoWSURL)
	wsURL = strings.TrimSpace(wsURL)
	if strings.HasSuffix(wsURL, "/api/v3/tts/unidirectional") {
		return wsURL + "/stream"
	}
	return wsURL
}

func (p *DoubaoWSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	frames := make([][]byte, 0, 32)
	for frame := range outputChan {
		if len(frame) > 0 {
			frames = append(frames, frame)
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("Doubao WebSocket TTS trả vềaudiolàrỗng")
	}
	return frames, nil
}

func (p *DoubaoWSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	voice := strings.TrimSpace(p.Voice)
	if voice == "" {
		return nil, fmt.Errorf("Doubao WebSocket TTS noi_dung voice")
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	derivedResolved, err := resolveDoubaoTTSModel(p.Model, voice)
	if err != nil {
		return nil, err
	}
	explicitResourceID := strings.TrimSpace(p.ResourceID)
	if sampleRate <= 0 {
		sampleRate = defaultDoubaoSampleHz
	}
	tryResolved := buildDoubaoWSAttemptModels(derivedResolved, explicitResourceID, voice)
	attemptedResources := make([]string, 0, len(tryResolved))
	attemptErrors := make([]error, 0, len(tryResolved))

	for idx, candidate := range tryResolved {
		attemptedResources = append(attemptedResources, candidate.ResourceID)

		outputChan, attemptErr := p.textToSpeechStreamWithModel(ctx, text, sampleRate, frameDuration, candidate)
		if attemptErr == nil {
			return outputChan, nil
		}
		attemptErrors = append(attemptErrors, attemptErr)
		if idx == len(tryResolved)-1 || !isDoubaoRetryableResourceError(attemptErr) {
			return nil, summarizeDoubaoWSAttemptError(voice, tryResolved[0], attemptedResources, attemptErrors)
		}
		log.Warnf("Doubao WebSocket TTS tài nguyênnoi_dung，noi_dung: voice=%s from=%s to=%s", voice, candidate.ResourceID, tryResolved[idx+1].ResourceID)
	}

	return nil, fmt.Errorf("Doubao WebSocket TTS chưanoi_dungtớikhả dụngnoi_dungtài nguyênnoi_dung")
}

func (p *DoubaoWSProvider) textToSpeechStreamWithModel(ctx context.Context, text string, sampleRate int, frameDuration int, resolved resolvedTTSModel) (chan []byte, error) {
	voice := strings.TrimSpace(p.Voice)
	reqBody := newDoubaoWSPayload(text, voice, sampleRate, resolved.RequestModel)
	requestFrame, err := buildDoubaoWSBinaryRequest(reqBody)
	if err != nil {
		return nil, fmt.Errorf("DựngDoubao WebSocket TTS request thất bại: %w", err)
	}

	headers := doubaoapi.NewTTSWebsocketHeaders(p.AppID, p.AccessToken, resolved.ResourceID, doubaoapi.NewConnectID())
	conn, resp, err := wsDialer.DialContext(ctx, p.WSURL, headers)
	if err != nil {
		return nil, formatDoubaoWSConnectError(err, resp)
	}

	outputChan := make(chan []byte, 1000)
	pipeReader, pipeWriter := io.Pipe()
	attemptResult := make(chan doubaoWSAttemptResult, 1)
	startTs := time.Now().UnixMilli()
	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, defaultDoubaoAudioFmt, sampleRate)
	if err != nil {
		_ = conn.Close()
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		close(outputChan)
		return nil, fmt.Errorf("tạoDoubao WebSocket audiodecoderthất bại: %w", err)
	}
	go func() {
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Doubao WebSocket Decode audio thất bại: %v", err)
		}
	}()

	go func() {
		defer conn.Close()
		defer pipeWriter.Close()

		if err := conn.WriteMessage(websocket.BinaryMessage, requestFrame); err != nil {
			trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: fmt.Errorf("gửiDoubao WebSocket TTS request thất bại: %w", err)})
			_ = pipeWriter.CloseWithError(fmt.Errorf("gửiDoubao WebSocket TTS request thất bại: %w", err))
			return
		}

		audioReceived := false
		for {
			_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					if !audioReceived {
						trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: fmt.Errorf("Doubao WebSocket TTS Kết nối đã đóngnoi_dungchưaNhậnaudio")})
					}
					return
				}
				trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: fmt.Errorf("đọcDoubao WebSocket TTS responsethất bại: %w", err)})
				_ = pipeWriter.CloseWithError(fmt.Errorf("đọcDoubao WebSocket TTS responsethất bại: %w", err))
				return
			}
			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}
			log.Debugf("Doubao WebSocket TTS Nhậnmessage: ws_type=%d payload_len=%d", messageType, len(payload))
			audioChunk, isLast, err := parseDoubaoWSMessage(messageType, payload)
			if err != nil {
				trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: err})
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if len(audioChunk) > 0 {
				if !audioReceived {
					audioReceived = true
					trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{outputChan: outputChan})
				}
				if _, err := pipeWriter.Write(audioChunk); err != nil {
					trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: err})
					_ = pipeWriter.CloseWithError(err)
					return
				}
			}
			if isLast {
				if !audioReceived {
					trySendDoubaoWSAttemptResult(attemptResult, doubaoWSAttemptResult{err: fmt.Errorf("Doubao WebSocket TTS chưaNhậnhợp lệaudio")})
				}
				return
			}
		}
	}()

	select {
	case result := <-attemptResult:
		if result.err != nil {
			return nil, result.err
		}
		return result.outputChan, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func trySendDoubaoWSAttemptResult(ch chan doubaoWSAttemptResult, result doubaoWSAttemptResult) {
	select {
	case ch <- result:
	default:
	}
}

func (p *DoubaoWSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
	}
	if model, ok := voiceConfig["model"].(string); ok && strings.TrimSpace(model) != "" {
		p.Model = normalizeDoubaoModel(model)
	}
	if resourceID, ok := voiceConfig["resource_id"].(string); ok {
		p.ResourceID = strings.TrimSpace(resourceID)
	}
	return nil
}

func (p *DoubaoWSProvider) Close() error {
	return nil
}

func (p *DoubaoWSProvider) IsValid() bool {
	return p != nil
}

func newDoubaoWSPayload(text, speaker string, sampleRate int, requestModel string) doubaoTTSV3Request {
	return newDoubaoTTSV3Request(text, speaker, sampleRate, requestModel)
}

func buildDoubaoWSBinaryRequest(req any) ([]byte, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("sequencenoi_dungDoubao WebSocket TTS request thất bại: %w", err)
	}
	compressed, err := gzipCompressDoubao(payload)
	if err != nil {
		return nil, fmt.Errorf("noi_dungDoubao WebSocket TTS request thất bại: %w", err)
	}

	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(compressed)))

	frame := make([]byte, 0, len(doubaoWSRequestHeader)+len(sizeBuf)+len(compressed))
	frame = append(frame, doubaoWSRequestHeader...)
	frame = append(frame, sizeBuf...)
	frame = append(frame, compressed...)
	return frame, nil
}

func parseDoubaoWSMessage(messageType int, payload []byte) ([]byte, bool, error) {
	if len(payload) == 0 {
		return nil, false, nil
	}

	if looksLikeJSONPayload(payload) {
		var directErr map[string]any
		if err := json.Unmarshal(payload, &directErr); err == nil {
			if errMsg, ok := directErr["error"].(string); ok && strings.TrimSpace(errMsg) != "" {
				return nil, false, fmt.Errorf("%s", strings.TrimSpace(errMsg))
			}
		}
	}

	if messageType == websocket.TextMessage || looksLikeJSONPayload(payload) {
		event, err := parseDoubaoTTSStreamEvent(payload)
		if err != nil || event == nil {
			return nil, false, err
		}
		chunk, err := decodeDoubaoTTSAudioChunk(event)
		if err != nil {
			return nil, false, err
		}
		return chunk, event.Sequence < 0, nil
	}

	return parseDoubaoWSBinaryResponse(payload)
}

func parseDoubaoWSBinaryResponse(frame []byte) ([]byte, bool, error) {
	if len(frame) < 4 {
		return nil, false, fmt.Errorf("parseDoubao WebSocket TTS responsethất bại: responseframeđộ dàinoi_dung")
	}

	headerSizeBytes := int(frame[0]&0x0f) * 4
	if headerSizeBytes <= 0 || len(frame) < headerSizeBytes {
		return nil, false, fmt.Errorf("parseDoubao WebSocket TTS responsethất bại: noi_dungđộ dài")
	}

	messageType := frame[1] >> 4
	flags := frame[1] & 0x0f
	compression := frame[2] & 0x0f
	payload := frame[headerSizeBytes:]

	switch messageType {
	case 0x09:
		log.Debugf("Doubao WebSocket TTS Nhận full server response: flags=%d compression=%d payload_len=%d", flags, compression, len(payload))
		return parseDoubaoWSFullServerResponse(payload, compression, flags)
	case 0x0b:
		log.Debugf("Doubao WebSocket TTS Nhận audio-only response: flags=%d payload_len=%d", flags, len(payload))
		if flags == 0 {
			return nil, false, nil
		}
		if flags == 4 {
			_, requestID, audio, err := extractDoubaoWSFlag4Payload(payload)
			if err != nil {
				return nil, false, err
			}
			log.Debugf("Doubao WebSocket audioframe(request_id=%s): audio_len=%d", requestID, len(audio))
			return audio, false, nil
		}
		if len(payload) < 8 {
			return nil, false, fmt.Errorf("parseDoubao WebSocket audioframethất bại: payload độ dàinoi_dung")
		}
		sequenceNumber := int32(binary.BigEndian.Uint32(payload[0:4]))
		payloadSize := int(binary.BigEndian.Uint32(payload[4:8]))
		audio := payload[8:]
		if payloadSize >= 0 && payloadSize < len(audio) {
			audio = audio[:payloadSize]
		}
		return audio, sequenceNumber < 0, nil
	case 0x0c:
		if len(payload) < 4 {
			return nil, false, fmt.Errorf("parseDoubao WebSocket trướcnoi_dungmessagethất bại: payload độ dàinoi_dung")
		}
		msgPayload := payload[4:]
		if compression == 1 {
			var err error
			msgPayload, err = gzipDecompressDoubao(msgPayload)
			if err != nil {
				return nil, false, fmt.Errorf("noi_dungDoubao WebSocket trướcnoi_dungmessagethất bại: %w", err)
			}
		}
		if !looksLikeJSONPayload(msgPayload) {
			return nil, false, nil
		}
		event, err := parseDoubaoTTSStreamEvent(msgPayload)
		if err != nil || event == nil {
			return nil, false, err
		}
		chunk, err := decodeDoubaoTTSAudioChunk(event)
		if err != nil {
			return nil, false, err
		}
		return chunk, event.Sequence < 0, nil
	case 0x0f:
		if len(payload) < 8 {
			return nil, false, fmt.Errorf("parseDoubao WebSocket lỗiframethất bại: payload độ dàinoi_dung")
		}
		code := int32(binary.BigEndian.Uint32(payload[0:4]))
		errPayload := payload[8:]
		if compression == 1 {
			var err error
			errPayload, err = gzipDecompressDoubao(errPayload)
			if err != nil {
				return nil, false, fmt.Errorf("noi_dungDoubao WebSocket lỗiframethất bại: %w", err)
			}
		}
		msg := strings.TrimSpace(string(errPayload))
		if msg == "" {
			msg = fmt.Sprintf("Doubao WebSocket TTS servicenoi_dunglỗi(code=%d)", code)
		}
		return nil, false, fmt.Errorf("%s", msg)
	default:
		preview := frame
		if len(preview) > 16 {
			preview = preview[:16]
		}
		return nil, false, fmt.Errorf("parseDoubao WebSocket TTS responsethất bại: không xác địnhmessageloại %d, noi_dung=%x", messageType, preview)
	}
}

func parseDoubaoWSFullServerResponse(payload []byte, compression byte, flags byte) ([]byte, bool, error) {
	if flags == 4 {
		marker, requestID, body, err := extractDoubaoWSFlag4Payload(payload)
		if err != nil {
			return nil, false, err
		}
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("{}")) {
			log.Debugf("Doubao WebSocket ACK(request_id=%s) noi_dung: marker=%d body=%q", requestID, marker, string(trimmed))
			return nil, true, nil
		}
		if !looksLikeJSONPayload(trimmed) {
			log.Debugf("Doubao WebSocket ACK(request_id=%s) noi_dungJSON payload: marker=%d body=%x", requestID, marker, trimmed)
			return nil, false, nil
		}

		var event doubaoTTSV3Event
		if err := json.Unmarshal(trimmed, &event); err == nil && (event.Code != 0 || event.Message != "" || event.Data != nil) {
			if event.Code != 0 {
				msg := strings.TrimSpace(event.Message)
				if msg == "" {
					msg = fmt.Sprintf("Doubao WebSocket TTS trả vềlỗinoi_dung %d", event.Code)
				}
				return nil, false, fmt.Errorf("%s", msg)
			}
			chunk, err := decodeDoubaoTTSAudioChunk(&event)
			if err != nil {
				return nil, false, err
			}
			return chunk, event.Sequence < 0, nil
		}

		log.Debugf("Doubao WebSocket ACK(request_id=%s) noi_dungdata: marker=%d body=%s", requestID, marker, string(trimmed))
		return nil, false, nil
	}

	if len(payload) < 8 {
		return nil, false, fmt.Errorf("parseDoubao WebSocket ACK framethất bại: payload độ dàinoi_dung")
	}

	sequenceNumber := int32(binary.BigEndian.Uint32(payload[0:4]))
	payloadSize := int(binary.BigEndian.Uint32(payload[4:8]))
	body := payload[8:]
	if payloadSize >= 0 && payloadSize < len(body) {
		body = body[:payloadSize]
	}
	if compression == 1 {
		var err error
		body, err = gzipDecompressDoubao(body)
		if err != nil {
			return nil, false, fmt.Errorf("noi_dungDoubao WebSocket ACK framethất bại: %w", err)
		}
	}
	if !looksLikeJSONPayload(body) {
		log.Debugf("Doubao WebSocket ACK noi_dungJSON payload: %x", body)
		return nil, sequenceNumber < 0, nil
	}

	var event doubaoTTSV3Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, false, fmt.Errorf("parseDoubao WebSocket ACK JSON thất bại: %w", err)
	}
	log.Debugf("Doubao WebSocket ACK JSON: code=%d sequence=%d message=%q", event.Code, event.Sequence, event.Message)
	if event.Code != 0 {
		msg := strings.TrimSpace(event.Message)
		if msg == "" {
			msg = fmt.Sprintf("Doubao WebSocket TTS trả vềlỗinoi_dung %d", event.Code)
		}
		return nil, false, fmt.Errorf("%s", msg)
	}
	chunk, err := decodeDoubaoTTSAudioChunk(&event)
	if err != nil {
		return nil, false, err
	}
	return chunk, event.Sequence < 0 || sequenceNumber < 0, nil
}

func extractDoubaoWSFlag4Payload(payload []byte) (int32, string, []byte, error) {
	if len(payload) < 12 {
		return 0, "", nil, fmt.Errorf("parseDoubao WebSocket flags=4 framethất bại: payload độ dàinoi_dung")
	}
	marker := int32(binary.BigEndian.Uint32(payload[0:4]))
	requestIDLen := int(binary.BigEndian.Uint32(payload[4:8]))
	if requestIDLen < 0 || len(payload) < 8+requestIDLen+4 {
		return 0, "", nil, fmt.Errorf("parseDoubao WebSocket flags=4 framethất bại: request_id độ dàinoi_dung")
	}
	requestID := string(payload[8 : 8+requestIDLen])
	bodyLenOffset := 8 + requestIDLen
	bodyLen := int(binary.BigEndian.Uint32(payload[bodyLenOffset : bodyLenOffset+4]))
	body := payload[bodyLenOffset+4:]
	if bodyLen >= 0 && bodyLen <= len(body) {
		body = body[:bodyLen]
	}
	return marker, requestID, body, nil
}

func buildDoubaoPublicFallbackModel(resolved resolvedTTSModel, voice string) *resolvedTTSModel {
	if isDoubaoCloneVoice(voice) {
		return nil
	}
	switch resolved.ResourceID {
	case resourceSeedTTS10:
		fallback := resolvedTTSModel{
			ConfigModel:  modelSeedTTS20Standard,
			RequestModel: modelSeedTTS20Standard,
			ResourceID:   resourceSeedTTS20,
			VoiceFamily:  resolved.VoiceFamily,
		}
		return &fallback
	case resourceSeedTTS20:
		fallback := resolvedTTSModel{
			ConfigModel:  modelSeedTTS11,
			RequestModel: modelSeedTTS11,
			ResourceID:   resourceSeedTTS10,
			VoiceFamily:  resolved.VoiceFamily,
		}
		return &fallback
	default:
		return nil
	}
}

func buildDoubaoWSAttemptModels(derived resolvedTTSModel, explicitResourceID, voice string) []resolvedTTSModel {
	models := make([]resolvedTTSModel, 0, 3)
	seen := map[string]struct{}{}
	push := func(candidate resolvedTTSModel) {
		key := candidate.ResourceID + "|" + candidate.RequestModel
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		models = append(models, candidate)
	}

	if explicit := strings.TrimSpace(explicitResourceID); explicit != "" {
		override := derived
		override.ResourceID = explicit
		push(override)
	}
	push(derived)
	if fallback := buildDoubaoPublicFallbackModel(derived, voice); fallback != nil {
		push(*fallback)
	}
	return models
}

func formatDoubaoWSConnectError(err error, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("noi_dungDoubao WebSocket TTS kết nốithất bại: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if readErr != nil && bodyText == "" {
		return fmt.Errorf("noi_dungDoubao WebSocket TTS kết nốithất bại: websocket handshake status=%d", resp.StatusCode)
	}
	if bodyText != "" {
		return fmt.Errorf("noi_dungDoubao WebSocket TTS kết nốithất bại: websocket handshake status=%d body=%s", resp.StatusCode, bodyText)
	}
	return fmt.Errorf("noi_dungDoubao WebSocket TTS kết nốithất bại: websocket handshake status=%d", resp.StatusCode)
}

func isDoubaoResourceMismatchError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "resource id is mismatched with speaker related resource")
}

func isDoubaoResourceNotGrantedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "requested resource not granted")
}

func isDoubaoRetryableResourceError(err error) bool {
	return isDoubaoResourceMismatchError(err) || isDoubaoResourceNotGrantedError(err)
}

func summarizeDoubaoWSAttemptError(voice string, preferred resolvedTTSModel, attemptedResources []string, attemptErrors []error) error {
	if len(attemptErrors) == 0 {
		return nil
	}

	lastErr := attemptErrors[len(attemptErrors)-1]
	hasMismatch := false
	hasNotGranted := false
	for _, attemptErr := range attemptErrors {
		hasMismatch = hasMismatch || isDoubaoResourceMismatchError(attemptErr)
		hasNotGranted = hasNotGranted || isDoubaoResourceNotGrantedError(attemptErr)
	}

	retryInfo := ""
	if len(attemptedResources) > 0 {
		retryInfo = fmt.Sprintf("，đãnoi_dung resource_id=%s", strings.Join(attemptedResources, ","))
	}

	if hasNotGranted && hasMismatch {
		return fmt.Errorf(
			"Doubao WebSocket TTS confignoi_dungkhả dụng: voice=%s, model=%s, resource_id=%s%s。hiện tạinoi_dung resource_id noi_dung app/token chưanoi_dung，noi_dungtớimodelnoi_dung resource saunoi_dungvoicenoi_dungtài nguyênnoi_dung。noi_dungđanghiện tại AccessKey thực tếnoi_dungvoice/ResourceID noi_dunglànoi_dung",
			voice,
			preferred.ConfigModel,
			preferred.ResourceID,
			retryInfo,
		)
	}
	if hasNotGranted {
		return fmt.Errorf(
			"Doubao WebSocket TTS resource_id chưanoi_dung: voice=%s, model=%s, resource_id=%s%s。hiện tại app/token noi_dung ResourceID，noi_dunghiện tại AccessKey thực tếnoi_dung ResourceID，hoặcnoi_dungrỗng resource_id noi_dungmodelmặc địnhnoi_dung",
			voice,
			preferred.ConfigModel,
			preferred.ResourceID,
			retryInfo,
		)
	}
	if !hasMismatch {
		return lastErr
	}

	hint := "noi_dungvoicenoi_dunghiện tại app/token noi_dung v3 ResourceID noi_dung。cũnoi_dung v1 voicelistnoi_dungtới v3。"
	switch strings.ToLower(strings.TrimSpace(voice)) {
	case "zh_female_wanwanxiaohe_moon_bigtts", "zh_female_qinqienvsheng_moon_bigtts":
		hint = "noi_dungvoicenoi_dungcũnoi_dungtĩnhlist，noi_dungtới v3 saunoi_dungmớinoi_dungthực tếnoi_dungvoice，cũdatanoi_dung。"
	}

	return fmt.Errorf(
		"Doubao WebSocket TTS voicenoi_dungtài nguyênnoi_dung: voice=%s, model=%s, resource_id=%s%s。%s noi_dunghoặcnoi_dungthực tếnoi_dungvoice/ResourceID noi_dunglànoi_dung",
		voice,
		preferred.ConfigModel,
		preferred.ResourceID,
		retryInfo,
		hint,
	)
}

func gzipCompressDoubao(input []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(input); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func gzipDecompressDoubao(input []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func looksLikeJSONPayload(payload []byte) bool {
	trimmed := bytes.TrimSpace(payload)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}
