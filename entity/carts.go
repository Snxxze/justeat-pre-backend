package entity

import (
	"gorm.io/gorm"
)

type Cart struct {
	gorm.Model
	UserID       uint `json:"userId" gorm:"uniqueIndex"`
	User         User `json:"-"`
	RestaurantID uint `json:"restaurantId"`
	Restaurant   Restaurant `json:"-"`

	Items []CartItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
