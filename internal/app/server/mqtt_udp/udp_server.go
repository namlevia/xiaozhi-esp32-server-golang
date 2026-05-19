package mqtt_udp

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	. "xiaozhi-esp32-server-golang/logger"
)

// UDPServer là cấu trúc UDP server
/*
type UDPServer struct {
	conn       *net.UDPConn
	sessions   map[string]*Session
	mqttServer *MqttServer
	udpPort    int
	sync.RWMutex
}*/

type UdpServer struct {
	conn           *net.UDPConn
	udpPort        int      //udp server listen port
	externalHost   string   //udp server external host
	externalPort   int      //udp server external port
	connId2Session sync.Map //connId => UdpSession
	mqttAdapter    *MqttUdpAdapter
	sync.RWMutex
}

const maxConnIDGenerateAttempts = 16

var udpRandReader io.Reader = rand.Reader

// NewUDPServer tạo UDP server mới
func NewUDPServer(udpPort int, externalHost string, externalPort int) *UdpServer {
	return &UdpServer{
		udpPort:        udpPort,
		externalHost:   externalHost,
		externalPort:   externalPort,
		connId2Session: sync.Map{},
	}
}

// Start khởi động UDP server
func (s *UdpServer) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: s.udpPort,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("Listen UDP thất bại: %v", err)
	}

	s.conn = conn
	Infof("UDP server đã khởi động tại %s:%d", "0.0.0.0", s.udpPort)

	// Khởi động dọn session
	//go s.cleanupSessions()

	// Khởi động xử lý packet
	go s.handlePackets()

	return nil
}

// Close đóng UDP server để handlePackets thoát
func (s *UdpServer) Close() error {
	s.Lock()
	conn := s.conn
	s.conn = nil
	s.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// handlePackets xử lý packet nhận được
func (s *UdpServer) handlePackets() {
	buffer := make([]byte, 4096) // dùng kích thước buffer mặc định
	for {
		s.RLock()
		conn := s.conn
		s.RUnlock()
		if conn == nil {
			return
		}
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			s.RLock()
			closed := s.conn == nil
			s.RUnlock()
			if closed {
				return
			}
			Errorf("Đọc dữ liệu UDP thất bại: %v", err)
			continue
		}

		// Copy dữ liệu để tránh sửa đồng thời
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Xử lý packet
		s.processPacket(addr, data)
	}
}

func (s *UdpServer) getSessionByConnID(connID string) *UdpSession {
	val, ok := s.connId2Session.Load(connID)
	if ok {
		return val.(*UdpSession)
	}
	return nil
}

// processPacket xử lý từng packet
func (s *UdpServer) processPacket(addr *net.UDPAddr, data []byte) {
	// Kiểm tra kích thước packet
	if len(data) < 16 {
		Warn("Packet quá nhỏ")
		return
	}

	fullNonce := data[:16]
	connID := fullNonce[4:8] // lấy byte 5-8 làm connection ID
	strConnID := hex.EncodeToString(connID)
	udpSession := s.getSessionByConnID(strConnID)
	if udpSession == nil {
		//Warnf("session không tồn tại addr: %s, connID: %s", addr, strConnID)
		return
	}

	// Cập nhật thời gian hoạt động gần nhất
	udpSession.LastActive = time.Now()

	decrypted, err := udpSession.Decrypt(data)
	if err != nil {
		Errorf("addr: %s giải mã thất bại: %v", addr, err)
		return
	}
	currentAddr := udpSession.GetRemoteAddr()
	if currentAddr == nil || currentAddr.String() != addr.String() {
		udpSession.SetRemoteAddr(addr)
	}
	Debugf("Nhận dữ liệu audio, addr: %s, kích thước: %d byte", addr, len(decrypted))
	ok, err := udpSession.RecvData(decrypted)
	if err != nil {
		Errorf("addr: %s nhận dữ liệu thất bại: %v", addr, err)
		return
	}
	if !ok {
		Warnf("addr: %s nhận dữ liệu thất bại, channel đã đầy", addr)
		return
	}
	/*select {
	case udpSession.RecvChannel <- decrypted:
		return
	default:
		Warnf("udpSession.RecvChannel is full, addr: %s", addr)
	}*/
}

// cleanupSessions dọn session hết hạn
func (s *UdpServer) cleanupSessions() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now()
		s.connId2Session.Range(func(key, value interface{}) bool {
			session := value.(*UdpSession)
			if now.Sub(session.LastActive) > 5*time.Minute {
				s.connId2Session.Delete(key)
				Infof("Dọn session hết hạn: %s", key)
			}
			return true
		})
	}
}

// CreateSession tạo session mới
func (s *UdpServer) CreateSession(deviceId, clientId string) *UdpSession {
	// Tạo session ID
	sessionID, err := generateSessionID()
	if err != nil {
		Errorf("Tạo session ID thất bại: %v", err)
		return nil
	}

	// Tạo AES key
	key := make([]byte, 16)
	if err := fillRandomBytes(key); err != nil {
		Errorf("Tạo AES key thất bại: %v", err)
		return nil
	}

	// Tạo AES block
	block, err := aes.NewCipher(key)
	if err != nil {
		Errorf("Tạo AES block thất bại: %v", err)
		return nil
	}

	// Chuyển key thành [16]byte
	aesKey := [16]byte{}
	copy(aesKey[:], key)

	for attempt := 0; attempt < maxConnIDGenerateAttempts; attempt++ {
		// Tạo connection ID 4 byte
		connID := make([]byte, 4)
		if err := fillRandomBytes(connID); err != nil {
			Errorf("Tạo connection ID thất bại: %v", err)
			return nil
		}
		strConnID := hex.EncodeToString(connID)

		// Timestamp 4 byte
		timestamp := make([]byte, 4)
		binary.BigEndian.PutUint32(timestamp, uint32(time.Now().Unix()))

		// Ghép nonce: connection ID 4 byte + timestamp 4 byte
		nonce := append(connID, timestamp...)

		// Chuyển nonce thành [8]byte
		nonceBytes := [8]byte{}
		copy(nonceBytes[:], nonce)

		// Tạo session
		session := &UdpSession{
			ID:          sessionID,
			ConnId:      strConnID,
			ClientId:    clientId,
			DeviceId:    deviceId,
			AesKey:      aesKey,
			Nonce:       nonceBytes, // lưu nonce template gốc
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
			Block:       block,
			RecvChannel: make(chan []byte, 100),
			SendChannel: make(chan []byte, 100),
			Status:      UdpSessionStatusActive,
			Lock:        sync.Mutex{},
		}

		if _, loaded := s.connId2Session.LoadOrStore(strConnID, session); loaded {
			Warnf("UDP connID bị trùng, tạo lại: device=%s, connID=%s, attempt=%d", deviceId, strConnID, attempt+1)
			continue
		}

		s.startSessionSender(session)
		return session
	}

	Errorf("Tạo UDP connID duy nhất thất bại: device=%s", deviceId)
	return nil
}

func (s *UdpServer) startSessionSender(session *UdpSession) {
	go func() {
		for data := range session.SendChannel {
			remoteAddr := session.WaitRemoteAddr(2 * time.Second)
			if remoteAddr == nil {
				dropped := 1 + session.DrainPendingAudio()
				Warnf("Chưa có địa chỉ UDP remote, audio TTS bị bỏ: device=%s, connId=%s, dropped=%d", session.DeviceId, session.ConnId, dropped)
				continue
			}
			encrypted, err := session.Encrypt(data)
			if err != nil {
				Errorf("Mã hóa thất bại: %v", err)
				continue
			}
			_, err = s.writeToUDP(encrypted, remoteAddr)
			if err != nil {
				Errorf("Gửi dữ liệu audio thất bại: %v", err)
				continue
			}
		}
	}()
}

func (s *UdpServer) writeToUDP(data []byte, remoteAddr *net.UDPAddr) (int, error) {
	s.RLock()
	conn := s.conn
	s.RUnlock()
	if conn == nil {
		return 0, fmt.Errorf("udp server is closed")
	}
	return conn.WriteToUDP(data, remoteAddr)
}

// CloseSession đóng session
func (s *UdpServer) CloseSession(connID string) {
	session := s.getSessionByConnID(connID)
	s.CloseSessionByRef(session)
}

// ClearSessionAddrBinding dọn binding địa chỉ UDP của session theo connID, không hủy session
func (s *UdpServer) ClearSessionAddrBinding(connID string) {
	session := s.getSessionByConnID(connID)
	if session == nil {
		return
	}
	session.SetRemoteAddr(nil)
}

func (s *UdpServer) SetConnId2Session(connID string, session *UdpSession) {
	Debugf("SetConnId2Session, connID: %s, session: %+v", connID, session)
	s.connId2Session.Store(connID, session)
}

// GetSessionByConnID lấy thông tin session
func (s *UdpServer) GetSessionByConnID(connID string) *UdpSession {
	val, ok := s.connId2Session.Load(connID)
	if ok {
		return val.(*UdpSession)
	}
	return nil
}

// generateSessionID tạo session ID
func generateSessionID() (string, error) {
	b := make([]byte, 8)
	if err := fillRandomBytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func fillRandomBytes(buffer []byte) error {
	_, err := io.ReadFull(udpRandReader, buffer)
	return err
}

func (s *UdpServer) CloseSessionByRef(session *UdpSession) {
	if session == nil {
		return
	}
	s.connId2Session.Delete(session.ConnId)
	session.SetRemoteAddr(nil)
	session.Destroy()
}
