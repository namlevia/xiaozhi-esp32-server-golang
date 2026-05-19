package manager

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"xiaozhi-esp32-server-golang/internal/components/http"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// Struct response interface HTTP.

// CheckActivationResponse là response kiểm tra trạng thái kích hoạt.
type CheckActivationResponse struct {
	Activated bool   `json:"activated"`
	Message   string `json:"message"`
}

// GetActivationInfoResponse là response lấy thông tin kích hoạt.
type GetActivationInfoResponse struct {
	Activated bool   `json:"activated"`
	Code      string `json:"code,omitempty"` // Đổi sang kiểu string để khớp API backend
	Challenge string `json:"challenge,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ActivateDeviceRequest là request kích hoạt thiết bị.
type ActivateDeviceRequest struct {
	DeviceId     string `json:"device_id"`
	ClientId     string `json:"client_id"`
	Code         string `json:"code"`
	Challenge    string `json:"challenge"`
	Algorithm    string `json:"algorithm"`
	SerialNumber string `json:"serial_number"`
	Hmac         string `json:"hmac"`
}

// ActivateDeviceResponse là response kích hoạt thiết bị.
type ActivateDeviceResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// IsDeviceActivated kiểm tra thiết bị đã kích hoạt hay chưa.
func (am *ConfigManager) IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error) {
	// Gọi trực tiếp HTTP API của backend manager
	activated, err := am.callCheckActivationAPI(ctx, deviceId, clientId)
	if err != nil {
		log.Log().Errorf("Kiểm tra trạng thái kích hoạt thiết bị %s thất bại: %v", deviceId, err)
		return false, err
	}

	log.Log().Debugf("Trạng thái kích hoạt thiết bị %s: %v", deviceId, activated)
	return activated, nil
}

// GetActivationInfo lấy thông tin kích hoạt thiết bị.
func (am *ConfigManager) GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int) {
	// Gọi trực tiếp HTTP API của backend manager
	activated, codeStr, challenge, message, err := am.callGetActivationInfoAPI(ctx, deviceId, clientId)
	if err != nil {
		log.Log().Errorf("Lấy thông tin kích hoạt thiết bị %s thất bại: %v", deviceId, err)
		return "", "", "", 0
	}

	// Nếu thiết bị đã kích hoạt thì trả về trực tiếp
	if activated {
		log.Log().Debugf("Thiết bị %s đã kích hoạt", deviceId)
		return "", "", message, 0
	}

	// Kiểm tra Challenge có rỗng hay không
	if challenge == "" {
		log.Log().Errorf("Field Challenge của thiết bị %s rỗng", deviceId)
		return "", "", "Field Challenge rỗng, vui lòng liên hệ quản trị viên", 0
	}

	// Thiết bị chưa kích hoạt, trả về thông tin kích hoạt
	timeoutMs := 300 // Mặc định timeout 5 phút
	log.Log().Debugf("Lấy thông tin kích hoạt thiết bị %s: code=%s, challenge=%s", deviceId, codeStr, challenge)
	if codeStr == "" {
		log.Log().Warnf("Mã kích hoạt của thiết bị %s rỗng", deviceId)
	}

	return codeStr, challenge, message, timeoutMs
}

// VerifyChallenge xác minh challenge code và HMAC.
func (am *ConfigManager) VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	// Xác minh HMAC nếu có cung cấp HMAC
	if activationPayload.HMAC != "" {
		if !am.verifyHMAC(activationPayload.Challenge, activationPayload.HMAC) {
			log.Log().Warnf("Xác minh HMAC thiết bị %s thất bại", deviceId)
			return false, fmt.Errorf("Xác minh HMAC thất bại")
		}
	}

	// Gọi trực tiếp API kích hoạt của backend manager
	verified, err := am.callActivateDeviceAPI(ctx, deviceId, clientId, activationPayload)
	if err != nil {
		log.Log().Errorf("Kích hoạt thiết bị thất bại: %v", err)
		return false, err
	}

	if verified {
		log.Log().Infof("Xác minh kích hoạt thiết bị %s thành công", deviceId)
	}

	return verified, nil
}

// verifyHMAC xác minh chữ ký HMAC.
func (am *ConfigManager) verifyHMAC(challenge, providedHmac string) bool {
	// Có thể cấu hình secret key theo nhu cầu thực tế tại đây
	// Tạm thời dùng secret key rỗng; thực tế nên lấy từ config
	secretKey := ""

	if secretKey == "" {
		// Nếu chưa cấu hình secret key thì cho qua xác minh trực tiếp
		return true
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(challenge))
	expectedHmac := hex.EncodeToString(mac.Sum(nil))

	return expectedHmac == providedHmac
}

// Method gọi HTTP API.

// callCheckActivationAPI gọi API kiểm tra trạng thái kích hoạt.
func (am *ConfigManager) callCheckActivationAPI(ctx context.Context, deviceId, clientId string) (bool, error) {
	var response CheckActivationResponse

	// Gửi HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/device/check-activation",
		QueryParams: map[string]string{
			"device_id": deviceId,
			"client_id": clientId,
		},
		Response: &response,
	})
	if err != nil {
		return false, fmt.Errorf("Request thất bại: %w", err)
	}

	log.Log().Debugf("Response kiểm tra trạng thái kích hoạt: %+v", response)
	return response.Activated, nil
}

// callGetActivationInfoAPI gọi API lấy thông tin kích hoạt.
func (am *ConfigManager) callGetActivationInfoAPI(ctx context.Context, deviceId, clientId string) (bool, string, string, string, error) {
	var response GetActivationInfoResponse

	// Gửi HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/device/activation-info",
		QueryParams: map[string]string{
			"device_id": deviceId,
			"client_id": clientId,
		},
		Response: &response,
	})
	if err != nil {
		return false, "", "", "", fmt.Errorf("Request thất bại: %w", err)
	}

	log.Log().Debugf("Response lấy thông tin kích hoạt: %+v", response)

	if response.Activated {
		return true, "", "", response.Message, nil
	}

	return false, response.Code, response.Challenge, response.Message, nil
}

// callActivateDeviceAPI gọi API kích hoạt thiết bị.
func (am *ConfigManager) callActivateDeviceAPI(ctx context.Context, deviceId, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	// Tạo request body
	request := ActivateDeviceRequest{
		DeviceId:     deviceId,
		ClientId:     clientId,
		Challenge:    activationPayload.Challenge,
		Algorithm:    activationPayload.Algorithm,
		SerialNumber: activationPayload.SerialNumber,
		Hmac:         activationPayload.HMAC,
	}

	var response ActivateDeviceResponse

	// Gửi HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method:   "POST",
		Path:     "/api/internal/device/activate",
		Body:     request,
		Response: &response,
	})
	if err != nil {
		return false, fmt.Errorf("Request thất bại: %w", err)
	}

	log.Log().Debugf("Response kích hoạt thiết bị: %+v", response)

	if !response.Success {
		return false, nil
	}

	return response.Success, nil
}
