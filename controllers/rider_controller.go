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

/* =========================
   WORK HISTORIES (รวม service เข้ามา)
   GET /api/rider/works
   พารามิเตอร์: page, pageSize, orderId, status, dateFrom, dateTo (RFC3339)
   ส่งคืน: items, total, summary{totalTrips,totalFare}
========================= */

// GET /rider/works
func (h *RiderController) ListWorks(c *gin.Context) {
	uid := c.GetUint("userId")
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// หา rider จาก user_id
	var rider entity.Rider
	if err := h.DB.Where("user_id=?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	// query params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 { page = 1 }
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize <= 0 || pageSize > 100 { pageSize = 10 }

	orderQ := strings.TrimSpace(c.Query("orderId"))
	statusQ := strings.TrimSpace(c.Query("status"))
	var dateFrom *time.Time
	var dateTo *time.Time
	if s := c.Query("dateFrom"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil { dateFrom = &t }
	}
	if s := c.Query("dateTo"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil { dateTo = &t }
	}

	// base query — เลือกเฉพาะฟิลด์ที่มีจริง
	base := h.DB.Table("rider_works AS rw").
		Joins("JOIN orders AS o ON o.id = rw.order_id").
		Joins("JOIN order_statuses AS os ON os.id = o.order_status_id").
		Where("rw.rider_id = ?", rider.ID)

	// filters
	if orderQ != "" {
		base = base.Where("CAST(o.id AS CHAR) LIKE ?", "%"+orderQ+"%")
	}
	if statusQ != "" {
		// ให้เทียบกับชื่อ status ตรง ๆ เช่น Pending/Preparing/Delivering/Completed/Cancelled
		base = base.Where("os.status_name = ?", statusQ)
	}
	if dateFrom != nil {
		base = base.Where("rw.work_at >= ?", dateFrom.UTC())
	}
	if dateTo != nil {
		base = base.Where("rw.work_at <= ?", dateTo.UTC())
	}

	// count
	var total int64
	if err := base.Select("rw.id").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// row shape ที่จะ scan
	type row struct {
		ID               uint       `json:"id"`                // rider_works.id
		WorkAt           *time.Time `json:"workAt"`
		FinishAt         *time.Time `json:"finishAt"`
		OrderID          uint       `json:"orderId"`
		Address          string     `json:"address"`
		Subtotal         int64      `json:"subtotal"`
		Discount         int64      `json:"discount"`
		DeliveryFee      int64      `json:"deliveryFee"`
		Total            int64      `json:"total"`
		OrderStatusName  string     `json:"orderStatusName"`
	}

	var rows []row
	if err := base.
		Select(`
			rw.id AS id,
			rw.work_at, rw.finish_at,
			o.id AS order_id,
			o.address,
			o.subtotal, o.discount, o.delivery_fee, o.total,
			os.status_name AS order_status_name
		`).
		Order("rw.work_at DESC, rw.id DESC").
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// map → items ที่ FE ใช้
	type item struct {
		ID              uint       `json:"id"`
		OrderID         uint       `json:"orderId"`
		Address         string     `json:"address"`
		Subtotal        int64      `json:"subtotal"`
		Discount        int64      `json:"discount"`
		DeliveryFee     int64      `json:"deliveryFee"`
		Total           int64      `json:"total"`
		WorkAt          *time.Time `json:"workAt,omitempty"`
		FinishAt        *time.Time `json:"finishAt,omitempty"`
		OrderStatusName string     `json:"orderStatusName,omitempty"`
	}

	items := make([]item, 0, len(rows))
	var sumTotal int64 = 0
	for _, r := range rows {
		items = append(items, item{
			ID:              r.ID,
			OrderID:         r.OrderID,
			Address:         r.Address,
			Subtotal:        r.Subtotal,
			Discount:        r.Discount,
			DeliveryFee:     r.DeliveryFee,
			Total:           r.Total,
			WorkAt:          r.WorkAt,
			FinishAt:        r.FinishAt,
			OrderStatusName: r.OrderStatusName,
		})
		sumTotal += r.Total
	}

	c.JSON(http.StatusOK, gin.H{
		"items":   items,
		"total":   total,
		"summary": gin.H{"totalTrips": total, "totalFare": sumTotal},
	})
}

func fallback(s, alt string) string {
	if strings.TrimSpace(s) == "" { return alt }
	return s
}
func derefOr(t *time.Time, zero time.Time) time.Time {
	if t != nil { return *t }
	return zero
}

// map order_status_id → FE code
func mapOrderStatus(id uint) string {
	switch id {
	case getOrderStatusID(nil, "Pending"):
		return "PENDING"
	case getOrderStatusID(nil, "Preparing"):
		return "PICKED_UP" // หรือสถานะที่ตรงที่สุดในระบบพี่
	case getOrderStatusID(nil, "Delivering"):
		return "PICKED_UP"
	case getOrderStatusID(nil, "Completed"):
		return "DELIVERED"
	case getOrderStatusID(nil, "Cancelled"):
		return "CANCELLED"
	default:
		return "PENDING"
	}
}

// map payment method code/name → FE code
func normalizePM(code *string) string {
	if code == nil { return "CASH" }
	v := strings.ToUpper(strings.TrimSpace(*code))
	switch v {
	case "CASH", "COD", "CASH_ON_DELIVERY":
		return "CASH"
	case "WALLET", "EWALLET", "WALLETS":
		return "WALLET"
	case "QR", "PROMPTPAY", "QR_PROMPTPAY":
		return "QR"
	default:
		return "CASH"
	}
}

/* =========================
   ONLINE / OFFLINE (มีอยู่แล้ว)
========================= */

func (h *RiderController) SetAvailability(c *gin.Context) {
	uid := c.GetUint("userId")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
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

/* =========================
   ACCEPT (มีอยู่แล้ว)
========================= */

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
		res := tx.Model(&entity.Order{}).
			Where("id=? AND order_status_id=?", oid, preparingID).
			Update("order_status_id", deliveringID)
		if res.Error != nil { return res.Error }
		if res.RowsAffected == 0 { return fmt.Errorf("order not in preparing state") }
		return nil
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

/* =========================
   COMPLETE (มีอยู่แล้ว)
========================= */

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

	if rider.RiderStatusID != assignedID {
		c.JSON(http.StatusConflict, gin.H{"error": "not assigned"})
		return
	}
	if order.OrderStatusID != deliveringID {
		c.JSON(http.StatusConflict, gin.H{"error": "order not delivering"})
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&entity.RiderWork{}).
			Where("rider_id=? AND order_id=? AND finish_at IS NULL", rider.ID, order.ID).
			Update("finish_at", &now).Error; err != nil {
			return err
		}
		if err := tx.Model(&entity.Order{}).
			Where("id=?", order.ID).
			Update("order_status_id", completedID).Error; err != nil {
			return err
		}

		// COD → mark paid
		var payment entity.Payment
		if err := tx.Where("order_id=?", order.ID).First(&payment).Error; err == nil {
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

type RiderProfileResponse struct {
	ID          uint   `json:"id"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Address     string `json:"address"`
	Avatar      string `json:"avatarBase64,omitempty"`

	VehiclePlate string `json:"vehiclePlate"`
	License      string `json:"license"`
	DriveCard    string `json:"driveCard"`

	Status string `json:"status"`
}

func (h *RiderController) GetProfile(c *gin.Context) {
	uid := c.GetUint("userId")

	var rider entity.Rider
	if err := h.DB.
		Preload("User").
		Preload("RiderStatus").
		Where("user_id = ?", uid).
		First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	resp := gin.H{
		"userId":       rider.User.ID,
		"firstName":    rider.User.FirstName,
		"lastName":     rider.User.LastName,
		"phoneNumber":  rider.User.PhoneNumber,
		"avatarBase64": rider.User.AvatarBase64,

		"riderId":      rider.ID,
		"nationalId":   rider.NationalID,
		"vehiclePlate": rider.VehiclePlate,
		"zone":         rider.Zone,
		"license":      rider.License,
		"driveCard":    rider.DriveCard, // base64 (ไม่มี header)
		"status":       rider.RiderStatus.StatusName,
	}

	c.JSON(http.StatusOK, resp)
}

// PUT /rider/me
func (h *RiderController) UpdateMe(c *gin.Context) {
	uid := c.GetUint("userId")

	// หา rider ของ user
	var rider entity.Rider
	if err := h.DB.Where("user_id = ?", uid).First(&rider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rider not found"})
		return
	}

	// รับ request
	var req struct {
		NationalID      string  `json:"nationalId"`
		VehiclePlate    string  `json:"vehiclePlate"`
		Zone            string  `json:"zone"`
		License         string  `json:"license"`
		DriveCardBase64 *string `json:"driveCardBase64"` // undefined = ไม่แตะ, "" = ลบ, dataURL = อัปเดต
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// เตรียมค่าอัปเดต
	updates := map[string]interface{}{
		"national_id":   strings.TrimSpace(req.NationalID),
		"vehicle_plate": strings.TrimSpace(req.VehiclePlate),
		"zone":          strings.TrimSpace(req.Zone),
		"license":       strings.TrimSpace(req.License),
	}

	// การอัปเดต DriveCard
	if req.DriveCardBase64 != nil {
		val := *req.DriveCardBase64
		if strings.HasPrefix(val, "data:") {
			// ถ้า FE ส่งมาเป็น dataURL → ตัด header ออก
			parts := strings.SplitN(val, ",", 2)
			if len(parts) == 2 {
				updates["drive_card"] = parts[1]
			}
		} else {
			// อาจเป็น "" (ลบ) หรือ base64 ตรง ๆ
			updates["drive_card"] = val
		}
	}

	// บันทึก
	if err := h.DB.Model(&rider).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}


// ---------- HELPER ----------
func getRiderStatusID(db *gorm.DB, name string) uint {
	var rs entity.RiderStatus
	db.Where("status_name=?", name).First(&rs)
	return rs.ID
}
func getOrderStatusID(db *gorm.DB, name string) uint {
	var os entity.OrderStatus
	if db != nil {
		db.Where("status_name=?", name).First(&os)
	}
	// หมายเหตุ: mapOrderStatus เรียกแบบ db=nil เพื่อเทียบค่าแบบ lazy; ถ้าอยากชัวร์ให้ hardcode mapping
	return os.ID
}
