package controllers

import (
	"backend/entity"
	"backend/services"
	"backend/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type RestaurantApplicationController struct {
	Service *services.RestaurantApplicationService
}

func NewRestaurantApplicationController(s *services.RestaurantApplicationService) *RestaurantApplicationController {
	return &RestaurantApplicationController{Service: s}
}

// ===== helpers =====
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func isValidPromptPay(digits string) bool {
	// 10 = เบอร์มือถือ, 13 = เลขบัตรประชาชน
	return len(digits) == 10 || len(digits) == 13
}

// ====== Request DTO ======
type ApplyRestaurantReq struct {
	Name                 string `json:"name" binding:"required"`
	Phone                string `json:"phone"`
	Address              string `json:"address"`
	Description          string `json:"description"`
	PictureBase64        string `json:"pictureBase64"`
	OpeningTime          string `json:"openingTime" binding:"required"`
	ClosingTime          string `json:"closingTime" binding:"required"`
	RestaurantCategoryID uint   `json:"restaurantCategoryId" binding:"required"`

	// ใหม่: PromptPay (รับทั้งเบอร์/บัตร ป้อนมาอย่างไรก็ได้ เดี๋ยว server เก็บเฉพาะตัวเลข)
	PromptPay string `json:"promptPay" binding:"required"`
}

// ====== Response DTO ======
type RestaurantApplicationResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	OpeningTime string `json:"openingTime"`
	ClosingTime string `json:"closingTime"`
	SubmittedAt string `json:"submittedAt"`

	RestaurantCategory struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"restaurantCategory"`

	OwnerUser struct {
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
	} `json:"ownerUser"`

	Status string `json:"status"`
	// หมายเหตุ: ไม่ส่ง PromptPay ออกใน list response เพื่อลดความเสี่ยงข้อมูลอ่อนไหว
}

// Apply Response
type ApplyResponse struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

// Approve Response
type ApproveResponse struct {
	ApplicationID uint   `json:"applicationId"`
	RestaurantID  uint   `json:"restaurantId"`
	Status        string `json:"status"`
	OwnerUserID   uint   `json:"ownerUserId"`
	NewRole       string `json:"newRole"`
}

// Reject Response
type RejectResponse struct {
	ApplicationID uint   `json:"applicationId"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

// ====== User สมัครร้าน ======
func (ctl *RestaurantApplicationController) Apply(c *gin.Context) {
	var req ApplyRestaurantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// sanitize & validate PromptPay
	pp := onlyDigits(req.PromptPay)
	if !isValidPromptPay(pp) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid promptPay: ต้องเป็นเบอร์ 10 หลัก หรือเลขบัตรประชาชน 13 หลัก",
		})
		return
	}

	app := entity.RestaurantApplication{
		Name:                 req.Name,
		Phone:                req.Phone,
		Address:              req.Address,
		Description:          req.Description,
		OpeningTime:          req.OpeningTime,
		ClosingTime:          req.ClosingTime,
		RestaurantCategoryID: req.RestaurantCategoryID,
		OwnerUserID:          utils.CurrentUserID(c),

		// map ค่า PromptPay ลง entity (คอลัมน์ prompt_pay)
		PromptPay: pp,
	}

	id, err := ctl.Service.Apply(&app, req.PictureBase64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ApplyResponse{ID: id, Status: "pending"})
}

// ====== Admin ดูรายการ ======
func (ctl *RestaurantApplicationController) List(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	apps, err := ctl.Service.List(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []RestaurantApplicationResponse
	for _, app := range apps {
		item := RestaurantApplicationResponse{
			ID:          app.ID,
			Name:        app.Name,
			Address:     app.Address,
			Description: app.Description,
			Logo:        app.Picture,
			Phone:       app.Phone,
			OpeningTime: app.OpeningTime,
			ClosingTime: app.ClosingTime,
			SubmittedAt: app.CreatedAt.Format(time.RFC3339),
			Status:      app.Status,
		}
		item.RestaurantCategory.ID = app.RestaurantCategory.ID
		item.RestaurantCategory.Name = app.RestaurantCategory.CategoryName
		item.OwnerUser.FirstName = app.OwnerUser.FirstName
		item.OwnerUser.LastName = app.OwnerUser.LastName
		item.OwnerUser.Email = app.OwnerUser.Email
		item.OwnerUser.PhoneNumber = app.OwnerUser.PhoneNumber

		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ====== Admin อนุมัติ ======
type ApproveReq = services.ApproveReq

func (ctl *RestaurantApplicationController) Approve(c *gin.Context) {
	idStr := c.Param("id")
	appID, _ := strconv.Atoi(idStr)

	var req ApproveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rest, owner, err := ctl.Service.Approve(uint(appID), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ApproveResponse{
		ApplicationID: uint(appID),
		RestaurantID:  rest.ID,
		Status:        "approved",
		OwnerUserID:   owner.ID,
		NewRole:       owner.Role,
	})
}

// ====== Admin ปฏิเสธ ======
type RejectReq struct {
	Reason  string `json:"reason" binding:"required"`
	AdminID *uint  `json:"adminId,omitempty"`
}

func (ctl *RestaurantApplicationController) Reject(c *gin.Context) {
	idStr := c.Param("id")
	appID, _ := strconv.Atoi(idStr)

	var req RejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctl.Service.Reject(uint(appID), req.Reason, req.AdminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RejectResponse{
		ApplicationID: uint(appID),
		Status:        "rejected",
		Reason:        req.Reason,
	})
}
