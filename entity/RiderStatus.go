package entity

import (
	"gorm.io/gorm"
)

type RiderStatus struct {
	gorm.Model
	StatusName string `gorm:"size:100;uniqueIndex;not null" json:"statusName"`

	// ไม่จำเป็นต้อง preload riders ทุกครั้ง
	Riders []Rider `json:"-"`
}
