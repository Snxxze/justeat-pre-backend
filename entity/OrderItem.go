package entity

import (
	"gorm.io/gorm"
)

type OrderItem struct {
	gorm.Model
	Qty       int   
	UnitPrice int64 
	Total     int64 

	OrderID uint
	Order   Order 
	MenuID  uint
	Menu    Menu 

	Selections []OrderItemSelection `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}