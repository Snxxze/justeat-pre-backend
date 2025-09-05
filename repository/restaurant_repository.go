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

// ---------- restaurant -----
func (r *RestaurantRepository) CreateRestaurant(rest *entity.Restaurant) error {
	return r.DB.Create(rest).Error
}

// -------------- Menu --------
func (r *RestaurantRepository) CountMenus(restaurantID uint) (int64, error) {
	var total int64
	err := r.DB.Model(&entity.Menu{}).
		Where("restaurant_id = ?", restaurantID).
		Count(&total).Error
	return total, err
}

func (r *RestaurantRepository) FindMenus(restaurantID uint, limit, offset int) ([]entity.Menu, error) {
	var items []entity.Menu
	err := r.DB.Model(&entity.Menu{}).
		Select("id, menu_name, price, picture, menu_status_id, menu_type_id").
		Where("restaurant_id = ?", restaurantID).
		Order("id DESC").
		Limit(limit).Offset(offset).
		Find(&items).Error
	return items, err
}

func (r *RestaurantRepository) CreateMenu(m *entity.Menu) error {
	return r.DB.Create(m).Error
}

func (r *RestaurantRepository) FindMenuByID(id uint) (*entity.Menu, error) {
	var m entity.Menu
	if err := r.DB.Select("id, restaurant_id").First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *RestaurantRepository) UpdateMenu(m *entity.Menu, updates map[string]any) error {
	return r.DB.Model(m).Updates(updates).Error
}

// ------- Dashboard -------
func (r *RestaurantRepository) CountOrdersToday(restaurantID uint, start string) (int64, error) {
	var orders int64
	err := r.DB.Model(&entity.Order{}).
		Where("restaurant_id = ? AND created_at >= ?", restaurantID, start).
		Count(&orders).Error
	return orders, err
}

func (r *RestaurantRepository) SumRevenueToday(restaurantID uint, start string) (int64, error) {
	var revenue int64
	err := r.DB.Model(&entity.Order{}).
		Select("COALESCE(SUM(total),0)").
		Where("restaurant_id = ? AND created_at >= ? AND order_status_id = ?", restaurantID, start, 4).
		Scan(&revenue).Error
	return revenue, err
}