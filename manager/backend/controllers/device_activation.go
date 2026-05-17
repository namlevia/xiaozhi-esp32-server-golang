package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceActivationController struct {
	DB *gorm.DB
}

// 生成6位随机数字代码
func generateCode() string {
	randomBytes := make([]byte, 3)
	rand.Read(randomBytes)
	code := 0
	for i, b := range randomBytes {
		code += int(b) << (8 * i)
	}
	return fmt.Sprintf("%06d", code%1000000)
}

// 生成UUID格式的挑战码
func generateChallenge() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)

	// 设置版本 (4) 和变体位
	randomBytes[6] = (randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] = (randomBytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		randomBytes[0:4],
		randomBytes[4:6],
		randomBytes[6:8],
		randomBytes[8:10],
		randomBytes[10:16])
}

// 1. 判断设备是否已激活
// GET /api/internal/device/check-activation?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) CheckDeviceActivation(c *gin.Context) {
	deviceId := c.Query("device_id")
	//clientId := c.Query("client_id")

	if deviceId == "" /*|| clientId == ""*/ {
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "Tham số device_id là bắt buộc",
		})
		return
	}

	var device models.Device
	// 使用device_id (对应device_name字段) 查找设备
	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"activated": false,
				"message":   "Thiết bị không tồn tại",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "Truy vấn thiết bị thất bại",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": device.Activated,
		"message": func() string {
			if device.Activated {
				return "Thiết bị đã được kích hoạt"
			}
			return "设备未激活"
		}(),
	})
}

// 2. 获取激活信息
// GET /api/internal/device/activation-info?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) GetActivationInfo(c *gin.Context) {
	deviceId := c.Query("device_id")
	//clientId := c.Query("client_id")

	if deviceId == "" /*|| clientId == ""*/ {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tham số device_id và client_id là bắt buộc"})
		return
	}

	var device models.Device
	var isNewDevice bool

	// 使用device_id (对应device_name字段) 查找设备
	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Thiết bị không tồn tại，创建新设备记录
			device = models.Device{
				DeviceName: deviceId,
				UserID:     0, // user_id置为0
				DeviceCode: generateCode(),
				Challenge:  generateChallenge(),
				Activated:  false,
			}

			if err := dac.DB.Create(&device).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Tạo bản ghi thiết bị thất bại"})
				return
			}
			isNewDevice = true
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Truy vấn thiết bị thất bại"})
			return
		}
	}

	// 如果Thiết bị đã được kích hoạt，直接返回状态
	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"activated": true,
			"message":   "Thiết bị đã được kích hoạt",
		})
		return
	}

	// 如果设备未激活，生成或返回激活信息
	needUpdate := false

	// 如果没有激活码，生成新的激活码
	if device.DeviceCode == "" {
		device.DeviceCode = generateCode()
		needUpdate = true
	}

	// 如果没有挑战码，生成新的挑战码
	if device.Challenge == "" {
		device.Challenge = generateChallenge()
		needUpdate = true
	}

	// 确保user_id为0（如果不是新设备且未激活）
	if !isNewDevice && device.UserID != 0 {
		device.UserID = 0
		needUpdate = true
	}

	// 更新数据库
	if needUpdate {
		updates := map[string]interface{}{}
		if device.DeviceCode != "" {
			updates["device_code"] = device.DeviceCode
		}
		if device.Challenge != "" {
			updates["challenge"] = device.Challenge
		}
		if !isNewDevice && device.UserID == 0 {
			updates["user_id"] = device.UserID
		}
		if err := updateDeviceColumns(dac.DB, device.ID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cập nhật thông tin thiết bị thất bại"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": false,
		"code":      device.DeviceCode,
		"challenge": device.Challenge,
		"message":   "Vui lòng liên kết thiết bị kích hoạt trong trang quản trị, mã kích hoạt: " + device.DeviceCode,
	})
}

// 验证HMAC-SHA256
func verifyHMAC(challenge, secretKey, providedHmac string) bool {
	if secretKey == "" {
		return true // 如果pre_secret_key为空，直接通过验证
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(challenge))
	expectedHmac := hex.EncodeToString(mac.Sum(nil))

	return expectedHmac == providedHmac
}

// 3. 设备激活接口
// POST /api/internal/device/activate
func (dac *DeviceActivationController) ActivateDevice(c *gin.Context) {
	var req struct {
		DeviceId     string `json:"device_id" binding:"required"`
		ClientId     string `json:"client_id" binding:"required"`
		Challenge    string `json:"challenge" binding:"required"`
		Algorithm    string `json:"algorithm" binding:"required"`
		SerialNumber string `json:"serial_number" binding:"required"`
		Hmac         string `json:"hmac" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi tham số: " + err.Error()})
		return
	}

	var device models.Device
	// 使用device_id (对应device_name字段) 查找设备
	if err := dac.DB.Where("device_name = ?", req.DeviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "Thiết bị không tồn tại",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Truy vấn thiết bị thất bại",
		})
		return
	}

	// 检查设备是否已经激活
	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Thiết bị đã được kích hoạt",
		})
		return
	}

	if device.UserID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Thiết bị chưa được liên kết với người dùng",
		})
		return
	}

	if device.Challenge != req.Challenge {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Mã thách thức không chính xác",
		})
		return
	}

	// 验证HMAC（如果pre_secret_key为空则直接通过）
	if !verifyHMAC(req.Challenge, device.PreSecretKey, req.Hmac) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Xác thực HMAC thất bại",
		})
		return
	}

	// 激活设备
	device.Activated = true
	if err := updateDeviceColumns(dac.DB, device.ID, map[string]interface{}{
		"activated": device.Activated,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Kích hoạt thiết bị thất bại",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kích hoạt thiết bị thành công",
		"data": gin.H{
			"device_id": device.DeviceName,
			"activated": device.Activated,
		},
	})
}
