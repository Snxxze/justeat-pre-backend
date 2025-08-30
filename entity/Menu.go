package entity

import (
	"gorm.io/gorm"
)

type Menu struct {
	gorm.Model
	MenuName string `json:"menuName"`
	Detail   string `json:"detail"`
	Price    int64  `json:"price"`
	Picture  string `json:"picture"`

	MenuTypeID   uint     `json:"menuTypeId"`
	MenuType     MenuType `json:"-"` // preload เฉพาะตอน detail

	RestaurantID uint `json:"restaurantId"`
	Restaurant   Restaurant `json:"-"` // preload เมื่อจำเป็น

	MenuStatusID uint       `json:"menuStatusId"`
	MenuStatus   MenuStatus `json:"-"` // preload เฉพาะ endpoint จัดการเมนู

	Options    []Option    `gorm:"many2many:menu_options;" json:"-"`
	OrderItems []OrderItem `json:"-"`
}
