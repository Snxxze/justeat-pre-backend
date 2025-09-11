package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

type RiderWorkRepository struct{ DB *gorm.DB }

func NewRiderWorkRepository(db *gorm.DB) *RiderWorkRepository { return &RiderWorkRepository{DB: db} }

func (rw *RiderWorkRepository) CreateAssign(tx *gorm.DB, riderID, orderID uint, workAt time.Time) error {
	w := entity.RiderWork{
		RiderID: riderID,
		OrderID: orderID,
		WorkAt:  &workAt,
	}
	return tx.Create(&w).Error
}

func (rw *RiderWorkRepository) FinishWork(tx *gorm.DB, riderID, orderID uint, finishAt time.Time) error {
	res := tx.Model(&entity.RiderWork{}).
		Where("rider_id = ? AND order_id = ? AND finish_at IS NULL", riderID, orderID).
		Update("finish_at", &finishAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (rw *RiderWorkRepository) HasWorkForOrder(orderID uint) (bool, error) {
	var cnt int64
	if err := rw.DB.Model(&entity.RiderWork{}).
		Where("order_id = ? AND finish_at IS NULL", orderID).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

type AvailableOrderRow struct {
	ID             uint      `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	RestaurantName string    `json:"restaurantName"`
	CustomerName   string    `json:"customerName"`
	Address        string    `json:"address"`
	Total          int64     `json:"total"`
}

func (rw *RiderWorkRepository) ListAvailable(preparingStatusID uint, limit int) ([]AvailableOrderRow, error) {
	if limit <= 0 || limit > 100 { limit = 20 }

	var rows []AvailableOrderRow
	err := rw.DB.
		Table("orders AS o").
		Select(`
      o.id,
      o.created_at,
      r.name AS restaurant_name,
      CONCAT(u.first_name, ' ', u.last_name) AS customer_name,
      o.address,
      o.total
    `).
		Joins("JOIN users u ON u.id = o.user_id").
		Joins("JOIN restaurants r ON r.id = o.restaurant_id").
		// ถ้ายังไม่มีงานของ order นี้ที่ยังไม่จบ -> แปลว่ายังว่าง
		Joins("LEFT JOIN rider_works rw2 ON rw2.order_id = o.id AND rw2.finish_at IS NULL").
		Where("o.order_status_id = ? AND rw2.id IS NULL", preparingStatusID).
		Order("o.id DESC").
		Limit(limit).
		Scan(&rows).Error

	return rows, err
}

func (rw *RiderWorkRepository) FindActiveWork(riderID uint) (*AvailableOrderRow, error) {
    var row AvailableOrderRow
    err := rw.DB.
        Table("rider_works rw").
        Select(`
          o.id,
          o.created_at,
          r.name AS restaurant_name,
          CONCAT(u.first_name, ' ', u.last_name) AS customer_name,
          o.address,
          o.total
        `).
        Joins("JOIN orders o ON o.id = rw.order_id").
        Joins("JOIN users u ON u.id = o.user_id").
        Joins("JOIN restaurants r ON r.id = o.restaurant_id").
        Where("rw.rider_id = ? AND rw.finish_at IS NULL", riderID).
        Limit(1).
        Scan(&row).Error

    if err != nil {
        return nil, err
    }
    if row.ID == 0 {
        return nil, nil
    }
    return &row, nil
}
