package entity

import (
	"gorm.io/gorm"
)

type ChatRoom struct {
	gorm.Model
	OrderID  uint  
	Order    Order 
	Messages []Message `gorm:"foreignKey:RoomID;references:ID"`
}