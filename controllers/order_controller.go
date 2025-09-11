package controllers

import (
	"errors"
	"time"
	"strconv"

	"backend/entity"
	"backend/pkg/resp"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct{ DB *gorm.DB }
func NewOrderController(db *gorm.DB) *OrderController { return &OrderController{DB: db} }

// ===== Create Order =====

type OrderItemSelectionIn struct {
	OptionID      uint `json:"optionId"`
	OptionValueID uint `json:"optionValueId"`
}
type OrderItemIn struct {
	MenuID uint `json:"menuId" binding:"required"`
	Qty    int  `json:"qty" binding:"required,min=1"`
	Selections []OrderItemSelectionIn `json:"selections"`
}
type CreateOrderReq struct {
	RestaurantID uint          `json:"restaurantId" binding:"required"`
	Items        []OrderItemIn `json:"items" binding:"required,min=1"`
	// (MVP) ยังไม่รองรับโปร/จ่ายเงินตอนนี้
}

func (oc *OrderController) Create(c *gin.Context) {
	uid := utils.CurrentUserID(c)

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil { resp.BadRequest(c, err.Error()); return }

	// ตรวจร้าน
	var r entity.Restaurant
	if err := oc.DB.Select("id").First(&r, req.RestaurantID).Error; err != nil {
		resp.BadRequest(c, "restaurant not found"); return
	}

	// เตรียมคำนวณราคา
	subtotal := int64(0)
	type itmp struct {
		menu *entity.Menu
		qty int
		sels []OrderItemSelectionIn
		unitPrice int64
	}
	tmp := make([]itmp, 0, len(req.Items))

	for _, it := range req.Items {
		var m entity.Menu
		if err := oc.DB.Select("id, price, restaurant_id").First(&m, it.MenuID).Error; err != nil {
			resp.BadRequest(c, "menu not found"); return
		}
		if m.RestaurantID != req.RestaurantID {
			resp.BadRequest(c, "menu not in this restaurant"); return
		}
		unit := m.Price

		// บวก price adjustment ตาม option value
		for _, s := range it.Selections {
			var ov entity.OptionValue
			if err := oc.DB.Select("id, price_adjustment, option_id").First(&ov, s.OptionValueID).Error; err != nil {
				resp.BadRequest(c, "option value not found"); return
			}
			unit += ov.PriceAdjustment
		}

		subtotal += unit * int64(it.Qty)
		tmp = append(tmp, itmp{menu: &m, qty: it.Qty, sels: it.Selections, unitPrice: unit})
	}

	discount := int64(0)     // MVP: 0
	deliveryFee := int64(20) // MVP: 20 บาทคงที่
	total := subtotal - discount + deliveryFee

	// สร้างออเดอร์ + รายการ (transaction)
	tx := oc.DB.Begin()
	order := entity.Order{
		Subtotal: subtotal, Discount: discount, DeliveryFee: deliveryFee, Total: total,
		UserID: uid, RestaurantID: req.RestaurantID, OrderStatusID: 1, // 1 = Pending
	}
	if err := tx.Create(&order).Error; err != nil { tx.Rollback(); resp.ServerError(c, err); return }

	for _, t := range tmp {
		oi := entity.OrderItem{
			Qty: t.qty, UnitPrice: t.unitPrice, Total: t.unitPrice * int64(t.qty),
			OrderID: order.ID, MenuID: t.menu.ID,
		}
		if err := tx.Create(&oi).Error; err != nil { tx.Rollback(); resp.ServerError(c, err); return }
		for _, s := range t.sels {
			ois := entity.OrderItemSelection{
				OrderItemID: oi.ID, OptionID: s.OptionID, OptionValueID: s.OptionValueID, PriceDelta: 0, // เรารวมไว้ใน unitPrice แล้ว
			}
			if err := tx.Create(&ois).Error; err != nil { tx.Rollback(); resp.ServerError(c, err); return }
		}
	}

	if err := tx.Commit().Error; err != nil { resp.ServerError(c, err); return }
	resp.Created(c, gin.H{"id": order.ID, "total": order.Total})
}

// ===== My Orders =====

// GET /profile/order
func (oc *OrderController) ListForMe(c *gin.Context) {
	uid := utils.CurrentUserID(c)
	type row struct {
		ID uint `json:"id"`
		RestaurantID uint `json:"restaurantId"`
		Total int64 `json:"total"`
		OrderStatusID uint `json:"orderStatusId"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var items []row
	if err := oc.DB.Model(&entity.Order{}).
		Select("id, restaurant_id, total, order_status_id, created_at").
		Where("user_id = ?", uid).
		Order("id DESC").Limit(50).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items})
}

// GET /orders/:id (เฉพาะเจ้าของออเดอร์)
func (oc *OrderController) Detail(c *gin.Context) {
	uid := utils.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var o entity.Order
	if err := oc.DB.Where("id = ? AND user_id = ?", id, uid).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { c.JSON(404, gin.H{"ok": false, "error": "not found"}); return }
		resp.ServerError(c, err); return
	}

	// โหลด items แบบพอประมาณ
	var items []entity.OrderItem
	if err := oc.DB.Model(&entity.OrderItem{}).
		Select("id, qty, unit_price, total, menu_id, order_id").
		Where("order_id = ?", o.ID).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}

	resp.OK(c, gin.H{
		"id": o.ID, "subtotal": o.Subtotal, "discount": o.Discount, "deliveryFee": o.DeliveryFee, "total": o.Total,
		"orderStatusId": o.OrderStatusID, "restaurantId": o.RestaurantID, "items": items,
	})
}
