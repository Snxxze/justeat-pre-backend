// entity/user_promotion.go
package entity

import (
	"gorm.io/gorm"
)

type UserPromotion struct {
	gorm.Model
	// FK ไปตาราง promotions
	PromotionID uint      `json:"promotionId" gorm:"index:uniq_user_promo,unique"`
	Promotion   Promotion `json:"Promotion" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// FK ไป users (ได้จาก JWT)
	UserID uint `json:"userId" gorm:"index:uniq_user_promo,unique"`
	User   User `json:"-"`

	IsUsed bool `json:"isUsed"`
}

func (UserPromotion) TableName() string { return "user_promotions" }
