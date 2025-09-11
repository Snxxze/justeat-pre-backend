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

type OrderController struct {
	OrderService *services.OrderService
}

func NewOrderController(s *services.OrderService) *OrderController {
	return &OrderController{OrderService: s}
}

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

// ---------- Orders ----------

// POST /orders
func (h *OrderController) Create(c *gin.Context) {
	userVal, ok := c.Get("userId")
	if !ok || userVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userVal.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var orderReq services.CreateOrderReq
	if err := c.ShouldBindJSON(&orderReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderRes, err := h.OrderService.Create(userID, &orderReq)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, orderRes) // { id, total }
}

// GET /profile/orders?limit=50
func (h *OrderController) ListForMe(c *gin.Context) {
	userVal, ok := c.Get("userId")
	if !ok || userVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userVal.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	orderList, err := h.OrderService.ListForUser(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orderList})
}

// GET /orders/:id (?withPayment=1 → แนบ payment summary)
func (h *OrderController) Detail(c *gin.Context) {
	userVal, ok := c.Get("userId")
	if !ok || userVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userVal.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil || orderID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	orderDetail, err := h.OrderService.DetailForUser(userID, uint(orderID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	withPayment := c.DefaultQuery("withPayment", "0") == "1"
	resp := OrderDetailWithPayment{OrderDetail: orderDetail}
	if withPayment {
		if paymentSummary, err := h.OrderService.PaymentSummaryForOrder(userID, uint(orderID)); err == nil && paymentSummary != nil {
			resp.PaymentSummary = &PaymentSummaryDTO{
				MethodID:   paymentSummary.MethodID,
				MethodName: paymentSummary.MethodName,
				StatusID:   paymentSummary.StatusID,
				StatusName: paymentSummary.StatusName,
				PaidAt:     paymentSummary.PaidAt,
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ---------- Checkout ----------

// POST /orders/checkout-from-cart
func (h *OrderController) CheckoutFromCart(c *gin.Context) {
	userVal, ok := c.Get("userId")
	if !ok || userVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userVal.(uint)
	if !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var checkoutReq services.CheckoutFromCartReq
	if err := c.ShouldBindJSON(&checkoutReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	checkoutRes, err := h.OrderService.CreateFromCart(userID, &checkoutReq)
	if err != nil {
		switch err.Error() {
		case "cart is empty", "cart has no restaurant":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, checkoutRes)
}
