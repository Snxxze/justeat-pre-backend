package controllers

import (
	"net/http"
	"strconv"
	"errors"

	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct{ Svc *services.OrderService }

func NewOrderController(s *services.OrderService) *OrderController { return &OrderController{Svc: s} }

// POST /orders
func (h *OrderController) Create(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req services.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Svc.Create(uid, &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
    }
    // ที่เหลือถือเป็นปัญหา validation → 400
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
	}

	c.JSON(http.StatusCreated, res) // { id, total }
}

// GET /profile/orders?limit=50
func (h *OrderController) ListForMe(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Svc.ListForUser(uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /orders/:id
func (h *OrderController) Detail(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	out, err := h.Svc.DetailForUser(uid, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, out)
}

// POST /orders/checkout-from-cart
func (h *OrderController) CheckoutFromCart(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
	uid, ok := v.(uint)
	if !ok || uid == 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }

	var req services.CheckoutFromCartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// FE จะส่ง address เสมอ ถ้าไม่มาก็ invalid
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Svc.CreateFromCart(uid, &req)
	if err != nil {
		switch err.Error() {
		case "cart is empty", "cart has no restaurant":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, res)
}

