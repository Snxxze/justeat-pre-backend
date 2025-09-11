package entity

import "gorm.io/gorm"

type Message struct {
	gorm.Model
	Body string `json:"body" gorm:"type:text;not null"`

	// ประเภทข้อความ (TEXT / IMAGE / SYSTEM)
	TypeMessageID uint        `json:"typeMessageId" gorm:"not null"`
	TypeMessage   MessageType `json:"-"`

	// ใครเป็นผู้ส่ง (ลูกค้า / rider)
	UserSenderID uint `json:"userSenderId" gorm:"not null"`
	UserSender   User `json:"-"`

	RoomID uint     `json:"roomId" gorm:"not null"`
	Room   ChatRoom `json:"-"`
}
