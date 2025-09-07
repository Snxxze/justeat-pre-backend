// repository/chat_repository.go
package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db}
}

// สร้างห้องแชทใหม่ (เชื่อมกับ order)
func (r *ChatRepository) CreateRoom(room *entity.ChatRoom) error {
	return r.db.Create(room).Error
}

// ดึงห้องแชททั้งหมดของ user (ผ่าน order)
func (r *ChatRepository) FindRoomsByUser(userID uint) ([]entity.ChatRoom, error) {
	var rooms []entity.ChatRoom
	err := r.db.
		Preload("Order").
		Where("order_id IN (?)", r.db.Table("orders").Select("id").Where("user_id = ?", userID)).
		Find(&rooms).Error
	return rooms, err
}

// ดึงข้อความในห้อง
func (r *ChatRepository) FindMessagesByRoom(roomID uint) ([]entity.Message, error) {
	var msgs []entity.Message
	err := r.db.
		Preload("UserSender").
		Preload("TypeMessage").
		Where("room_id = ?", roomID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// ส่งข้อความใหม่
func (r *ChatRepository) CreateMessage(msg *entity.Message) error {
	return r.db.Create(msg).Error
}
