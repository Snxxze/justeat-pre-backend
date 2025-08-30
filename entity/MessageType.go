package entity

import (
	"gorm.io/gorm"
)

type MessageType struct {
	gorm.Model
	TypeName string `gorm:"size:100;uniqueIndex;not null" json:"typeName"`

	// ซ่อน relation เพื่อเลี่ยง response บวม
	Messages []Message `gorm:"foreignKey:TypeMessageID;references:ID" json:"-"`
}
