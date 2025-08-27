package entity

import (
	"gorm.io/gorm"
)

type RiderStatus struct {
	gorm.Model
	StatusName string 
	Riders     []Rider
}