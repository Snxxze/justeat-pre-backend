package entity

import (
	"gorm.io/gorm"
)

type Menu struct {
    gorm.Model
    Name   string `json:"name"`
    Detail string `json:"detail"`
    Price  int64  `json:"price"`

    Image string `json:"image" gorm:"type:longtext"`

    MenuTypeID   uint     `json:"menuTypeId"`
    MenuType     MenuType `json:"-"`

    RestaurantID uint       `json:"restaurantId"`
    Restaurant   Restaurant `json:"-"`

    MenuStatusID uint       `json:"menuStatusId"`
    MenuStatus   MenuStatus `json:"-"`

    Options    []Option    `gorm:"many2many:menu_options;" json:"options"`
    OrderItems []OrderItem `json:"-"`
}
