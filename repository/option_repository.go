package repository

import (
	"backend/entity"
	"gorm.io/gorm"
)

type OptionRepository struct {
	DB *gorm.DB
}

func NewOptionRepository(db *gorm.DB) *OptionRepository {
	return &OptionRepository{DB: db}
}

// ดึง Option ทั้งหมด (พร้อม OptionValues)
func (r *OptionRepository) FindAll() ([]entity.Option, error) {
	var opts []entity.Option
	err := r.DB.Preload("OptionValues").Find(&opts).Error
	return opts, err
}

// ดึง Option ตาม ID
func (r *OptionRepository) FindByID(id uint) (*entity.Option, error) {
	var opt entity.Option
	err := r.DB.Preload("OptionValues").First(&opt, id).Error
	if err != nil {
		return nil, err
	}
	return &opt, nil
}

// สร้าง Option ใหม่
func (r *OptionRepository) Create(opt *entity.Option) error {
	return r.DB.Create(opt).Error
}

// อัปเดต Option
func (r *OptionRepository) Update(opt *entity.Option) error {
	return r.DB.Save(opt).Error
}

// ลบ Option
func (r *OptionRepository) Delete(id uint) error {
	return r.DB.Delete(&entity.Option{}, id).Error
}
