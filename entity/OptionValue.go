package entity

import (
	"gorm.io/gorm"
)

type OptionValue struct {
	gorm.Model
	OptionID uint `json:"optionId"`

	// ไม่ต้อง serialize option กลับไปเพื่อเลี่ยง loop
	Option Option `json:"-"`

	ValueName       string `json:"valueName"`
	PriceAdjustment int64  `json:"priceAdjustment"`
	DefaultSelect   bool   `json:"defaultSelect"`
	IsAvailable     bool   `json:"isAvailable"`
	SortOrder       int    `json:"sortOrder"`
}

