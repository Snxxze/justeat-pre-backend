package repository

import (
	"backend/entity"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OrderRepository struct{ DB *gorm.DB }

func NewOrderRepository(db *gorm.DB) *OrderRepository { return &OrderRepository{DB: db} }

// ---------- Lookups / Validations ----------
func (r *OrderRepository) RestaurantExists(id uint) (bool, error) {
	var cnt int64
	if err := r.DB.Model(&entity.Restaurant{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *OrderRepository) GetMenuBasics(id uint) (entity.Menu, error) {
	var m entity.Menu
	err := r.DB.Select("id, price, restaurant_id").First(&m, id).Error
	return m, err
}

func (r *OrderRepository) ValidateMenusBelongToRestaurant(menuIDs []uint, restID uint) (bool, error) {
	if len(menuIDs) == 0 { return true, nil }
	var cnt int64
	if err := r.DB.Model(&entity.Menu{}).
		Where("id IN ? AND restaurant_id = ?", menuIDs, restID).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt == int64(len(menuIDs)), nil
}

// ตรวจว่า option values ทั้งหมดเป็นของ "option" ที่ผูกกับเมนูนี้จริง
func (r *OrderRepository) CountOptionValuesBelongToMenu(menuID uint, valueIDs []uint) (int64, error) {
	if len(valueIDs) == 0 { return 0, nil }
	var cnt int64
	// ov -> options o -> menu_options mo (menu_id)
	err := r.DB.Table("option_values AS ov").
		Joins("JOIN options o ON ov.option_id = o.id").
		Joins("JOIN menu_options mo ON mo.option_id = o.id").
		Where("mo.menu_id = ? AND ov.id IN ?", menuID, valueIDs).
		Count(&cnt).Error
	return cnt, err
}

func (r *OrderRepository) GetOptionValuesByIDs(ids []uint) ([]entity.OptionValue, error) {
	if len(ids) == 0 { return nil, nil }
	var vals []entity.OptionValue
	err := r.DB.Select("id, option_id, price_adjustment").Where("id IN ?", ids).Find(&vals).Error
	return vals, err
}

// ---------- Mutations (use with Transaction) ----------
func (r *OrderRepository) CreateOrder(tx *gorm.DB, o *entity.Order) error {
	return tx.Create(o).Error
}
func (r *OrderRepository) CreateOrderItem(tx *gorm.DB, oi *entity.OrderItem) error {
	return tx.Create(oi).Error
}
func (r *OrderRepository) CreateOrderItemSelections(tx *gorm.DB, rows []entity.OrderItemSelection) error {
	if len(rows) == 0 { return nil }
	return tx.Create(&rows).Error
}

// ---------- Queries for list/detail ----------
type OrderSummary struct {
    ID            uint       `json:"id"`
    RestaurantID  uint       `json:"restaurantId"`
    Total         int64      `json:"total"`
    OrderStatusID uint       `json:"orderStatusId"`
    CreatedAt     time.Time  `json:"createdAt"` 
}

func (r *OrderRepository) ListOrdersForUser(userID uint, limit int) ([]OrderSummary, error) {
	if limit <= 0 { limit = 50 }
	var out []OrderSummary
	err := r.DB.Model(&entity.Order{}).
		Select("id, restaurant_id, total, order_status_id, created_at").
		Where("user_id = ?", userID).
		Order("id DESC").Limit(limit).
		Scan(&out).Error
	return out, err
}

func (r *OrderRepository) GetOrderForUser(userID, orderID uint) (*entity.Order, error) {
	var o entity.Order
	if err := r.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetOrderItems(orderID uint) ([]entity.OrderItem, error) {
	var items []entity.OrderItem
	err := r.DB.Model(&entity.OrderItem{}).
		Select("id, qty, unit_price, total, menu_id, order_id").
		Where("order_id = ?", orderID).
		Find(&items).Error
	return items, err
}

func (r *OrderRepository) GetPaymentMethodIDFromKey(key string) (uint, error) {
	if key == "" { return 0, nil }

	k := strings.ToLower(strings.TrimSpace(key))
	// map key จาก FE -> MethodName ใน DB
	var methodName string
	switch k {
	case "promptpay":
		methodName = "PromptPay"
	case "cod", "cash_on_delivery", "cash-on-delivery", "cash on delivery":
		methodName = "Cash on Delivery"
	default:
		// เผื่อกรณีส่งเป็นชื่อมาเลย
		methodName = key
	}

	type row struct{ ID uint }
	var rrow row
	if err := r.DB.Model(&entity.PaymentMethod{}).
		Select("id").
		Where("method_name = ?", methodName).
		Limit(1).
		Scan(&rrow).Error; err != nil {
		return 0, err
	}
	return rrow.ID, nil
}

func (r *OrderRepository) CreatePayment(tx *gorm.DB, p *entity.Payment) error {
	return tx.Create(p).Error
}

type OwnerOrderSummary struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"userId"`
	CustomerName  string    `json:"customerName"`
	Total         int64     `json:"total"`
	OrderStatusID uint      `json:"orderStatusId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (r *OrderRepository) ListOrdersForRestaurant(restID uint, statusID *uint, page, limit int) ([]OwnerOrderSummary, int64, error) {
	if page <= 0 { page = 1 }
	if limit <= 0 || limit > 200 { limit = 20 }
	offset := (page - 1) * limit

	// นับทั้งหมด (ตามเงื่อนไขเดียวกัน)
	var total int64
	dbCount := r.DB.Table("orders AS o").Where("o.restaurant_id = ?", restID)
	if statusID != nil && *statusID != 0 {
		dbCount = dbCount.Where("o.order_status_id = ?", *statusID)
	}
	if err := dbCount.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ดึงรายการ + join users เพื่อเอาชื่อลูกค้า
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
		name := strings.TrimSpace(strings.TrimSpace(r.FirstName) + " " + strings.TrimSpace(r.LastName))
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

func (r *OrderRepository) GetOrderForRestaurant(restID, orderID uint) (*entity.Order, error) {
	var o entity.Order
	if err := r.DB.
		Where("id = ? AND restaurant_id = ?", orderID, restID).
		First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}
