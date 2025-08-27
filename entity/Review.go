package entity

import (
	"time"
	"gorm.io/gorm"
)

type Review struct {
	gorm.Model
	Rating     int      
	Comments   string    
	ReviewDate time.Time

	UserID       uint
	User         User 
	RestaurantID uint
	Restaurant   Restaurant 
	OrderID      uint
	Order        Order 
}