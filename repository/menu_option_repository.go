package repository

import (
	"backend/entity"
	"gorm.io/gorm"
)

type MenuOptionRepository struct {
	DB *gorm.DB
}

func NewMenuOptionRepository(db *gorm.DB) *MenuOptionRepository {
	return &MenuOptionRepository{DB: db}
}

// Attach option เข้าเมนู
func (r *MenuOptionRepository) Attach(menuID, optionID uint) error {
	mo := entity.MenuOption{
			MenuID:   menuID,
			OptionID: optionID,
	}

	// หา record ที่มี menuID + optionID อยู่แล้ว
	return r.DB.
			Where("menu_id = ? AND option_id = ?", menuID, optionID).
			FirstOrCreate(&mo).Error
}

// Detach option ออกจากเมนู
func (r *MenuOptionRepository) Detach(menuID, optionID uint) error {
	return r.DB.Delete(&entity.MenuOption{}, "menu_id = ? AND option_id = ?", menuID, optionID).Error
}

// ดึง options ของเมนู
func (r *MenuOptionRepository) FindByMenu(menuID uint) ([]entity.Option, error) {
	var opts []entity.Option
	err := r.DB.Joins("JOIN menu_options mo ON mo.option_id = options.id").
		Where("mo.menu_id = ?", menuID).
		Preload("OptionValues").
		Find(&opts).Error
	return opts, err
}
