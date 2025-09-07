package entity

import (
	"time"
	"gorm.io/gorm"
)

type Review struct {
	gorm.Model
	Rating     int       `json:"rating"`
	Comments   string    `json:"comments"`
	ReviewDate time.Time `json:"reviewDate"`

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload เฉพาะตอนต้องการแสดงชื่อ user

	RestaurantID uint       `json:"restaurantId"`
	Restaurant   Restaurant `json:"-"` // preload เฉพาะตอน detail

	OrderID uint  `json:"orderId"`
	Order   Order `json:"-"` // preload เฉพาะตอนต้องการ
}


