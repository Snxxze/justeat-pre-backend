package entity

import "gorm.io/gorm"

type MessageType struct {
	gorm.Model
	Name     string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Messages []Message `gorm:"foreignKey:TypeMessageID;references:ID" json:"-"`
}
