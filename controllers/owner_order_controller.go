// controllers/owner_order_controller.go
package controllers

import (
	"backend/entity"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OwnerOrderController struct {
	DB *gorm.DB
}

func NewOwnerOrderController(db *gorm.DB) *OwnerOrderController {
	return &OwnerOrderController{DB: db}
}

// ---------------- DTO ----------------
type OwnerOrderListOut struct {
	Items []OwnerOrderSummary `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}
type OwnerOrderSummary struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"userId"`
	CustomerName  string    `json:"customerName"`
	Total         int64     `json:"total"`
	OrderStatusID uint      `json:"orderStatusId"`
	CreatedAt     time.Time `json:"createdAt"`
}
type OwnerOrderDetail struct {
	Order entity.Order       `json:"order"`
	Items []entity.OrderItem `json:"items"`
}

// ---------------- Handlers ----------------

// GET /owner/restaurants/:id/orders
func (ctl *OwnerOrderController) List(c *gin.Context) {
	userID := c.GetUint("userId")
	restID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// ✅ ตรวจสิทธิ์ร้าน
	var count int64
	if err := ctl.DB.Model(&entity.Restaurant{}).
		Where("id = ? AND user_id = ?", restID, userID).
		Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// filter
	var statusID *uint
	if s := c.Query("statusId"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			tmp := uint(v)
			statusID = &tmp
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	// count
	var total int64
	qCount := ctl.DB.Model(&entity.Order{}).Where("restaurant_id = ?", restID)
	if statusID != nil {
		qCount = qCount.Where("order_status_id = ?", *statusID)
	}
	if err := qCount.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// join users
	var rows []struct {
		ID, UserID, OrderStatusID uint
		Total                     int64
		CreatedAt                 time.Time
		FirstName, LastName       string
	}
	q := ctl.DB.Table("orders AS o").
		Select("o.id, o.user_id, o.total, o.order_status_id, o.created_at, u.first_name, u.last_name").
		Joins("JOIN users u ON u.id = o.user_id").
		Where("o.restaurant_id = ?", restID)
	if statusID != nil {
		q = q.Where("o.order_status_id = ?", *statusID)
	}
	if err := q.Order("o.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// map result
	items := make([]OwnerOrderSummary, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		items = append(items, OwnerOrderSummary{
			ID:            r.ID,
			UserID:        r.UserID,
			CustomerName:  name,
			Total:         r.Total,
			OrderStatusID: r.OrderStatusID,
			CreatedAt:     r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, &OwnerOrderListOut{Items: items, Total: total, Page: page, Limit: limit})
}

// GET /owner/restaurants/:id/orders/:orderId
func (ctl *OwnerOrderController) Detail(c *gin.Context) {
	userID := c.GetUint("userId")
	restID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	orderID, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)

	// ✅ ตรวจสิทธิ์ร้าน
	var count int64
	if err := ctl.DB.Model(&entity.Restaurant{}).
		Where("id = ? AND user_id = ?", restID, userID).
		Count(&count).Error; err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var order entity.Order
	if err := ctl.DB.Where("id = ? AND restaurant_id = ?", orderID, restID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	var items []entity.OrderItem
	ctl.DB.Where("order_id = ?", order.ID).Find(&items)

	c.JSON(http.StatusOK, &OwnerOrderDetail{Order: order, Items: items})
}

// ---------------- Actions (เปลี่ยนสถานะ) ----------------
func (ctl *OwnerOrderController) Accept(c *gin.Context)   { ctl.updateStatus(c, "Pending", "Preparing") }
func (ctl *OwnerOrderController) Handoff(c *gin.Context)  { ctl.updateStatus(c, "Preparing", "Delivering") }
func (ctl *OwnerOrderController) Complete(c *gin.Context) { ctl.updateStatus(c, "Delivering", "Completed") }
func (ctl *OwnerOrderController) Cancel(c *gin.Context)   { ctl.updateStatus(c, "Pending", "Cancelled") }

// ---------------- Helper ----------------
func (ctl *OwnerOrderController) updateStatus(c *gin.Context, fromName, toName string) {
	userID := c.GetUint("userId")
	orderID, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)

	// หา status id
	var fromID, toID uint
	if err := ctl.DB.Model(&entity.OrderStatus{}).
		Select("id").Where("status_name = ?", fromName).First(&fromID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "status not found"})
		return
	}
	if err := ctl.DB.Model(&entity.OrderStatus{}).
		Select("id").Where("status_name = ?", toName).First(&toID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "status not found"})
		return
	}

	// ✅ guard update
	tx := ctl.DB.Model(&entity.Order{}).
		Where("id = ? AND order_status_id = ?", orderID, fromID).
		Update("order_status_id", toID)
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}
	if tx.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "invalid state or already updated"})
		return
	}

	// ตรวจว่า order belong กับร้านนี้
	var rest entity.Restaurant
	if err := ctl.DB.Table("restaurants r").
		Joins("JOIN orders o ON o.restaurant_id = r.id").
		Where("o.id = ? AND r.user_id = ?", orderID, userID).
		First(&rest).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.Status(http.StatusNoContent)
}
