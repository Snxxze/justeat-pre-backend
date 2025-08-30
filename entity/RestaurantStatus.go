package entity

import (
	"gorm.io/gorm"
)

type RestaurantStatus struct {
	gorm.Model
	StatusName string `gorm:"size:100;uniqueIndex;not null" json:"statusName"`

	// ไม่จำเป็นต้อง preload restaurants ทุกครั้ง
	Restaurants []Restaurant `json:"-"`
}
