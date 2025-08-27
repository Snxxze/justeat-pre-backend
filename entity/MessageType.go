package entity

import (
	"gorm.io/gorm"
)

type MessageType struct {
	gorm.Model
	TypeName string `gorm:"size:100;uniqueIndex;not null"`
	Messages []Message	`gorm:"foreignKey:TypeMessageID;references:ID"`
}