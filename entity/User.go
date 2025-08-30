package entity

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email       string `gorm:"uniqueIndex;not null" json:"email"`
	Password    string `json:"-"` // ปลอดภัย
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	PhoneNumber string `json:"phoneNumber"`
	Role        string `gorm:"not null;default:customer" json:"role"`

	// Relations — preload เฉพาะตอนจำเป็น
	RestaurantsOwned []Restaurant   `gorm:"foreignKey:UserID" json:"-"`
	Orders           []Order        `json:"-"`
	Reviews          []Review       `json:"-"`
	MessagesSent     []Message      `gorm:"foreignKey:UserSenderID" json:"-"`
	UserPromotions   []UserPromotion `json:"-"`
	RiderProfile     *Rider         `gorm:"foreignKey:UserID" json:"-"`
	Reports          []Report       `json:"-"`
}
