package entity

import (
	"gorm.io/gorm"
)

type OrderItemSelection struct {
	gorm.Model
	OrderItemID   uint
	OrderItem     OrderItem 
	OptionID      uint
	Option        Option 
	OptionValueID uint
	OptionValue   OptionValue 
	PriceDelta    int64       
}