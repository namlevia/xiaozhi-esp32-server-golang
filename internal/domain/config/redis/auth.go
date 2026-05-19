package redis_config

import (
	"context"
	"fmt"
	"math/rand"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"

	"github.com/google/uuid"
)

type activationInfo struct {
	code      string
	challenge string
	msg       string
}

var verfiyDeviceId = map[string]bool{}
var preActivationInfo = map[string]activationInfo{}

// Thiết bị đã kích hoạt hay chưa?
func (r *UserConfig) IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error) {
	if _, ok := verfiyDeviceId[deviceId]; ok {
		return true, nil
	}
	return false, nil
}

// Lấy thông tin cần cho kích hoạt: code, challenge, msg, timeoutMs.
func (r *UserConfig) GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int) {
	if info, ok := preActivationInfo[deviceId]; ok {
		return info.code, info.challenge, info.msg, 300
	}
	challenge := uuid.New().String()
	code := fmt.Sprintf("%06d", rand.Intn(1000000)) // 000000~999999, giữ số 0 ở đầu
	preActivationInfo[deviceId] = activationInfo{
		code:      code,
		challenge: challenge,
		msg:       fmt.Sprintf("xiaozhi\n%s", code),
	}
	return code, challenge, preActivationInfo[deviceId].msg, 300
}

// Xác minh challenge và HMAC có khớp không, thiết bị đã kích hoạt chưa; ở đây có thể bỏ qua kiểm tra HMAC và chỉ truy vấn deviceId đã bind hay chưa.
func (r *UserConfig) VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	if _, ok := verfiyDeviceId[deviceId]; ok {
		return true, nil
	}
	if info, ok := preActivationInfo[deviceId]; ok {
		if info.challenge == activationPayload.Challenge {
			verfiyDeviceId[deviceId] = true
			delete(preActivationInfo, deviceId)
			return true, nil
		}
	}
	return false, nil
}
