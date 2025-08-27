package entity

import (
	"gorm.io/gorm"
)

type RestaurantStatus struct {
	gorm.Model
	StatusName  string      
	Restaurants []Restaurant 
}