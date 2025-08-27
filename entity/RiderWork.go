package entity

import (
	"time"
	"gorm.io/gorm"
)

type RiderWork struct {
	gorm.Model
	WorkAt   *time.Time
	FinishAt *time.Time

	OrderID uint
	Order   Order 
	RiderID uint
	Rider   Rider
}