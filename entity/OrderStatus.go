package entity

import (
	"gorm.io/gorm"
)

type OrderStatus struct {
	gorm.Model
	StatusName string `json:"statusName"`

	// ไม่จำเป็นต้องส่ง relation ทุกครั้ง
	Orders []Order `json:"-"`
}
