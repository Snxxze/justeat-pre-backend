package entity

import (
	"gorm.io/gorm"
)

type RestaurantCategory struct {
	gorm.Model
	CategoryName string `gorm:"size:100;uniqueIndex;not null" json:"categoryName"`

	// ไม่จำเป็นต้องส่ง relation ทุกครั้ง
	Restaurants []Restaurant `gorm:"foreignKey:RestaurantCategoryID" json:"-"`
}
