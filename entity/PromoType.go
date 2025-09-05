package entity

import (
	"gorm.io/gorm"
)

type PromoType struct {
    gorm.Model
    TypeName string `gorm:"type:text;not null;default:''"` 
    Promotions []Promotion // เพิ่มความสัมพันธ์แบบ has many
}