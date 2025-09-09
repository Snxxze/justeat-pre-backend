package entity

import (
	"time"
	"gorm.io/gorm"
)

type RiderWork struct {
	gorm.Model
	WorkAt   *time.Time `json:"workAt,omitempty"`
	FinishAt *time.Time `json:"finishAt,omitempty" gorm:"index:idx_order_finish;index:idx_rider_finish"`

	OrderID uint  `json:"orderId" gorm:"index:idx_order_finish"`
	Order   Order `json:"-"`

	RiderID uint  `json:"riderId" gorm:"index:idx_rider_finish"`
	Rider   Rider `json:"-"`
}
