package controllers

import (
	"log"
	"net/http"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SetupController struct {
	DB *gorm.DB
}

type SetupRequest struct {
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=50"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=100"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
}

// 检查数据库是否需要初始化
func (sc *SetupController) CheckSetupStatus(c *gin.Context) {
	if sc.DB == nil {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Kết nối cơ sở dữ liệu không khả dụng",
		})
		return
	}

	// 检查是否存在用户表
	if !sc.DB.Migrator().HasTable(&models.User{}) {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Cấu trúc bảng cơ sở dữ liệu chưa được khởi tạo",
		})
		return
	}

	// 检查是否存在管理员用户
	var count int64
	sc.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "Cần tạo tài khoản quản trị viên",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"needs_setup": false,
		"message":     "Hệ thống đã được khởi tạo",
	})
}

// 初始化数据库
func (sc *SetupController) InitializeDatabase(c *gin.Context) {
	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if sc.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kết nối cơ sở dữ liệu không khả dụng"})
		return
	}

	// 开始事务
	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Khởi tạo transaction cơ sở dữ liệu thất bại"})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 自动迁移表结构
	log.Println("开始自动迁移数据库表结构...")
	err := tx.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.Agent{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.ChatMessage{},
	)
	if err != nil {
		tx.Rollback()
		log.Printf("Migration cấu trúc bảng cơ sở dữ liệu thất bại: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration cấu trúc bảng cơ sở dữ liệu thất bại: " + err.Error()})
		return
	}
	log.Println("数据库表结构迁移成功")

	// 2. 检查是否已存在管理员用户
	var existingAdmin models.User
	if err := tx.Where("role = ?", "admin").First(&existingAdmin).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người dùng quản trị đã tồn tại, không thể khởi tạo lại"})
		return
	}

	// 3. 检查用户名是否已存在
	var existingUser models.User
	if err := tx.Where("username = ?", req.AdminUsername).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên đăng nhập đã tồn tại"})
		return
	}

	// 4. 检查邮箱是否已存在
	if err := tx.Where("email = ?", req.AdminEmail).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã tồn tại"})
		return
	}

	// 5. 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mã hóa mật khẩu thất bại"})
		return
	}

	// 6. 创建管理员用户
	admin := models.User{
		Username: req.AdminUsername,
		Password: string(hashedPassword),
		Email:    req.AdminEmail,
		Role:     "admin",
	}

	if err := tx.Create(&admin).Error; err != nil {
		tx.Rollback()
		log.Printf("Tạo người dùng quản trị thất bại: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tạo người dùng quản trị thất bại: " + err.Error()})
		return
	}

	// 7. 创建一些默认的全局角色
	defaultRoles := []models.GlobalRole{
		{
			Name:        "助手",
			Description: "一个友好的AI助手，能够帮助用户解决各种问题",
			Prompt:      "你是一个友好、专业的AI助手。请用简洁明了的语言回答用户的问题，并提供有用的建议。",
			IsDefault:   true,
		},
		{
			Name:        "老师",
			Description: "一位耐心的老师，能够详细解释复杂的概念",
			Prompt:      "你是一位经验丰富的老师。请用通俗易懂的方式解释复杂的概念，并给出具体的例子来帮助理解。",
			IsDefault:   false,
		},
		{
			Name:        "朋友",
			Description: "一个贴心的朋友，能够倾听和陪伴",
			Prompt:      "你是一个贴心的朋友。请用温暖、理解的态度与用户交流，给予情感支持和鼓励。",
			IsDefault:   false,
		},
	}

	for _, role := range defaultRoles {
		if err := tx.Create(&role).Error; err != nil {
			log.Printf("创建默认角色失败: %v", err)
			// 不中断初始化过程，继续执行
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit transaction cơ sở dữ liệu thất bại"})
		return
	}

	log.Printf("Khởi tạo cơ sở dữ liệu thành công，管理员用户: %s", req.AdminUsername)
	c.JSON(http.StatusOK, gin.H{
		"message": "Khởi tạo cơ sở dữ liệu thành công",
		"admin": gin.H{
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
		},
	})
}
