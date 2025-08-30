package entity

import (
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	Body string `json:"body"`

	TypeMessageID uint        `json:"typeMessageId"`
	TypeMessage   MessageType `json:"-"` // preload เฉพาะตอน detail

	UserSenderID uint `json:"userSenderId"`
	UserSender   User `json:"-"` // preload แยกเมื่อจำเป็น

	RoomID uint `json:"roomId"`
	Room   ChatRoom `json:"-"` // ซ่อนเพื่อเลี่ยง loop
}
