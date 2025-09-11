package entity

import (
	"gorm.io/gorm"
)

type OrderItemSelection struct {
	gorm.Model
	OrderItemID uint `json:"orderItemId"`
	OrderItem   OrderItem `json:"-"` // ไม่ serialize กลับ เพื่อเลี่ยง loop

	OptionID uint `json:"optionId"`
	Option   Option `json:"-"` // preload เฉพาะเวลาต้องการรายละเอียด option

	OptionValueID uint `json:"optionValueId"`
	OptionValue   OptionValue `json:"-"` // preload เฉพาะเวลาต้องการ label

	PriceDelta int64 `json:"priceDelta"`
}

