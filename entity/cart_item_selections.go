package entity

import (
	"gorm.io/gorm"
)

type CartItemSelection struct {
	gorm.Model
	CartItemID uint     `json:"cartItemId"`
	CartItem   CartItem `json:"-"`

	OptionID      uint       `json:"optionId"`
	Option        Option     `json:"-"`
	OptionValueID uint       `json:"optionValueId"`
	OptionValue   OptionValue `json:"-"`

	PriceDelta int64 `json:"priceDelta"`
}
