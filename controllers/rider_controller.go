package controllers

import (
	"backend/services"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RiderController struct{ Svc *services.RiderService }

func NewRiderController(s *services.RiderService) *RiderController { return &RiderController{Svc: s} }

// ======================== Profile DTOs ========================

type RiderMeResp struct {
	// from users
	UserID       uint   `json:"userId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	PhoneNumber  string `json:"phoneNumber"`
	AvatarBase64 string `json:"avatarBase64,omitempty"`
	// from riders
	RiderID      *uint  `json:"riderId,omitempty"`
	NationalID   string `json:"nationalId,omitempty"`
	VehiclePlate string `json:"vehiclePlate,omitempty"`
	Zone         string `json:"zone,omitempty"`
	License      string `json:"license,omitempty"`
	DriveCard    string `json:"driveCard,omitempty"` // base64 (no header)
}

type UpdateRiderReq struct {
	NationalID      string  `json:"nationalId" binding:"required"`
	VehiclePlate    string  `json:"vehiclePlate" binding:"required"`
	Zone            string  `json:"zone" binding:"required"`
	License         string  `json:"license" binding:"required"`
	DriveCardBase64 *string `json:"driveCardBase64,omitempty"` // nil=ไม่แตะ, ""=ลบ, "dataURL"=อัปเดต
}

var reNatID = regexp.MustCompile(`^\d{13}$`)

// ======================== GET /rider/me ========================

func (h *RiderController) Me(c *gin.Context) {
	uid := c.GetUint("userId")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	snap, err := h.Svc.GetProfileSnapshot(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := RiderMeResp{
		UserID:       snap.User.ID,
		FirstName:    snap.User.FirstName,
		LastName:     snap.User.LastName,
		PhoneNumber:  snap.User.PhoneNumber,
		AvatarBase64: snap.User.AvatarBase64,
	}
	if snap.Rider != nil {
		resp.RiderID = &snap.Rider.ID
		resp.NationalID = snap.Rider.NationalID
		resp.VehiclePlate = snap.Rider.VehiclePlate
		resp.Zone = snap.Rider.Zone
		resp.License = snap.Rider.License
		resp.DriveCard = snap.Rider.DriveCard // เก็บแบบไม่มี header
	}

	c.JSON(http.StatusOK, resp)
}

// ======================== PUT /rider/me ========================

func (h *RiderController) UpdateMe(c *gin.Context) {
	uid := c.GetUint("userId")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateRiderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !reNatID.MatchString(strings.TrimSpace(req.NationalID)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nationalId (must be 13 digits)"})
		return
	}

	// เตรียมรูปแบบสำหรับ service (ใช้ helper ที่มีอยู่แล้วใน package นี้)
	var cleanB64 *string
	if req.DriveCardBase64 != nil {
		trim := strings.TrimSpace(*req.DriveCardBase64)
		if trim == "" {
			empty := ""
			cleanB64 = &empty // ลบรูป
		} else {
			// เรียกใช้ stripDataURLHeader ที่ประกาศไว้ (ครั้งเดียว) ใน payment_controller.go
			v, err := stripDataURLHeader(trim)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid driveCard base64"})
				return
			}
			cleanB64 = &v // อัปเดตรูป
		}
	}

	err := h.Svc.UpsertRiderProfile(uid, services.UpsertRiderProfileInput{
		NationalID:   strings.TrimSpace(req.NationalID),
		VehiclePlate: strings.TrimSpace(req.VehiclePlate),
		Zone:         strings.TrimSpace(req.Zone),
		License:      strings.TrimSpace(req.License),
		DriveCardB64: cleanB64, // nil=ไม่แตะ
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ======================== ของเดิมของคุณ (คงไว้) ========================

func (h *RiderController) SetAvailability(c *gin.Context) {
	uid := c.GetUint("userId")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch strings.ToUpper(strings.TrimSpace(req.Status)) {
	case "ONLINE":
		if err := h.Svc.SetAvailability(uid, true); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
	case "OFFLINE":
		if err := h.Svc.SetAvailability(uid, false); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RiderController) Accept(c *gin.Context) {
	uid := c.GetUint("userId")
	oid64, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if oid64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	if err := h.Svc.AcceptWork(uid, uint(oid64)); err != nil {
		code := http.StatusBadRequest
		msg := err.Error()
		if strings.Contains(msg, "not ONLINE") || strings.Contains(msg, "active work") ||
			strings.Contains(msg, "already assigned") || strings.Contains(msg, "not in preparing") {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RiderController) Complete(c *gin.Context) {
	uid := c.GetUint("userId")
	oid64, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)
	if oid64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	if err := h.Svc.CompleteWork(uid, uint(oid64)); err != nil {
		code := http.StatusBadRequest
		msg := err.Error()
		if strings.Contains(msg, "not in delivering") ||
			strings.Contains(msg, "not on an assigned work") ||
			strings.Contains(msg, "no active work") {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RiderController) ListAvailable(c *gin.Context) {
	items, err := h.Svc.ListAvailable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *RiderController) GetStatus(c *gin.Context) {
	uid := c.GetUint("userId")

	status, err := h.Svc.GetStatus(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status) // {status: "ONLINE", isWorking: true}
}

func (h *RiderController) GetCurrentWork(c *gin.Context) {
	uid := c.GetUint("userId")
	work, err := h.Svc.GetCurrentWork(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if work == nil {
		c.JSON(http.StatusOK, gin.H{"work": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work": work})
}
