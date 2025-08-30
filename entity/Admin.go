package entity

import (
	"gorm.io/gorm"
)

type Admin struct {
	gorm.Model
	Name string `json:"name"`

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload แยกเมื่อจำเป็น

	// Relations ซ่อนเพื่อเลี่ยง payload บวม
	Restaurants []Restaurant `json:"-"`
	Riders      []Rider      `json:"-"`
	Promotions  []Promotion  `json:"-"`
	Reports     []Report     `json:"-"`
}

