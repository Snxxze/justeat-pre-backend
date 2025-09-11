package controllers

import (
	"backend/entity"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct {
	DB *gorm.DB
}

func NewOrderController(db *gorm.DB) *OrderController {
	return &OrderController{DB: db}
}

// DTOs
type OrderItemIn struct {
	MenuID uint `json:"menuId"`
	Qty    int  `json:"qty"`
}

type CreateOrderReq struct {
	RestaurantID uint          `json:"restaurantId"`
	Items        []OrderItemIn `json:"items"`
	Address      string        `json:"address"`
}

type CreateOrderRes struct {
	ID    uint  `json:"id"`
	Total int64 `json:"total"`
}

type OrderSummary struct {
	ID            uint      `json:"id"`
	RestaurantID  uint      `json:"restaurantId"`
	Total         int64     `json:"total"`
	OrderStatusID uint      `json:"orderStatusId"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CheckoutFromCartReq struct {
	Address string `json:"address"`
}

// ---------- Orders ----------

// POST /orders
func (h *OrderController) Create(c *gin.Context) {
	userVal, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userVal.(uint)

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items required"})
		return
	}

	var subtotal int64
	for _, it := range req.Items {
		var menu entity.Menu
		if err := h.DB.First(&menu, it.MenuID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "menu not found"})
			return
		}
		subtotal += menu.Price * int64(it.Qty)
	}

	order := entity.Order{
		UserID:        userID,
		RestaurantID:  req.RestaurantID,
		Subtotal:      subtotal,
		Total:         subtotal,
		Address:       req.Address,
		OrderStatusID: 1, // Pending
	}

	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save items
	for _, it := range req.Items {
		var menu entity.Menu
		h.DB.First(&menu, it.MenuID)
		oi := entity.OrderItem{
			OrderID:   order.ID,
			MenuID:    it.MenuID,
			Qty:       it.Qty,
			UnitPrice: menu.Price,
			Total:     menu.Price * int64(it.Qty),
		}
		h.DB.Create(&oi)
	}

	c.JSON(http.StatusCreated, CreateOrderRes{ID: order.ID, Total: order.Total})
}

// GET /orders/profile
func (h *OrderController) ListForMe(c *gin.Context) {
	userVal, _ := c.Get("userId")
	userID := userVal.(uint)

	var orders []OrderSummary
	h.DB.Model(&entity.Order{}).
		Select("id, restaurant_id, total, order_status_id, created_at").
		Where("user_id = ?", userID).
		Order("id DESC").
		Scan(&orders)

	c.JSON(http.StatusOK, gin.H{"items": orders})
}

// GET /orders/:id
func (h *OrderController) Detail(c *gin.Context) {
	userVal, _ := c.Get("userId")
	userID := userVal.(uint)

	id, _ := strconv.Atoi(c.Param("id"))

	var order entity.Order
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var items []entity.OrderItem
	h.DB.Where("order_id = ?", order.ID).Find(&items)

	c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

// POST /orders/checkout-from-cart
func (h *OrderController) CheckoutFromCart(c *gin.Context) {
	userVal, _ := c.Get("userId")
	userID := userVal.(uint)

	var req CheckoutFromCartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var cart entity.Cart
	if err := h.DB.Preload("Items").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart empty"})
		return
	}

	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart empty"})
		return
	}

	var subtotal int64
	for _, it := range cart.Items {
		subtotal += it.Total
	}

	order := entity.Order{
		UserID:        userID,
		RestaurantID:  cart.RestaurantID,
		Subtotal:      subtotal,
		Total:         subtotal,
		Address:       req.Address,
		OrderStatusID: 1,
	}

	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, it := range cart.Items {
		oi := entity.OrderItem{
			OrderID:   order.ID,
			MenuID:    it.MenuID,
			Qty:       it.Qty,
			UnitPrice: it.UnitPrice,
			Total:     it.Total,
		}
		h.DB.Create(&oi)
	}

	// clear cart
	h.DB.Where("user_id = ?", userID).Delete(&entity.CartItem{})

	c.JSON(http.StatusCreated, CreateOrderRes{ID: order.ID, Total: order.Total})
}
