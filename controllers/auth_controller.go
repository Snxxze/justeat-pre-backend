package controllers

import (
	"net/http"
	"strings"
	"strconv"

	"backend/entity"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Request models
type RegisterRequest struct {
	Email       string `json:"email" 			binding:"required,email"`
	Password    string `json:"password" 	binding:"required,min=6"`
	FirstName   string `json:"firstName" 	binding:"required"`
	LastName    string `json:"lastName" 	binding:"required"`
	PhoneNumber string `json:"phoneNumber"`
}
type LoginRequest struct {
	Email    string `json:"email" 		binding:"required,email"`
	Password string `json:"password" 	binding:"required"`
}

type UpdateMeRequest struct {
  FirstName   *string `json:"firstName"   binding:"omitempty,min=1,max=100"`
  LastName    *string `json:"lastName"    binding:"omitempty,min=1,max=100"`
  PhoneNumber *string `json:"phoneNumber" binding:"omitempty,max=32"`
  Address     *string `json:"address"     binding:"omitempty,max=255"`
}

type AuthController struct{ 
	DB *gorm.DB 
}

func NewAuthController(db *gorm.DB) *AuthController { 
	return &AuthController{DB: db} 
}

// POST /auth/register
func (a *AuthController) Register(c *gin.Context) {
	// ตรวจรูปแบบ
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ข้อมูลที่สำคัญ
	email := strings.ToLower(strings.TrimSpace(req.Email))
	first := strings.TrimSpace(req.FirstName)
	last 	:= strings.TrimSpace(req.LastName)
	phone := strings.TrimSpace(req.PhoneNumber)

	// ตรวจสอบว่าอีเมลถูกใช้ลงทะเบียนไปแล้วหรือยัง
	var count int64
	if err := a.DB.Model(&entity.User{}).
		Where("email = ?", email).
		Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}

	// สร้าง
	user := entity.User {
		Email: 				email,
		Password: 		string(hashed),
		FirstName: 		first,
		LastName: 		last,
		PhoneNumber: 	phone,
		Role: 				"customer",
	}
	// บันทึกลง
	if err := a.DB.Create(&user).Error; err != nil {
		// กันกรณีส่ง email เหมือนกันมา เวลาพร้อมๆกัน
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// HTTP 201 สำเร็จ
	// ข้อมูลที่ส่งกลับ resp
	c.JSON(http.StatusCreated, gin.H{
		"id":						user.ID,
		"email": 				user.Email,
		"firstName": 		user.FirstName,
		"lastName": 		user.LastName,
		"phoneNumber": 	user.PhoneNumber,
		"role":					user.Role,
	})
}

// POST /auth/login
func (a *AuthController) Login(c *gin.Context) {
	// ตรวจรูปแบบ
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// หา user
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var user entity.User
	if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil {
		// HTTP 401 (Unauthorized)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// เทียบรหัส
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// ออก token ให้
	token, err := utils.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot generate token"})
		return
	}
	
	// ตอบกลับ
	c.JSON(http.StatusOK, gin.H{
		"ok":	true,
		"token": token,
		"user": gin.H{
			"id":	user.ID,
			"email": user.Email,
			"firstName": user.FirstName,
			"lastName": user.LastName,
			"phoneNumber": user.PhoneNumber,
			"address": user.Address,
			"role": user.Role,
		},
	})
}

// GET /auth/me (ต้อง login)
func (a *AuthController) Me(c *gin.Context) {
	// ดึง userId 
	userID, ok := userIDFromCtx(c)
	if !ok {
		// ไม่มี หรือ ไม่ได้ login
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// ดึงข้อมูลผู้ใช้
	var user entity.User
	if err := a.DB.
		Select("id, email, first_name, last_name, phone_number, address, role").
		First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// ตอบกลับ
	c.JSON(http.StatusOK, gin.H{
		"id":						user.ID,
		"email":				user.Email,
		"firstName":		user.FirstName,
		"lastName": 		user.LastName,
		"phoneNumber": 	user.PhoneNumber,
		"address":			user.Address,
		"role": 				user.Role,
	})
}

// helper: ดึง userId จาก context ให้ครอบทุกเคสชนิดข้อมูลที่ middleware อาจใส่มา
func userIDFromCtx(c *gin.Context) (uint, bool) {
	v, exists := c.Get("userId")
	if !exists || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case uint:
		return t, true
	case uint64:
		return uint(t), true
	case int:
		if t > 0 {
			return uint(t), true
		}
	case int64:
		if t > 0 {
			return uint(t), true
		}
	case string:
		if id64, err := strconv.ParseUint(t, 10, 64); err == nil {
			return uint(id64), true
		}
	}
	return 0, false
}

func (a *AuthController) UpdateMeRequest(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3) เตรียม fields ที่จะอัปเดต (เฉพาะที่ส่งมา)
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

	// ถ้าไม่มีอะไรให้อัปเดต ก็คืนข้อมูลเดิม
	if len(updates) == 0 {
		var u entity.User
		if err := a.DB.
			Select("id, email, first_name, last_name, phone_number, address, role").
			First(&u, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":          u.ID,
				"email":       u.Email,
				"firstName":   u.FirstName,
				"lastName":    u.LastName,
				"phoneNumber": u.PhoneNumber,
				"address":     u.Address,
				"role":        u.Role,
			},
		})
		return
	}

	// อัปเดต
	if err := a.DB.Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// โหลดใหม่แล้วตอบกลับ (แน่ใจว่า FE ได้ค่าล่าสุด)
	var user entity.User
	if err := a.DB.
		Select("id, email, first_name, last_name, phone_number, address, role").
		First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reload user failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"phoneNumber": user.PhoneNumber,
			"address":     user.Address,
			"role":        user.Role,
		},
	})
}