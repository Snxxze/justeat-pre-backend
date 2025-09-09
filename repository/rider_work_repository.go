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
