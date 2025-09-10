package controllers

import (
	"backend/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RiderController struct{ Svc *services.RiderService }

func NewRiderController(s *services.RiderService) *RiderController { return &RiderController{Svc: s} }

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
	// จะคืนเป็น {items: [...]} หรือ [...] ก็ได้ เลือกแบบนี้ให้ FE รองรับง่าย
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
