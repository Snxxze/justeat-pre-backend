// repository/rider_repository.go
package repository

import (
	"backend/entity"
	"gorm.io/gorm"
)

type RiderRepository interface {
	Create(*entity.Rider) error
	FindByUserID(uint) (*entity.Rider, error)
}

type riderRepository struct{ db *gorm.DB }

func NewRiderRepository(db *gorm.DB) RiderRepository { return &riderRepository{db} }

func (r *riderRepository) Create(m *entity.Rider) error { return r.db.Create(m).Error }

func (r *riderRepository) FindByUserID(userID uint) (*entity.Rider, error) {
	var rd entity.Rider
	return &rd, r.db.Where("user_id = ?", userID).First(&rd).Error
}
