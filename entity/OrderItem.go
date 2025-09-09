package entity

import (
	"gorm.io/gorm"
)

type OrderItem struct {
	gorm.Model
	Qty       int   `json:"qty"`
	UnitPrice int64 `json:"unitPrice"`
	Total     int64 `json:"total"`

	OrderID uint  `json:"orderId"`
	Order   Order `json:"-"` // preload แค่ตอนต้องการ order detail

	MenuID uint `json:"menuId"`
	Menu   Menu `json:"-"` // preload เฉพาะตอนต้องการชื่อเมนู
}

