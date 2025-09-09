package entity

import (
	"gorm.io/gorm"
)

type CartItem struct {
	gorm.Model
	CartID uint `json:"cartId"`
	Cart   Cart `json:"-"`

	MenuID uint `json:"menuId"`
	Menu   *Menu `json:"menu" gorm:"foreignKey:MenuID;references:ID"`

	Qty       int   `json:"qty"`
	UnitPrice int64 `json:"unitPrice"`
	Total     int64 `json:"total"`
	Note       string `json:"note"`
}
