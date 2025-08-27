package entity

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email	string
	Password	string
	FirstName	string
	LastName	string
	PhoneNumber	string
	Role	string	`gorm:"not null;default:customer"`

	RestaurantsOwned []Restaurant `gorm:"foreignKey:UserID"` // owner
	Orders           []Order
	Reviews          []Review
	MessagesSent     []Message `gorm:"foreignKey:UserSenderID"`
	UserPromotions   []UserPromotion
	RiderProfile     *Rider `gorm:"foreignKey:UserID"`
	Reports          []Report
}