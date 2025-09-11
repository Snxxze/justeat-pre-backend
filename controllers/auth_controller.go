package controllers

import (
	"backend/services"
	"backend/utils"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthController รับผิดชอบเฉพาะ HTTP Layer (รับ request, ส่ง response)
type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// POST /auth/register
func (a *AuthController) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=6"`
		FirstName   string `json:"firstName" binding:"required"`
		LastName    string `json:"lastName" binding:"required"`
		PhoneNumber string `json:"phoneNumber"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := a.authService.Register(req.Email, req.Password, req.FirstName, req.LastName, req.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          user.ID,
		"email":       user.Email,
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"phoneNumber": user.PhoneNumber,
		"role":        user.Role,
	})
}

// POST /auth/login
func (a *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := a.authService.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"phoneNumber": user.PhoneNumber,
			"address":     user.Address,
			"role":        user.Role,
			"avatarUrl":	 utils.BuildAvatarURL(user),
		},
	})
}

// GET /auth/me
func (a *AuthController) Me(c *gin.Context) {
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

	user, err := a.authService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"phoneNumber": user.PhoneNumber,
			"address":     user.Address,
			"role":        user.Role,
			"avatarUrl":	 utils.BuildAvatarURL(user),
		},
	})
}

// PATCH /auth/me
func (a *AuthController) UpdateMe(c *gin.Context) {
	// ดึง userId จาก context ที่ middleware set ไว้
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

	// อ่านข้อมูลจาก body
	var req struct {
		FirstName   *string `json:"firstName"`
		LastName    *string `json:"lastName"`
		PhoneNumber *string `json:"phoneNumber"`
		Address     *string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เตรียม map ของ field ที่จะอัปเดต
	updates := map[string]any{}
	if req.FirstName != nil {
		updates["first_name"] = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*req.LastName)
	}
	if req.PhoneNumber != nil {
		updates["phone_number"] = strings.TrimSpace(*req.PhoneNumber)
	}
	if req.Address != nil {
		updates["address"] = strings.TrimSpace(*req.Address)
	}

	// ถ้าไม่มี field ให้แก้ไข → โหลด user เดิมกลับมา
	if len(updates) == 0 {
		user, err := a.authService.GetProfile(userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user})
		return
	}

	// อัปเดตใน DB
	user, err := a.authService.UpdateProfile(userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update profile"})
		return
	}

	// ตอบกลับข้อมูลใหม่
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"phoneNumber": user.PhoneNumber,
			"address":     user.Address,
			"role":        user.Role,
			"avatarUrl":   utils.BuildAvatarURL(user),
		},
	})
}

// POST /auth/me/avatar
func (a *AuthController) UploadAvatar(c *gin.Context) {
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

	fh, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar is required"})
		return
	}

	file, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer file.Close()

	// จำกัดขนาด (5MB)
	const masSize = 5 << 20
	lr := &io.LimitedReader{R: file, N: masSize + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read error"})
		return
	}
	if int64(len(data)) > masSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
		return
	}

	ct := http.DetectContentType(data)
	if !strings.HasPrefix(ct, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "db error"})
		return
	}

	// เรียก service ไปบันทึก
	if err := a.authService.UploadAvatar(userID, data, ct); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /auth/me/avatar
func (a *AuthController) GetAvatar(c *gin.Context) {
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

	u, err := a.authService.GetAvatar(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if len(u.Avatar) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no avatar"})
		return
	}

	c.Data(http.StatusOK, u.AvatarType, u.Avatar)
}