package entity

import (
	"gorm.io/gorm"
)

type UserPromotion struct {
	gorm.Model
	PromotionID uint      `json:"promotionId"`
	Promotion   Promotion `json:"-"` // preload เฉพาะตอน detail

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload เฉพาะตอนต้องการชื่อ user

	IsUsed bool `json:"isUsed"`
}

