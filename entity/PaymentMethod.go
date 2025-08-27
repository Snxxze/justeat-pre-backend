package entity

import (
	"gorm.io/gorm"
)

type PaymentMethod struct {
	gorm.Model
	MethodName string 
	Payments   []Payment
}