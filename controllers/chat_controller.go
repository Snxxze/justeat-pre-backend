package controllers

import (
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChatController struct {
	Service *services.ChatService
}

func NewChatController(s *services.ChatService) *ChatController {
	return &ChatController{Service: s}
}

// GET /orders/:id/chatroom
func (ctl *ChatController) GetOrCreateRoom(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	uidAny, _ := c.Get("userId")
	userID := uidAny.(uint)

	// ✅ ตรวจสิทธิ์ก่อน
	ok, err := ctl.Service.CanAccessRoom(userID, uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	room, err := ctl.Service.GetOrCreateRoom(uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"room": room})
}

// GET /orders/:id/messages
func (ctl *ChatController) GetMessages(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	uidAny, _ := c.Get("userId")
	userID := uidAny.(uint)

	// ✅ ตรวจสิทธิ์ก่อน
	ok, err := ctl.Service.CanAccessRoom(userID, uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	room, err := ctl.Service.GetOrCreateRoom(uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msgs, err := ctl.Service.GetMessages(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// POST /orders/:id/messages
func (ctl *ChatController) SendMessage(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var req struct {
		Body          string `json:"body"`
		TypeMessageID uint   `json:"typeMessageId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uidAny, _ := c.Get("userId")
	userID := uidAny.(uint)

	// ✅ ตรวจสิทธิ์ก่อน
	ok, err := ctl.Service.CanAccessRoom(userID, uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
		return
	}

	room, err := ctl.Service.GetOrCreateRoom(uint(orderID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msg, err := ctl.Service.SendMessage(room.ID, userID, req.TypeMessageID, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}
