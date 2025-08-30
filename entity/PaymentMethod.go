package entity

import (
	"gorm.io/gorm"
)

type PaymentMethod struct {
	gorm.Model
	MethodName string `gorm:"size:100;uniqueIndex;not null" json:"methodName"`

	// ไม่จำเป็นต้องส่ง relation ทุกครั้ง
	Payments []Payment `json:"-"`
}
