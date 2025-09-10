package entity

import (
	"gorm.io/gorm"
)

type PromoType struct {
    gorm.Model
    NameType string `gorm:"type:text;not null;default:''column:name_type" json:"nameType"` 
    Promotions []Promotion `gorm:"foreignKey:PromoTypeID"`
}