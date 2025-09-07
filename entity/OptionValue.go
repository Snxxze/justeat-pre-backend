package entity

import (
	"gorm.io/gorm"
)

type OptionValue struct {
    gorm.Model
    OptionID uint   `json:"optionId"`
    Option   Option `json:"-"`

    Name            string `json:"name"`
    PriceAdjustment int64  `json:"priceAdjustment"`
    DefaultSelect   bool   `json:"defaultSelect"`
    IsAvailable     bool   `json:"isAvailable"`
    SortOrder       int    `json:"sortOrder"`
}

