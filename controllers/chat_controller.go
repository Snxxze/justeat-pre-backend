// controllers/chat_controller.go
package controllers

import (
	"net/http"
	"strconv"

	"backend/services"

	"github.com/gin-gonic/gin"
)

type ChatController struct {
	service *services.ChatService
}

func NewChatController(s *services.ChatService) *ChatController {
	return &ChatController{s}
}

// GET /chatrooms (ห้องแชทของ user)
func (c *ChatController) ListRooms(ctx *gin.Context) {
	// สมมติได้ userID จาก JWT
	userID := ctx.GetUint("userID")

	rooms, err := c.service.GetRoomsByUser(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, rooms)
}

// GET /chatrooms/:id/messages
func (c *ChatController) ListMessages(ctx *gin.Context) {
	roomID, _ := strconv.Atoi(ctx.Param("id"))

	msgs, err := c.service.GetMessages(uint(roomID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, msgs)
}

// POST /chatrooms/:id/messages
func (c *ChatController) SendMessage(ctx *gin.Context) {
	roomID, _ := strconv.Atoi(ctx.Param("id"))
	userID := ctx.GetUint("userID")

	var req struct {
		Body string `json:"body"`
		Type uint   `json:"typeMessageId"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	msg, err := c.service.SendMessage(uint(roomID), userID, req.Type, req.Body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, msg)
}
