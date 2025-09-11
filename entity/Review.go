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

	UserID uint `json:"userId" gorm:"not null;index"` // ผู้รีวิว
	User   User `json:"-"`

	RestaurantID uint       `json:"restaurantId" gorm:"not null;index;index:idx_restaurant_date,priority:1"` // เร่ง query ตามร้าน
	Restaurant   Restaurant `json:"-"`

	OrderID uint  `json:"orderId" gorm:"not null;uniqueIndex"` // 1 order = 1 review
	Order   Order `json:"-"`
}