package repository

import (
	"backend/entity"
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
