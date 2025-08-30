package entity

import (
	"gorm.io/gorm"
)

type PromoType struct {
	gorm.Model
	NameType string `gorm:"size:100;uniqueIndex;not null" json:"nameType"`

	// ไม่จำเป็นต้องส่ง relation ทุกครั้ง
	Promotions []Promotion `json:"-"`
}
