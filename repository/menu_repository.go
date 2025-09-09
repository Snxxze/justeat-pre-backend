// repository/menu_repository.go
package repository

import (
	"backend/entity"
	"gorm.io/gorm"
)

type MenuRepository struct {
	DB *gorm.DB
}

func NewMenuRepository(db *gorm.DB) *MenuRepository {
	return &MenuRepository{DB: db}
}

// ดึงเมนูทั้งหมดของร้าน
func (r *MenuRepository) FindByRestaurant(restID uint) ([]entity.Menu, error) {
	var menus []entity.Menu
	err := r.DB.
		Preload("MenuType").
		Preload("MenuStatus").
		Where("restaurant_id = ?", restID).
		Find(&menus).Error
	return menus, err
}

// ดึงเมนูเดียว
func (r *MenuRepository) FindByID(id uint) (*entity.Menu, error) {
	var menu entity.Menu
	err := r.DB.
		Preload("MenuType").
		Preload("MenuStatus").
		First(&menu, id).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

// สร้างเมนูใหม่
func (r *MenuRepository) Create(menu *entity.Menu) error {
	return r.DB.Create(menu).Error
}

// อัปเดตเมนู
func (r *MenuRepository) Update(menu *entity.Menu) error {
	fields := map[string]interface{}{
			"name":   menu.Name,
			"detail": menu.Detail,
			"price":  menu.Price,
			"image":  menu.Image,
			"menu_type_id":   menu.MenuTypeID,
			"menu_status_id": menu.MenuStatusID,
	}

	return r.DB.Model(&entity.Menu{}).
			Where("id = ?", menu.ID).
			Updates(fields).Error
}

// ลบเมนู
func (r *MenuRepository) Delete(id uint) error {
	return r.DB.Delete(&entity.Menu{}, id).Error
}

func (r *MenuRepository) UpdateStatus(id uint, statusID uint) error {
    return r.DB.Model(&entity.Menu{}).
        Where("id = ?", id).
        Update("menu_status_id", statusID).Error
}