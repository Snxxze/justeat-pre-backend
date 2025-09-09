package controllers

import (
	"backend/entity"
	"backend/services"
	"backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RiderApplicationController struct{ Service *services.RiderApplicationService }

func NewRiderApplicationController(s *services.RiderApplicationService) *RiderApplicationController {
	return &RiderApplicationController{Service: s}
}

// ===== Request DTO =====
type ApplyRiderReq struct {
	VehiclePlate    string `json:"vehiclePlate" binding:"required"`
	License         string `json:"license" binding:"required"`
	DriveCarPicture string `json:"driveCarPicture"` // base64
}

// ===== Response DTO =====
type ApplyResp struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

type ListItem struct {
	ID           uint   `json:"id"`
	VehiclePlate string `json:"vehiclePlate"`
	License      string `json:"license"`
	DriveCar     string `json:"driveCarPicture"` // << สำคัญ: key ที่ FE ใช้
	Status       string `json:"status"`
	SubmittedAt  string `json:"submittedAt"`
	User         struct {
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
	} `json:"user"`
}

type ApproveResp struct {
	ApplicationID uint   `json:"applicationId"`
	RiderID       uint   `json:"riderId"`
	Status        string `json:"status"`
	OwnerUserID   uint   `json:"ownerUserId"`
	NewRole       string `json:"newRole"`
}

type RejectRiderReq struct {
	Reason  string `json:"reason" binding:"required"`
	AdminID *uint  `json:"adminId,omitempty"`
}

type RejectResp struct {
	ApplicationID uint   `json:"applicationId"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

// ===== ผู้ใช้ ยื่นสมัคร =====
func (ctl *RiderApplicationController) Apply(c *gin.Context) {
	var req ApplyRiderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	app := entity.RiderApplication{
		UserID:       utils.CurrentUserID(c),
		VehiclePlate: req.VehiclePlate,
		License:      req.License,
		DriveCar:     req.DriveCarPicture, // เก็บรูป base64 ลงฟิลด์ application
		// Status = pending เซ็ตใน service
	}

	id, err := ctl.Service.Apply(&app)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

	c.JSON(http.StatusCreated, ApplyResp{ID: id, Status: "pending"})
}

// ===== ผู้ใช้ ดูใบสมัครตัวเอง (option: ?status=) =====
func (ctl *RiderApplicationController) ListMine(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	userID := utils.CurrentUserID(c)

	apps, err := ctl.Service.ListMine(userID, status)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

	out := make([]ListItem, 0, len(apps))
	for _, a := range apps {
		var it ListItem
		it.ID = a.ID
		it.VehiclePlate = a.VehiclePlate
		it.License = a.License
		it.Status = a.Status
		it.SubmittedAt = a.CreatedAt.Format(time.RFC3339)
		it.User.FirstName = a.User.FirstName
		it.User.LastName = a.User.LastName
		it.User.Email = a.User.Email
		it.User.PhoneNumber = a.User.PhoneNumber
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// ===== แอดมิน ดูรายการตามสถานะ =====
func (ctl *RiderApplicationController) List(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	apps, err := ctl.Service.List(status)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }

	out := make([]ListItem, 0, len(apps))
	for _, a := range apps {
		var it ListItem
		it.ID = a.ID
		it.VehiclePlate = a.VehiclePlate
		it.License = a.License
		it.DriveCar = a.DriveCar
		it.Status = a.Status
		it.SubmittedAt = a.CreatedAt.Format(time.RFC3339)
		it.User.FirstName = a.User.FirstName
		it.User.LastName = a.User.LastName
		it.User.Email = a.User.Email
		it.User.PhoneNumber = a.User.PhoneNumber
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// ===== แอดมิน อนุมัติ =====
func (ctl *RiderApplicationController) Approve(c *gin.Context) {
    idStr := c.Param("id")
    appID64, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

    var req services.RiderApproveReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }

    r, owner, err := ctl.Service.Approve(uint(appID64), req)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }

    newRole := owner.Role
    if req.NewRole != nil && *req.NewRole != "" {
        newRole = *req.NewRole
    }

    c.JSON(http.StatusOK, ApproveResp{
        ApplicationID: uint(appID64),
        RiderID:       r.ID,
        Status:        "approved",
        OwnerUserID:   owner.ID,
        NewRole:       newRole,
    })
}


// ===== แอดมิน ปฏิเสธ =====
func (ctl *RiderApplicationController) Reject(c *gin.Context) {
	idStr := c.Param("id")
	appID64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

	var req RejectRiderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	if err := ctl.Service.Reject(uint(appID64), req.Reason, req.AdminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	c.JSON(http.StatusOK, RejectResp{
		ApplicationID: uint(appID64),
		Status:        "rejected",
		Reason:        req.Reason,
	})
}