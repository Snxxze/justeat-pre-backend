package entity

import (
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	Subtotal    int64 
	Discount    int64 
	DeliveryFee int64 
	Total       int64 

	UserID       uint
	User         User 
	RestaurantID uint
	Restaurant   Restaurant 
	OrderStatusID uint
	OrderStatus   OrderStatus 

	OrderItems []OrderItem
	Payments   []Payment
	ChatRoom *ChatRoom `gorm:"foreignKey:OrderID;references:ID"`
	RiderWork  []RiderWork `gorm:"foreignKey:OrderID"`
	Reviews    []Review
}
