package entity

import (
	"gorm.io/gorm"
)

type OrderStatus struct {
	gorm.Model
	StatusName string 
	Orders     []Order
}