package controllers

import (
	"net/http"
	"strings"

	"backend/entity"
	"backend/pkg/resp"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=6"`
	FirstName   string `json:"firstName" binding:"required"`
	LastName    string `json:"lastName" binding:"required"`
	PhoneNumber string `json:"phoneNumber"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthController struct{ DB *gorm.DB }
func NewAuthController(db *gorm.DB) *AuthController { return &AuthController{DB: db} }

// POST /auth/register
func (a *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error()); return
	}

	var exist entity.User
	if err := a.DB.Where("email = ?", strings.ToLower(req.Email)).First(&exist).Error; err == nil {
		resp.BadRequest(c, "email already registered"); return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil { resp.ServerError(c, err); return }

	user := entity.User{
		Email:       strings.ToLower(req.Email),
		Password:    string(hashed),
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Role:	"customer",
	}

	if err := a.DB.Create(&user).Error; err != nil {
		resp.ServerError(c, err); return
	}

	resp.Created(c, gin.H{
		"id": user.ID, "email": user.Email, "firstName": user.FirstName,
		"lastName": user.LastName, "phoneNumber": user.PhoneNumber, "role": user.Role,
	})
}

// POST /auth/login
func (a *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error()); return
	}

	var user entity.User
	if err := a.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		resp.Unauthorized(c, "invalid credentials"); return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		resp.Unauthorized(c, "invalid credentials"); return
	}

	token, err := utils.GenerateToken(user)
	if err != nil { resp.ServerError(c, err); return }

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"token": token,
		"user": gin.H{
			"id": user.ID, "email": user.Email, "firstName": user.FirstName,
			"lastName": user.LastName, "phoneNumber": user.PhoneNumber, "role": user.Role,
		},
	})
}

// GET /auth/me (ต้อง login)
func (a *AuthController) Me(c *gin.Context) {
	var user entity.User
	idVal, _ := c.Get("userId")
	if err := a.DB.First(&user, idVal).Error; err != nil {
		resp.BadRequest(c, "user not found"); return
	}
	resp.OK(c, gin.H{
		"id": user.ID, "email": user.Email, "firstName": user.FirstName,
		"lastName": user.LastName, "phoneNumber": user.PhoneNumber, "role": user.Role,
	})
}
