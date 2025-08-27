package entity

import (
	"gorm.io/gorm"
)

type UserPromotion struct {
	gorm.Model
	PromotionID uint
	Promotion   Promotion 
	UserID      uint
	User        User 
	IsUsed      bool 
}