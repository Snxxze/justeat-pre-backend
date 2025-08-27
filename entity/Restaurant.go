package entity

import (
	"gorm.io/gorm"
)

type Restaurant struct {
	gorm.Model
	Name        string 
	Address     string 
	Description string 
	Picture     string 

	RestaurantCategoryID uint
	RestaurantCategory   RestaurantCategory 
	RestaurantStatusID   uint
	RestaurantStatus     RestaurantStatus 

	UserID uint // owner (users.id)
	User   User

	AdminID *uint
	Admin   *Admin 

	Menus   []Menu
	Orders  []Order
	Reviews []Review
}