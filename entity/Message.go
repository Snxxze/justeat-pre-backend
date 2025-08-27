package entity

import (
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	Body string 

	TypeMessageID uint
	TypeMessage   MessageType 
	UserSenderID  uint
	UserSender    User 
	RoomID        uint
	Room          ChatRoom 
}