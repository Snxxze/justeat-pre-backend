package entity

import (
	"gorm.io/gorm"
)

type Admin struct {
	gorm.Model
	Name   string 
	UserID uint  
	User   User   
	Restaurants []Restaurant
	Riders      []Rider
	Promotions  []Promotion
	Reports     []Report
}