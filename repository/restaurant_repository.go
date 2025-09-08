// repository/restaurant_repository.go
package repository

import (
	"backend/entity"
	"gorm.io/gorm"
)

type RestaurantRepository struct {
	DB *gorm.DB
}

func NewRestaurantRepository(db *gorm.DB) *RestaurantRepository {
	return &RestaurantRepository{DB: db}
}

// ดึงร้านทั้งหมด
func (r *RestaurantRepository) FindAll() ([]entity.Restaurant, error) {
	var rests []entity.Restaurant
	err := r.DB.
		Preload("RestaurantCategory").
		Preload("RestaurantStatus").
		Preload("User").
		Find(&rests).Error
	return rests, err
}

// ดึงร้านตาม ID
func (r *RestaurantRepository) FindByID(id uint) (*entity.Restaurant, error) {
	var rest entity.Restaurant
	err := r.DB.
		Preload("RestaurantCategory").
		Preload("RestaurantStatus").
		Preload("User").
		First(&rest, id).Error
	if err != nil {
		return nil, err
	}
	return &rest, nil
}

// อัปเดตร้าน
func (r *RestaurantRepository) Update(rest *entity.Restaurant) error {
	return r.DB.Save(rest).Error
}

// เช็คความเป็นเจ้าของร้าน + ดึงออเดอร์ระดับร้าน
func (r *RestaurantRepository) IsOwnedBy(restID, userID uint) (bool, error) {
	var cnt int64
	err := r.DB.Model(&entity.Restaurant{}).
		Where("id = ? AND user_id = ?", restID, userID).
		Count(&cnt).Error
	return cnt > 0, err
}