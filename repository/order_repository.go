package repository

import (
	"backend/entity"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OrderRepository struct {
	DB *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

// ---------------- Orders (CRUD หลัก) ----------------

// POST /orders → สร้าง order
func (r *OrderRepository) CreateOrder(tx *gorm.DB, o *entity.Order) error {
	return tx.Create(o).Error
}

// GET /orders/:id (ใช้ทั่วไป เช่น admin/owner)
func (r *OrderRepository) GetOrder(orderID uint) (*entity.Order, error) {
	var o entity.Order
	if err := r.DB.First(&o, orderID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GET /orders (ลูกค้า) → รายการ order ของ user
// ดึงข้อมูลตามนี้ แล้วส่งไป
type OrderSummary struct {
	ID            uint      `json:"id"`
	RestaurantID  uint      `json:"restaurantId"`
	Total         int64     `json:"total"`
	OrderStatusID uint      `json:"orderStatusId"`
	CreatedAt     time.Time `json:"createdAt"`
}
func (r *OrderRepository) ListOrdersForUser(userID uint, limit int) ([]OrderSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []OrderSummary
	err := r.DB.Model(&entity.Order{}).
		Select("id, restaurant_id, total, order_status_id, created_at").
		Where("user_id = ?", userID).
		Order("id DESC").Limit(limit).
		Scan(&out).Error
	return out, err
}

// GET /orders/:id (ลูกค้า) → รายละเอียด order
func (r *OrderRepository) GetOrderForUser(userID, orderID uint) (*entity.Order, error) {
	var o entity.Order
	if err := r.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GET /owner/restaurants/:id/orders → รายการ order ของร้าน
// ดึงข้อมูลตามนี้ แล้วส่งไป
type OwnerOrderSummary struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"userId"`
	CustomerName  string    `json:"customerName"`
	Total         int64     `json:"total"`
	OrderStatusID uint      `json:"orderStatusId"`
	CreatedAt     time.Time `json:"createdAt"`
}
func (r *OrderRepository) ListOrdersForRestaurant(restID uint, statusID *uint, page, limit int) ([]OwnerOrderSummary, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	// count orders
	var total int64
	dbCount := r.DB.Table("orders AS o").Where("o.restaurant_id = ?", restID)
	if statusID != nil && *statusID != 0 {
		dbCount = dbCount.Where("o.order_status_id = ?", *statusID)
	}
	if err := dbCount.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// join users → ดึงชื่อลูกค้า
	var rows []struct {
		ID            uint
		UserID        uint
		Total         int64
		OrderStatusID uint
		CreatedAt     time.Time
		FirstName     string
		LastName      string
	}
	db := r.DB.Table("orders AS o").
		Select("o.id, o.user_id, o.total, o.order_status_id, o.created_at, u.first_name, u.last_name").
		Joins("JOIN users u ON u.id = o.user_id").
		Where("o.restaurant_id = ?", restID)
	if statusID != nil && *statusID != 0 {
		db = db.Where("o.order_status_id = ?", *statusID)
	}
	if err := db.Order("o.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]OwnerOrderSummary, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		out = append(out, OwnerOrderSummary{
			ID:            r.ID,
			UserID:        r.UserID,
			CustomerName:  name,
			Total:         r.Total,
			OrderStatusID: r.OrderStatusID,
			CreatedAt:     r.CreatedAt,
		})
	}
	return out, total, nil
}

// GET /owner/restaurants/:id/orders/:oid → รายละเอียด order ของร้าน
func (r *OrderRepository) GetOrderForRestaurant(restID, orderID uint) (*entity.Order, error) {
	var o entity.Order
	if err := r.DB.Where("id = ? AND restaurant_id = ?", orderID, restID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// PUT /orders/:id/status → อัปเดตสถานะ (มี guard)
func (r *OrderRepository) UpdateStatusGuard(tx *gorm.DB, orderID, fromID, toID uint) (int64, error) {
	res := tx.Model(&entity.Order{}).
		Where("id = ? AND order_status_id = ?", orderID, fromID).
		Update("order_status_id", toID)
	return res.RowsAffected, res.Error
}
func (r *OrderRepository) UpdateStatusFromTo(tx *gorm.DB, orderID, fromID, toID uint) (bool, error) {
	res := tx.Model(&entity.Order{}).
		Where("id = ? AND order_status_id = ?", orderID, fromID).
		Update("order_status_id", toID)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ---------------- Order Items ----------------
func (r *OrderRepository) CreateOrderItem(tx *gorm.DB, oi *entity.OrderItem) error {
	return tx.Create(oi).Error
}
func (r *OrderRepository) GetOrderItems(orderID uint) ([]entity.OrderItem, error) {
	var items []entity.OrderItem
	err := r.DB.Model(&entity.OrderItem{}).
		Select("id, qty, unit_price, total, menu_id, order_id").
		Where("order_id = ?", orderID).
		Find(&items).Error
	return items, err
}

// ---------------- Payments ----------------
func (r *OrderRepository) CreatePayment(tx *gorm.DB, p *entity.Payment) error {
	return tx.Create(p).Error
}
func (r *OrderRepository) GetPaymentMethodIDFromKey(key string) (uint, error) {
	if key == "" {
		return 0, nil
	}
	k := strings.ToLower(strings.TrimSpace(key))
	var methodName string
	switch k {
	case "promptpay":
		methodName = "PromptPay"
	case "cod", "cash_on_delivery", "cash-on-delivery", "cash on delivery":
		methodName = "Cash on Delivery"
	default:
		methodName = key
	}
	var row struct{ ID uint }
	if err := r.DB.Model(&entity.PaymentMethod{}).
		Select("id").Where("method_name = ?", methodName).
		Limit(1).Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// ---------------- Validations / Helpers ----------------

// เช็คร้านว่ามีอยู่จริงมั้ย
func (r *OrderRepository) RestaurantExists(id uint) (bool, error) {
	var cnt int64
	if err := r.DB.Model(&entity.Restaurant{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// เอา menu พื้นฐาน (id, price, restaurant_id)
func (r *OrderRepository) GetMenuBasics(id uint) (entity.Menu, error) {
	var m entity.Menu
	err := r.DB.Select("id, price, restaurant_id").First(&m, id).Error
	return m, err
}

// ตรวจว่าเมนูทั้งหมด belong กับร้านเดียว
func (r *OrderRepository) ValidateMenusBelongToRestaurant(menuIDs []uint, restID uint) (bool, error) {
	if len(menuIDs) == 0 {
		return true, nil
	}
	var cnt int64
	if err := r.DB.Model(&entity.Menu{}).
		Where("id IN ? AND restaurant_id = ?", menuIDs, restID).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt == int64(len(menuIDs)), nil
}

// หาค่า status id จากชื่อ
func (r *OrderRepository) GetStatusIDByName(name string) (uint, error) {
	var row struct{ ID uint }
	err := r.DB.Model(&entity.OrderStatus{}).
		Select("id").Where("status_name = ?", name).First(&row).Error
	return row.ID, err
}
