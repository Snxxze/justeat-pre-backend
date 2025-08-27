package entity

import (
	"gorm.io/gorm"
)

type PromoType struct {
	gorm.Model
	NameType   string 
	Promotions []Promotion
}