package entity

import (
	"time"
	"gorm.io/gorm"
)

type Promotion struct {
	gorm.Model
	PromoCode   string     
	PromoDetail string     
	IsValues    bool       
	MinOrder    int64      
	StartAt     *time.Time
	EndAt       *time.Time

	PromoTypeID uint
	PromoType   PromoType 

	AdminID uint
	Admin   Admin 

	UserPromotions []UserPromotion
}
