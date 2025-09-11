package controllers

import (
	"backend/services"
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

	c.JSON(http.StatusCreated, gin.H{"ok": true, "user": user})
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
		"user":  user,
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

	c.JSON(http.StatusOK, gin.H{"ok": true, "user": user})
}

// PATCH /auth/me
func (a *AuthController) UpdateMe(c *gin.Context) {
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

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

	user, err := a.authService.UpdateProfile(userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// POST /auth/me/avatar
func (a *AuthController) UploadAvatar(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	var req struct {
		AvatarBase64 string `json:"avatarBase64" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64"})
		return
	}

	if err := a.authService.UploadAvatarBase64(userID, req.AvatarBase64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, _ := a.authService.GetProfile(userID)
	c.JSON(http.StatusOK, gin.H{"ok": true, "user": user})
}

// GET /auth/me/avatar
func (a *AuthController) GetAvatar(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	b64, err := a.authService.GetAvatarBase64(userID)
	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
	}
	if b64 == "" {
			// ✅ ส่ง default แทน ไม่ต้อง 404
			c.JSON(http.StatusOK, gin.H{"avatarBase64": ""})
			return
	}

	c.JSON(http.StatusOK, gin.H{"avatarBase64": b64})
}

// GET /auth/me/restaurant
func (a *AuthController) MeRestaurant(c *gin.Context) {
	userID := c.GetUint("userId")
	role := c.GetString("role")

	if role != "owner" {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: not an owner"})
			return
	}

	restaurant, err := a.authService.GetRestaurantByUserID(userID)
	if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "restaurant not found"})
			return
	}

	c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"restaurant": restaurant,
	})
}
