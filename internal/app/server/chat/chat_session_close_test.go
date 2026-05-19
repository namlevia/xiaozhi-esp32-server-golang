package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	types_conn "xiaozhi-esp32-server-golang/internal/app/server/types"
	. "xiaozhi-esp32-server-golang/internal/data/client"
)

type sessionCloseTestConn struct {
	deviceID        string
	transportType   string
	sentCmds        [][]byte
	closeAudioCalls int
}

func (c *sessionCloseTestConn) SendCmd(msg []byte) error {
	copyMsg := append([]byte(nil), msg...)
	c.sentCmds = append(c.sentCmds, copyMsg)
	return nil
}

func (c *sessionCloseTestConn) RecvCmd(ctx context.Context, timeout int) ([]byte, error) {
	return nil, nil
}

func (c *sessionCloseTestConn) SendAudio(audio []byte) error {
	return nil
}

func (c *sessionCloseTestConn) RecvAudio(ctx context.Context, timeout int) ([]byte, error) {
	return nil, nil
}

func (c *sessionCloseTestConn) GetDeviceID() string {
	return c.deviceID
}

func (c *sessionCloseTestConn) Close() error {
	return nil
}

func (c *sessionCloseTestConn) OnClose(func(deviceId string)) {}

func (c *sessionCloseTestConn) CloseAudioChannel() error {
	c.closeAudioCalls++
	return nil
}

func (c *sessionCloseTestConn) GetTransportType() string {
	return c.transportType
}

func (c *sessionCloseTestConn) GetData(key string) (interface{}, error) {
	return nil, nil
}

func assertSingleGoodbyeCommand(t *testing.T, fakeConn *sessionCloseTestConn, sessionID string) {
	t.Helper()

	if len(fakeConn.sentCmds) != 1 {
		t.Fatalf("mong đợi một mqtt goodbye command, nhận được %d", len(fakeConn.sentCmds))
	}

	var msg map[string]any
	if err := json.Unmarshal(fakeConn.sentCmds[0], &msg); err != nil {
		t.Fatalf("unmarshal goodbye command thất bại: %v", err)
	}
	if got := msg["type"]; got != "goodbye" {
		t.Fatalf("mong đợi type goodbye, nhận được %v", got)
	}
	if got := msg["state"]; got != "stop" {
		t.Fatalf("mong đợi state stop của goodbye, nhận được %v", got)
	}
	if got := msg["session_id"]; got != sessionID {
		t.Fatalf("mong đợi session_id %s, nhận được %v", sessionID, got)
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnExplicitExit(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonExplicitExit)

	if manager.GetSession() != nil {
		t.Fatalf("mong đợi session được xóa sau explicit exit")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi bỏ qua CloseAudioChannel khi explicit exit, nhận được %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("mong đợi helloInited vẫn true sau mqtt explicit exit")
	}
	if !manager.needFreshHello {
		t.Fatalf("mong đợi mqtt explicit exit yêu cầu hello mới trước lần bootstrap session tiếp theo")
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnFatalError(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonFatalError)

	if manager.GetSession() != nil {
		t.Fatalf("mong đợi session được xóa sau fatal_error")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi bỏ qua CloseAudioChannel khi fatal_error, nhận được %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("mong đợi helloInited vẫn true sau fatal_error")
	}
	if !manager.needFreshHello {
		t.Fatalf("mong đợi fatal_error yêu cầu hello mới trước lần bootstrap session tiếp theo")
	}
}

func TestHandleSessionClosedSkipsMqttGoodbyeOnRetainedIdleTimeout(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonRetainedIdleTimeout)

	if manager.GetSession() != nil {
		t.Fatalf("mong đợi session được xóa sau retained_idle_timeout")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi bỏ qua CloseAudioChannel khi retained_idle_timeout, nhận được %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("mong đợi retained_idle_timeout không gửi mqtt goodbye, nhận được %d command", len(fakeConn.sentCmds))
	}
	if !manager.helloInited {
		t.Fatalf("mong đợi helloInited vẫn true sau retained_idle_timeout")
	}
	if !manager.needFreshHello {
		t.Fatalf("mong đợi retained_idle_timeout yêu cầu hello mới trước lần bootstrap session tiếp theo")
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnAudioIdleTimeout(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonAudioIdleTimeout)

	if manager.GetSession() != nil {
		t.Fatalf("mong đợi session được xóa sau audio_idle_timeout")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi bỏ qua CloseAudioChannel khi audio_idle_timeout, nhận được %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("mong đợi helloInited vẫn true sau audio_idle_timeout")
	}
	if !manager.needFreshHello {
		t.Fatalf("mong đợi audio_idle_timeout yêu cầu hello mới trước lần bootstrap session tiếp theo")
	}
}

func TestHandleSessionClosedDoesNotRequireHelloAfterMqttExplicitExit(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonExplicitExit)

	if !manager.needFreshHello {
		t.Fatalf("mong đợi explicit exit yêu cầu hello mới")
	}
}

func TestHandleSessionClosedIgnoresStaleSessionCallback(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	staleSession := &ChatSession{}
	currentSession := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         currentSession,
		helloInited:     true,
	}

	manager.handleSessionClosed(staleSession, chatSessionCloseReasonExplicitExit)

	if manager.GetSession() != currentSession {
		t.Fatalf("mong đợi session hiện tại vẫn active sau stale callback")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi stale callback không đóng audio, nhận được %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("mong đợi stale callback không gửi mqtt goodbye, nhận được %d command", len(fakeConn.sentCmds))
	}
}

func TestEnsureSessionRejectsClosingSession(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	closingSession := &ChatSession{}
	closingSession.closing.Store(true)
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         closingSession,
		helloInited:     true,
	}

	session, err := manager.ensureSession()
	if err == nil {
		t.Fatalf("mong đợi ensureSession từ chối session đang đóng")
	}
	if session != nil {
		t.Fatalf("mong đợi không thay session khi session hiện tại đang đóng")
	}
	if manager.GetSession() != closingSession {
		t.Fatalf("mong đợi session đang đóng vẫn được đăng ký tới khi close callback hoàn tất")
	}
	if !strings.Contains(err.Error(), "đóng") {
		t.Fatalf("mong đợi lỗi ensureSession nhắc tới trạng thái đóng, nhận được %v", err)
	}
}

func TestEnsureSessionWaitsForStartingSession(t *testing.T) {
	manager := &ChatManager{
		helloInited:         true,
		startingSession:     &ChatSession{},
		startingSessionDone: make(chan struct{}),
	}
	expectedSession := &ChatSession{}

	type result struct {
		session *ChatSession
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		session, err := manager.ensureSession()
		resultCh <- result{session: session, err: err}
	}()

	select {
	case <-resultCh:
		t.Fatalf("mong đợi ensureSession chờ khi session đang startup")
	case <-time.After(50 * time.Millisecond):
	}

	manager.sessionMu.Lock()
	waitCh := manager.startingSessionDone
	manager.startingSession = nil
	manager.startingSessionDone = nil
	manager.session = expectedSession
	manager.sessionMu.Unlock()
	close(waitCh)

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("mong đợi ensureSession đang chờ thành công, nhận được %v", result.err)
		}
		if result.session != expectedSession {
			t.Fatalf("mong đợi ensureSession đang chờ tái sử dụng published session")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout khi chờ ensureSession tiếp tục")
	}
}

func TestHandleSessionClosedClearsStartingSessionWithoutTransportCleanup(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	startingSession := &ChatSession{}
	waitCh := make(chan struct{})
	manager := &ChatManager{
		transport:           fakeConn,
		serverTransport:     NewServerTransport(fakeConn, clientState),
		startingSession:     startingSession,
		startingSessionDone: waitCh,
		helloInited:         true,
	}

	manager.handleSessionClosed(startingSession, chatSessionCloseReasonFatalError)

	select {
	case <-waitCh:
	default:
		t.Fatalf("mong đợi wait channel của startingSession được đóng")
	}
	if manager.startingSession != nil {
		t.Fatalf("mong đợi startingSession được xóa sau close callback")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("mong đợi startup close callback không đóng audio, nhận được %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("mong đợi startup close callback không gửi mqtt goodbye, nhận được %d command", len(fakeConn.sentCmds))
	}
}

func TestShutdownClosesStartingSession(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{
		DeviceID: "device-1",
		Ctx:      context.Background(),
	}
	serverTransport := NewServerTransport(fakeConn, clientState)
	waitCh := make(chan struct{})
	manager := &ChatManager{
		transport:           fakeConn,
		serverTransport:     serverTransport,
		startingSessionDone: waitCh,
	}
	startingSession := NewChatSession(clientState, serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.startingSession = startingSession

	if err := manager.shutdown(false); err != nil {
		t.Fatalf("mong đợi shutdown không kèm transport thành công, nhận được %v", err)
	}
	if manager.startingSession != nil {
		t.Fatalf("mong đợi shutdown xóa startingSession")
	}
	if !startingSession.IsClosing() {
		t.Fatalf("mong đợi shutdown đóng startingSession")
	}

	select {
	case <-waitCh:
	default:
		t.Fatalf("mong đợi shutdown giải phóng waiter của startingSession")
	}
}
