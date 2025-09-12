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

type OrderController struct{ DB *gorm.DB }

func NewOrderController(db *gorm.DB) *OrderController { return &OrderController{DB: db} }

// ---------------- DTO ----------------
type OrderItemIn struct {
	MenuID uint   `json:"menuId"`
	Qty    int    `json:"qty"`
	Note   string `json:"note"` // ✅ รับ note จาก FE
}

type CreateOrderReq struct {
	RestaurantID  uint          `json:"restaurantId"`
	Items         []OrderItemIn `json:"items"`
	Address       string        `json:"address"`
	PaymentMethod string        `json:"paymentMethod"`            // "PromptPay" | "Cash on Delivery"
	Discount      *int64        `json:"discount,omitempty"`       // ✅ optional
	DeliveryFee   *int64        `json:"deliveryFee,omitempty"`    // ✅ optional
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
	Address       string `json:"address"`
	PaymentMethod string `json:"paymentMethod"`
	Discount      *int64 `json:"discount,omitempty"`    // ✅ optional
	DeliveryFee   *int64 `json:"deliveryFee,omitempty"` // ✅ optional
}

// ---- PaymentSummary DTO ----
type PaymentSummary struct {
	MethodId   uint       `json:"methodId"`
	MethodName string     `json:"methodName"`
	StatusId   uint       `json:"statusId"`
	StatusName string     `json:"statusName"`
	PaidAt     *time.Time `json:"paidAt,omitempty"`
}

// ---- OrderDetail Response ----
type OrderDetailRes struct {
	ID             uint               `json:"id"`
	Subtotal       int64              `json:"subtotal"`
	Discount       int64              `json:"discount"`
	DeliveryFee    int64              `json:"deliveryFee"`
	Total          int64              `json:"total"`
	Address        string             `json:"address"`
	RestaurantID   uint               `json:"restaurantId"`
	OrderStatusID  uint               `json:"orderStatusId"`
	Items          []entity.OrderItem `json:"items"`
	PaymentSummary *PaymentSummary    `json:"paymentSummary,omitempty"`
}

// ---------------- Handlers ----------------

// POST /orders
func (h *OrderController) Create(c *gin.Context) {
	userID := c.MustGet("userId").(uint)

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items required"})
		return
	}

	// คำนวณ subtotal จากเมนูล่าสุด
	var subtotal int64
	for _, it := range req.Items {
		var menu entity.Menu
		if err := h.DB.Select("id, price").First(&menu, it.MenuID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "menu not found"})
			return
		}
		subtotal += menu.Price * int64(it.Qty)
	}
	discount := int64(0)
	if req.Discount != nil {
		discount = *req.Discount
	}
	delivery := int64(0)
	if req.DeliveryFee != nil {
		delivery = *req.DeliveryFee
	}
	total := subtotal - discount + delivery
	if total < 0 { total = 0 }

	var out CreateOrderRes
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		order := entity.Order{
			UserID:        userID,
			RestaurantID:  req.RestaurantID,
			Subtotal:      subtotal,
			Discount:      discount,   // ✅ เก็บส่วนลด
			DeliveryFee:   delivery,   // ✅ เก็บค่าส่ง
			Total:         total,
			Address:       req.Address,
			OrderStatusID: 1, // Pending
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		for _, it := range req.Items {
			var menu entity.Menu
			if err := tx.Select("id, price").First(&menu, it.MenuID).Error; err != nil {
				return err
			}
			oi := entity.OrderItem{
				OrderID:   order.ID,
				MenuID:    it.MenuID,
				Qty:       it.Qty,
				UnitPrice: menu.Price,
				Total:     menu.Price * int64(it.Qty),
				Note:      it.Note, // ✅ เก็บ note
			}
			if err := tx.Create(&oi).Error; err != nil {
				return err
			}
		}

		if req.PaymentMethod != "" {
			pmID := uint(0)
			switch req.PaymentMethod {
			case "PromptPay":
				pmID = 1
			case "Cash on Delivery":
				pmID = 2
			}
			if pmID != 0 {
				p := entity.Payment{
					Amount:          order.Total,
					OrderID:         order.ID,
					PaymentMethodID: pmID,
					PaymentStatusID: 1, // Pending
				}
				if err := tx.Create(&p).Error; err != nil {
					return err
				}
			}
		}

		out = CreateOrderRes{ID: order.ID, Total: order.Total}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, out)
}

// GET /orders/profile
func (h *OrderController) ListForMe(c *gin.Context) {
	userID := c.MustGet("userId").(uint)

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
	userID := c.MustGet("userId").(uint)
	id, _ := strconv.Atoi(c.Param("id"))

	var order entity.Order
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var items []entity.OrderItem
	// ดึง fields ที่ FE ใช้ + note
	h.DB.Model(&entity.OrderItem{}).
		Select("id, qty, unit_price, total, menu_id, order_id, note").
		Where("order_id = ?", order.ID).Find(&items)

	// payment ล่าสุด (อาจไม่มี)
	var payment entity.Payment
	err := h.DB.Preload("PaymentMethod").
		Preload("PaymentStatus").
		Where("order_id = ?", order.ID).
		Order("id DESC").
		First(&payment).Error

	var paySummary *PaymentSummary
	if err == nil {
		paySummary = &PaymentSummary{
			MethodId:   payment.PaymentMethodID,
			MethodName: payment.PaymentMethod.MethodName,
			StatusId:   payment.PaymentStatusID,
			StatusName: payment.PaymentStatus.StatusName,
			PaidAt:     payment.PaidAt,
		}
	}

	res := OrderDetailRes{
		ID:             order.ID,
		Subtotal:       order.Subtotal,
		Discount:       order.Discount,
		DeliveryFee:    order.DeliveryFee,
		Total:          order.Total,
		Address:        order.Address,
		RestaurantID:   order.RestaurantID,
		OrderStatusID:  order.OrderStatusID,
		Items:          items,
		PaymentSummary: paySummary,
	}
	c.JSON(http.StatusOK, res)
}

// POST /orders/checkout-from-cart
func (h *OrderController) CheckoutFromCart(c *gin.Context) {
	userID := c.MustGet("userId").(uint)

	var req CheckoutFromCartReq
	if err := c.ShouldBindJSON(&req); err != nil {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var cart entity.Cart
	// อ้างอิงคอลัมน์ carts.user_id ตรง (schema ของคุณใช้ชื่อ user_id อยู่แล้ว)
	if err := h.DB.Preload("Items").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart empty"})
		return
	}
	if len(cart.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart empty"})
		return
	}

	// คำนวณราคาจาก snapshot ใน cart
	var subtotal int64
	for _, it := range cart.Items {
		subtotal += it.Total
	}
	discount := int64(0)
	if req.Discount != nil {
		discount = *req.Discount
	}
	delivery := int64(0)
	if req.DeliveryFee != nil {
		delivery = *req.DeliveryFee
	}
	total := subtotal - discount + delivery
	if total < 0 { total = 0 }

	var out CreateOrderRes
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		order := entity.Order{
			UserID:        userID,
			RestaurantID:  cart.RestaurantID,
			Subtotal:      subtotal,
			Discount:      discount,  // ✅
			DeliveryFee:   delivery,  // ✅
			Total:         total,
			Address:       req.Address,
			OrderStatusID: 1,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		// copy รายการจาก cart → order (รวม note)
		for _, it := range cart.Items {
			oi := entity.OrderItem{
				OrderID:   order.ID,
				MenuID:    it.MenuID,
				Qty:       it.Qty,
				UnitPrice: it.UnitPrice,
				Total:     it.Total,
				Note:      it.Note, // ✅ copy note จาก cart item
			}
			if err := tx.Create(&oi).Error; err != nil {
				return err
			}
		}

		// payment pending ถ้ามีวิธีจ่าย
		if req.PaymentMethod != "" {
			pmID := uint(0)
			switch req.PaymentMethod {
			case "PromptPay":
				pmID = 1
			case "Cash on Delivery":
				pmID = 2
			}
			if pmID != 0 {
				p := entity.Payment{
					Amount:          order.Total,
					OrderID:         order.ID,
					PaymentMethodID: pmID,
					PaymentStatusID: 1, // Pending
				}
				if err := tx.Create(&p).Error; err != nil {
					return err
				}
			}
		}

		// เคลียร์ cart items ของ user นี้ (subquery หา cart_id ที่เป็นของ user)
		if err := tx.Where("cart_id IN (?)",
			tx.Table("carts").Select("id").Where("user_id = ?", userID),
		).Delete(&entity.CartItem{}).Error; err != nil {
			return err
		}

		out = CreateOrderRes{ID: order.ID, Total: order.Total}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, out)
}
