package entity

import (
	"time"
	"gorm.io/gorm"
)

type RiderWork struct {
	gorm.Model
	WorkAt   *time.Time `json:"workAt,omitempty"`
	FinishAt *time.Time `json:"finishAt,omitempty"`

	OrderID uint  `json:"orderId"`
	Order   Order `json:"-"` // preload เฉพาะตอน detail

	RiderID uint  `json:"riderId"`
	Rider   Rider `json:"-"` // preload เฉพาะตอนต้องการดูข้อมูล rider
}

