package controllers

import (
	"errors"
	"net/http"
	"strings"

	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CartController struct{ Svc *services.CartService }
func NewCartController(s *services.CartService) *CartController { return &CartController{Svc: s} }

// GET /cart
func (h *CartController) Get(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	cart, subtotal, err := h.Svc.Get(uid)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"cart": cart, "subtotal": subtotal})
}

// POST /cart/items
func (h *CartController) Add(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	var req services.AddToCartIn
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if err := h.Svc.Add(uid, &req); err != nil {
    if strings.Contains(err.Error(), "another restaurant") {
        c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        return
    }
    if errors.Is(err, gorm.ErrRecordNotFound) {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    // ที่เหลือเป็น 400
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// PATCH /cart/items/qty
func (h *CartController) UpdateQty(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	var body struct {
		ItemID uint `json:"itemId" binding:"required"`
		Qty    int  `json:"qty" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if err := h.Svc.UpdateQty(uid, body.ItemID, body.Qty); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /cart/items
func (h *CartController) RemoveItem(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	var body struct {
		ItemID uint `json:"itemId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if err := h.Svc.RemoveItem(uid, body.ItemID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /cart
func (h *CartController) Clear(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	if err := h.Svc.Clear(uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
