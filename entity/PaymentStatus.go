package entity

import (
	"gorm.io/gorm"
)

type PaymentStatus struct {
	gorm.Model
	StatusName string `gorm:"size:100;uniqueIndex;not null" json:"statusName"`

	// ไม่จำเป็นต้องส่ง relation ทุกครั้ง
	Payments []Payment `json:"-"`
}

