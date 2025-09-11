package entity

import (
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	Subtotal    int64 `json:"subtotal"`
	Discount    int64 `json:"discount"`
	DeliveryFee int64 `json:"deliveryFee"`
	Total       int64 `json:"total"`

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload เฉพาะตอนต้องการ user detail

	RestaurantID uint `json:"restaurantId"`
	Restaurant   Restaurant `json:"-"` // preload เมื่อจำเป็น

	OrderStatusID uint        `json:"orderStatusId"`
	OrderStatus   OrderStatus `json:"orderStatus"`

	// preload แค่ตอน detail
	OrderItems []OrderItem `json:"-"`
	Payments   []Payment   `json:"-"`
	Reviews    []Review    `json:"-"`

	// ความสัมพันธ์หนัก → preload เฉพาะ endpoint เฉพาะ
	ChatRoom  *ChatRoom   `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	RiderWork []RiderWork `gorm:"foreignKey:OrderID" json:"-"`
}
