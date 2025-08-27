package entity

import (
	"gorm.io/gorm"
)

type Menu struct {
	gorm.Model
	MenuName     string 
	Detail       string 
	Price        int64  
	Picture      string 

	MenuTypeID   uint
	MenuType     MenuType 
	RestaurantID uint
	Restaurant   Restaurant 
	MenuStatusID uint
	MenuStatus   MenuStatus 

	Options []Option `gorm:"many2many:menu_options;"`
	OrderItems []OrderItem
}