package mqtt_server

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	mqttServer "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/spf13/viper"
)

// AuthHook triển khai logic xác thực tùy chỉnh.
// Hỗ trợ người dùng thường và super admin.
// Người dùng thường: username là base64 của {"ip":"1.202.193.194"}, password là chữ ký HMAC-SHA256.
// Super admin: username admin, password shijingbo!@#.
type AuthHook struct {
	mqttServer.HookBase
}

func (h *AuthHook) ID() string {
	return "custom-auth-hook"
}

func (h *AuthHook) Provides(b byte) bool {
	return b == mqttServer.OnConnectAuthenticate
}

func (h *AuthHook) OnConnectAuthenticate(cl *mqttServer.Client, pk packets.Packet) bool {
	// Kiểm tra có bật xác thực hay không.
	enableAuth := viper.GetBool("mqtt_server.enable_auth")
	if !enableAuth {
		//log.Infof("Xác thực MQTT đã tắt, cho phép mọi kết nối")
		return true
	}

	username := string(pk.Connect.Username)
	password := string(pk.Connect.Password)
	clientId := string(pk.Connect.ClientIdentifier)

	// Kiểm tra super admin.
	adminUsername := configuredAdminUsername()
	adminPassword := configuredAdminPassword()
	if username == adminUsername && password == adminPassword {
		log.Infof("Super admin đăng nhập thành công: %s", username)
		return true
	}
	if username == adminUsername {
		log.Warnf("Đăng nhập MQTT admin thất bại: username=%s, clientId=%s, lý do=sai mật khẩu", username, clientId)
		return false
	}

	// Kiểm tra người dùng thường bằng logic xác minh chữ ký mới.
	signatureKey := viper.GetString("mqtt_server.signature_key")
	if signatureKey != "" {
		credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
		//log.Infof("Bắt đầu xác minh người dùng MQTT: clientId=%s, username=%s, password=%s, signatureKey=%s",
		//	clientId, username, password, signatureKey)
		//log.Infof("Bắt đầu xác minh người dùng MQTT: credentialInfo=%+v", credentialInfo)

		if err != nil {
			log.Warnf("Xác minh thông tin MQTT thất bại: username=%s, clientId=%s, err=%v", username, clientId, err)
			return false
		}

		log.Infof("Xác minh người dùng MQTT thành công: groupId=%s, macAddress=%s, uuid=%s",
			credentialInfo.GroupId, credentialInfo.MacAddress, credentialInfo.UUID)
		return true
	}

	// Nếu chưa cấu hình signature key, fallback về logic xác minh AES cũ.
	log.Warnf("Thiếu cấu hình OTA signature key, dùng phương thức xác minh AES")
	return h.validateWithAes(username, password)
}

// validateWithAes xác minh password bằng AES để tương thích ngược.
func (h *AuthHook) validateWithAes(username, password string) bool {
	// Kiểm tra người dùng thường.
	decoded, err := base64.StdEncoding.DecodeString(username)
	if err != nil {
		return false
	}
	var userInfo map[string]string
	if err := json.Unmarshal(decoded, &userInfo); err != nil {
		return false
	}
	if _, ok := userInfo["ip"]; !ok {
		return false
	}
	// Kiểm tra password có phải username đã được AES encrypt rồi base64 hay không.
	if !checkAesPassword(username, password) {
		return false
	}
	return true
}

// checkAesPassword kiểm tra password có bằng base64(AES-ECB(username)) hay không.
func checkAesPassword(username, password string) bool {
	key := []byte("xiaozhi_aes_key_1") // Khóa 16 byte; thực tế nên đưa vào cấu hình.
	ciphertext, err := aesEncryptECB([]byte(username), key)
	if err != nil {
		return false
	}
	cipherBase64 := base64.StdEncoding.EncodeToString(ciphertext)
	return cipherBase64 == password
}

// aesEncryptECB triển khai mã hóa AES-ECB.
func aesEncryptECB(src, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	// PKCS7 padding.
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	src = append(src, padtext...)
	encrypted := make([]byte, len(src))
	for bs, be := 0, blockSize; bs < len(src); bs, be = bs+blockSize, be+blockSize {
		block.Encrypt(encrypted[bs:be], src[bs:be])
	}
	return encrypted, nil
}
