package entity

import (
	"time"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	Amount  int64      
	SlipURL string     
	PaidAt  *time.Time

	PaymentMethodID uint
	PaymentMethod   PaymentMethod 
	OrderID         uint
	Order           Order 
	PaymentStatusID uint
	PaymentStatus   PaymentStatus 
}