package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

type RiderRepository struct{ DB *gorm.DB }

func NewRiderRepository(db *gorm.DB) *RiderRepository { return &RiderRepository{DB: db} }

func (r *RiderRepository) GetByUserID(userID uint) (*entity.Rider, error) {
	var rd entity.Rider
	if err := r.DB.Where("user_id = ?", userID).First(&rd).Error; err != nil {
		return nil, err
	}
	return &rd, nil
}

func (r *RiderRepository) UpdateStatus(tx *gorm.DB, riderID, statusID uint) error {
	return tx.Model(&entity.Rider{}).Where("id = ?", riderID).
		Update("rider_status_id", statusID).Error
}

func (r *RiderRepository) HasActiveWork(riderID uint) (bool, error) {
	var cnt int64
	if err := r.DB.Model(&entity.RiderWork{}).
		Where("rider_id = ? AND finish_at IS NULL", riderID).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// ✅ helper: หา id ของ RiderStatus จากชื่อ (ใช้ตอน initIDs)
func (r *RiderRepository) GetStatusIDByName(name string) (uint, error) {
	var row struct{ ID uint }
	if err := r.DB.Model(&entity.RiderStatus{}).
		Select("id").
		Where("status_name = ?", name).
		Limit(1).
		Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}
