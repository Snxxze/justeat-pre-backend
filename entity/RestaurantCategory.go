package entity

import (
	"gorm.io/gorm"
)

type RestaurantCategory struct {
	gorm.Model
	CategoryName string
	Restaurants  []Restaurant `gorm:"foreignKey:RestaurantCategoryID"`
}