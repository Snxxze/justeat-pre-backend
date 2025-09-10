package entity

import (
	"time"
	"gorm.io/gorm"
)

type Promotion struct {
	gorm.Model
	PromoCode   string     `gorm:"size:50;uniqueIndex;not null" json:"promoCode"`
	PromoDetail string     `json:"promoDetail"`
	Values    uint       `json:"values"`
	MinOrder    int64      `json:"minOrder"`
	StartAt     *time.Time `json:"startAt,omitempty"`
	EndAt       *time.Time `json:"endAt,omitempty"`

	PromoTypeID uint      `json:"promoTypeId"`
	PromoType   PromoType `json:"-"` // preload เฉพาะตอน detail

	AdminID uint  `json:"adminId"`
	Admin   Admin `json:"-"` // preload เฉพาะ endpoint ที่ต้องแสดง admin
	

	UserPromotions []UserPromotion `json:"-"` // preload เฉพาะตอนต้องการดูว่า user ใช้โปรหรือยัง
}

