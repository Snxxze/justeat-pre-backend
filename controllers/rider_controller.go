package controllers

import (
	"backend/entity"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RiderController struct {
	DB *gorm.DB
}

func NewRiderController(db *gorm.DB) *RiderController { return &RiderController{DB: db} }

// ---------- ONLINE / OFFLINE ----------
func (h *RiderController) SetAvailability(c *gin.Context) {
	uid := c.GetUint("userId")
	var req struct{ Status string `json:"status" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var rider entity.Rider
	if err := h.DB.Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	var statusID uint
	if status == "ONLINE" {
		statusID = getRiderStatusID(h.DB, "ONLINE")
	} else if status == "OFFLINE" {
		// ห้าม offline ถ้ามีงานค้าง
		var cnt int64
		h.DB.Model(&entity.RiderWork{}).
			Where("rider_id=? AND finish_at IS NULL", rider.ID).
			Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot go offline with active work"})
			return
		}
		statusID = getRiderStatusID(h.DB, "OFFLINE")
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	h.DB.Model(&entity.Rider{}).Where("id=?", rider.ID).
		Update("rider_status_id", statusID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- ACCEPT ----------
func (h *RiderController) Accept(c *gin.Context) {
	uid := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)

	var rider entity.Rider
	if err := h.DB.Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	onlineID := getRiderStatusID(h.DB, "ONLINE")
	assignedID := getRiderStatusID(h.DB, "ASSIGNED")
	preparingID := getOrderStatusID(h.DB, "Preparing")
	deliveringID := getOrderStatusID(h.DB, "Delivering")

	if rider.RiderStatusID != onlineID {
		c.JSON(http.StatusConflict, gin.H{"error": "rider not online"})
		return
	}

	// transaction
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Create(&entity.RiderWork{
			RiderID: rider.ID, OrderID: uint(oid), WorkAt: &now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&entity.Rider{}).
			Where("id=?", rider.ID).
			Update("rider_status_id", assignedID).Error; err != nil {
			return err
		}
		// order Preparing → Delivering
		res := tx.Model(&entity.Order{}).
			Where("id=? AND order_status_id=?", oid, preparingID).
			Update("order_status_id", deliveringID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("order not in preparing state")
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- COMPLETE ----------
func (h *RiderController) Complete(c *gin.Context) {
	uid := c.GetUint("userId")
	oid, _ := strconv.ParseUint(c.Param("orderId"), 10, 64)

	var rider entity.Rider
	if err := h.DB.Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	var order entity.Order
	if err := h.DB.First(&order, oid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	assignedID := getRiderStatusID(h.DB, "ASSIGNED")
	deliveringID := getOrderStatusID(h.DB, "Delivering")
	completedID := getOrderStatusID(h.DB, "Completed")
	onlineID := getRiderStatusID(h.DB, "ONLINE")

	// guard
	if rider.RiderStatusID != assignedID {
		c.JSON(http.StatusConflict, gin.H{"error": "not assigned"})
		return
	}
	if order.OrderStatusID != deliveringID {
		c.JSON(http.StatusConflict, gin.H{"error": "order not delivering"})
		return
	}

	// transaction
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// ปิด RiderWork
		if err := tx.Model(&entity.RiderWork{}).
			Where("rider_id=? AND order_id=? AND finish_at IS NULL", rider.ID, order.ID).
			Update("finish_at", &now).Error; err != nil {
			return err
		}

		// อัปเดต Order → Completed
		if err := tx.Model(&entity.Order{}).
			Where("id=?", order.ID).
			Update("order_status_id", completedID).Error; err != nil {
			return err
		}

		// กรณีเป็น COD → อัปเดต payment เป็น "Paid"
		var payment entity.Payment
		if err := tx.Where("order_id=?", order.ID).First(&payment).Error; err == nil {
			// หา payment method
			var method entity.PaymentMethod
			if err := tx.First(&method, payment.PaymentMethodID).Error; err == nil {
				if strings.EqualFold(method.MethodName, "Cash on Delivery") {
					paidID := getPaymentStatusID(tx, "Paid")
					if err := tx.Model(&entity.Payment{}).
						Where("id=?", payment.ID).
						Update("payment_status_id", paidID).Error; err != nil {
						return err
					}
				}
			}
		}

		// Rider → ONLINE
		return tx.Model(&entity.Rider{}).
			Where("id=?", rider.ID).
			Update("rider_status_id", onlineID).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- HELPER ----------
func getPaymentStatusID(db *gorm.DB, name string) uint {
	var ps entity.PaymentStatus
	db.Where("status_name=?", name).First(&ps)
	return ps.ID
}


// ---------- LIST AVAILABLE ----------
func (h *RiderController) ListAvailable(c *gin.Context) {
	preparingID := getOrderStatusID(h.DB, "Preparing")
	var rows []struct {
		ID             uint      `json:"id"`
		CreatedAt      time.Time `json:"createdAt"`
		RestaurantName string    `json:"restaurantName"`
		CustomerName   string    `json:"customerName"`
		Address        string    `json:"address"`
		Total          int64     `json:"total"`
	}

	err := h.DB.
		Table("orders AS o").
		Select(`o.id, o.created_at, r.name AS restaurant_name,
		        CONCAT(u.first_name,' ',u.last_name) AS customer_name,
		        o.address, o.total`).
		Joins("JOIN users u ON u.id=o.user_id").
		Joins("JOIN restaurants r ON r.id=o.restaurant_id").
		Joins("LEFT JOIN rider_works rw ON rw.order_id=o.id AND rw.finish_at IS NULL").
		Where("o.order_status_id=? AND rw.id IS NULL", preparingID).
		Order("o.id DESC").
		Scan(&rows).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// ---------- GET STATUS ----------
func (h *RiderController) GetStatus(c *gin.Context) {
	uid := c.GetUint("userId")
	var rider entity.Rider
	if err := h.DB.Preload("RiderStatus").
		Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    rider.RiderStatus.StatusName,
		"isWorking": rider.RiderStatus.StatusName != "OFFLINE",
	})
}

// ---------- GET CURRENT WORK ----------
func (h *RiderController) GetCurrentWork(c *gin.Context) {
	uid := c.GetUint("userId")
	var rider entity.Rider
	if err := h.DB.Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	var row struct {
		ID             uint      `json:"id"`
		CreatedAt      time.Time `json:"createdAt"`
		RestaurantName string    `json:"restaurantName"`
		CustomerName   string    `json:"customerName"`
		Address        string    `json:"address"`
		Total          int64     `json:"total"`
	}
	h.DB.Table("rider_works rw").
		Select(`o.id, o.created_at, r.name AS restaurant_name,
		        CONCAT(u.first_name,' ',u.last_name) AS customer_name,
		        o.address, o.total`).
		Joins("JOIN orders o ON o.id=rw.order_id").
		Joins("JOIN users u ON u.id=o.user_id").
		Joins("JOIN restaurants r ON r.id=o.restaurant_id").
		Where("rw.rider_id=? AND rw.finish_at IS NULL", rider.ID).
		Limit(1).Scan(&row)

	if row.ID == 0 {
		c.JSON(http.StatusOK, gin.H{"work": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work": row})
}

// ---------- HELPER ----------
func getRiderStatusID(db *gorm.DB, name string) uint {
	var rs entity.RiderStatus
	db.Where("status_name=?", name).First(&rs)
	return rs.ID
}
func getOrderStatusID(db *gorm.DB, name string) uint {
	var os entity.OrderStatus
	db.Where("status_name=?", name).First(&os)
	return os.ID
}
