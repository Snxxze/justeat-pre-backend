package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct{ Svc *services.OrderService }

func NewOrderController(s *services.OrderService) *OrderController {
	return &OrderController{Svc: s}
}

// ---------- DTOs ----------
type PaymentSummaryDTO struct {
	MethodID   uint       `json:"methodId"`
	MethodName string     `json:"methodName"`
	StatusID   uint       `json:"statusId"`
	StatusName string     `json:"statusName"`
	PaidAt     *time.Time `json:"paidAt,omitempty"`
}

// ตอบรายละเอียดออเดอร์ + แนบ paymentSummary
type OrderDetailWithPayment struct {
	*services.OrderDetail
	PaymentSummary *PaymentSummaryDTO `json:"paymentSummary,omitempty"`
}

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

// GET /orders/:id  (?withPayment=1 → แนบ payment summary)
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

	withPayment := c.DefaultQuery("withPayment", "0") == "1"
	resp := OrderDetailWithPayment{OrderDetail: out}
	if withPayment {
		if ps, err := h.Svc.PaymentSummaryForOrder(uid, uint(id)); err == nil && ps != nil {
			resp.PaymentSummary = &PaymentSummaryDTO{
				MethodID:   ps.MethodID,
				MethodName: ps.MethodName,
				StatusID:   ps.StatusID,
				StatusName: ps.StatusName,
				PaidAt:     ps.PaidAt,
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ---------- Checkout ----------

// POST /orders/checkout-from-cart
func (h *OrderController) CheckoutFromCart(c *gin.Context) {
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

	var req services.CheckoutFromCartReq
	if err := c.ShouldBindJSON(&req); err != nil {
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
