// controllers/rider_application_controller.go
package controllers

import (
	"backend/entity"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RiderApplicationController struct {
	DB *gorm.DB
}

func NewRiderApplicationController(db *gorm.DB) *RiderApplicationController {
	return &RiderApplicationController{DB: db}
}

// -------- Request DTO --------
type ApplyRiderReq struct {
	VehiclePlate string `json:"vehiclePlate" binding:"required"`
	License      string `json:"license" binding:"required"`
	DriveCar     string `json:"driveCarPicture"` // base64
}

// -------- Response DTO --------
type RiderApplicationResponse struct {
	ID           uint   `json:"id"`
	VehiclePlate string `json:"vehiclePlate"`
	License      string `json:"license"`
	DriveCar     string `json:"driveCarPicture"`

	Status     string `json:"status"`
	SubmittedAt string `json:"submittedAt"`

	User struct {
		ID        uint   `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	} `json:"user"`
}

type ApplyRiderResponse struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

type ApproveRiderResponse struct {
	ApplicationID uint   `json:"applicationId"`
	RiderID       uint   `json:"riderId"`
	Status        string `json:"status"`
	UserID        uint   `json:"userId"`
	NewRole       string `json:"newRole"`
}

type RejectRiderResponse struct {
	ApplicationID uint   `json:"applicationId"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

// ========== User สมัคร Rider ==========
func (ctl *RiderApplicationController) Apply(c *gin.Context) {
	var req ApplyRiderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uidAny, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	app := entity.RiderApplication{
		VehiclePlate: req.VehiclePlate,
		License:      req.License,
		DriveCar:     req.DriveCar,
		UserID:       userID,
		Status:       "pending",
	}

	if err := ctl.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ApplyRiderResponse{ID: app.ID, Status: "pending"})
}

// ========== User ดูใบสมัครของตัวเอง ==========
func (ctl *RiderApplicationController) ListMine(c *gin.Context) {
	uidAny, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	status := c.Query("status")

	var apps []entity.RiderApplication
	q := ctl.DB.Preload("User").Where("user_id = ?", userID).Order("id DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := []RiderApplicationResponse{}
	for _, app := range apps {
		item := RiderApplicationResponse{
			ID:           app.ID,
			VehiclePlate: app.VehiclePlate,
			License:      app.License,
			DriveCar:     app.DriveCar,
			Status:       app.Status,
			SubmittedAt:  app.CreatedAt.Format(time.RFC3339),
		}
		item.User.ID = app.User.ID
		item.User.FirstName = app.User.FirstName
		item.User.LastName = app.User.LastName
		item.User.Email = app.User.Email
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ========== Admin ดูรายการ ==========
func (ctl *RiderApplicationController) List(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")

	var apps []entity.RiderApplication
	if err := ctl.DB.Preload("User").Where("status = ?", status).Order("id DESC").Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := []RiderApplicationResponse{}
	for _, app := range apps {
		item := RiderApplicationResponse{
			ID:           app.ID,
			VehiclePlate: app.VehiclePlate,
			License:      app.License,
			DriveCar:     app.DriveCar,
			Status:       app.Status,
			SubmittedAt:  app.CreatedAt.Format(time.RFC3339),
		}
		item.User.ID = app.User.ID
		item.User.FirstName = app.User.FirstName
		item.User.LastName = app.User.LastName
		item.User.Email = app.User.Email
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ========== Admin อนุมัติ ==========
func (ctl *RiderApplicationController) Approve(c *gin.Context) {
	appID, _ := strconv.Atoi(c.Param("id"))

	uidAny, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	// หา admin
	var admin entity.Admin
	if err := ctl.DB.Where("user_id = ?", userID).First(&admin).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not an admin"})
		return
	}

	var app entity.RiderApplication
	if err := ctl.DB.First(&app, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if app.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application is not pending"})
		return
	}

	// หา id ของสถานะ ONLINE
	var statusID uint
	ctl.DB.Model(&entity.RiderStatus{}).
		Select("id").Where("status_name = ?", "ONLINE").
		Scan(&statusID)
	if statusID == 0 {
		// fallback: OFFLINE
		ctl.DB.Model(&entity.RiderStatus{}).
			Select("id").Where("status_name = ?", "OFFLINE").
			Scan(&statusID)
	}

	rider := entity.Rider{
		UserID:        app.UserID,
		VehiclePlate:  app.VehiclePlate,
		License:       app.License,
		DriveCard:     app.DriveCar,
		RiderStatusID: statusID,
		AdminID:       &admin.ID,
	}

	now := time.Now()
	tx := ctl.DB.Begin()
	if err := tx.Create(&rider).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// อัปเกรด role user เป็น rider
	if err := tx.Model(&entity.User{}).
		Where("id = ?", app.UserID).
		Where("role = '' OR role = 'customer' OR role IS NULL").
		Update("role", "rider").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	app.Status = "approved"
	app.ReviewedAt = &now
	app.AdminID = &admin.ID
	if err := tx.Save(&app).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tx.Commit()

	var user entity.User
	ctl.DB.First(&user, app.UserID)

	c.JSON(http.StatusOK, ApproveRiderResponse{
		ApplicationID: uint(appID),
		RiderID:       rider.ID,
		Status:        "approved",
		UserID:        user.ID,
		NewRole:       user.Role,
	})
}

// ========== Admin ปฏิเสธ ==========
type RejectRiderReq struct {
	Reason string `json:"reason" binding:"required"`
}

func (ctl *RiderApplicationController) Reject(c *gin.Context) {
	appID, _ := strconv.Atoi(c.Param("id"))

	var req RejectRiderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var app entity.RiderApplication
	if err := ctl.DB.First(&app, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if app.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot reject with status " + app.Status})
		return
	}

	now := time.Now()
	app.Status = "rejected"
	app.ReviewedAt = &now
	app.AdminID = nil
	app.RejectReason = &req.Reason

	if err := ctl.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RejectRiderResponse{
		ApplicationID: uint(appID),
		Status:        "rejected",
		Reason:        req.Reason,
	})
}
