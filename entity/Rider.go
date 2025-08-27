package entity

import (
	"gorm.io/gorm"
)

type Rider struct {
	gorm.Model
	VehiclePlate string
	License      string 
	DriveCar     bool   

	RiderStatusID uint
	RiderStatus   RiderStatus 

	AdminID *uint
	Admin   *Admin 
	UserID  uint
	User    User 

	Works []RiderWork
}
