package entity

import (
	"gorm.io/gorm"
)

type ChatRoom struct {
	gorm.Model
	OrderID uint `json:"orderId"`

	// preload เฉพาะเวลาต้องการรายละเอียด order
	Order Order `json:"-"`

	// preload messages เฉพาะ endpoint ที่ต้องการ (เช่น /chatrooms/:id/messages)
	Messages []Message `gorm:"foreignKey:RoomID;references:ID" json:"-"`
}