package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// ---------------------- Rooms ----------------------

// หา/สร้าง ChatRoom ตาม orderID
func (r *ChatRepository) FindOrCreateRoom(orderID uint) (*entity.ChatRoom, error) {
	var room entity.ChatRoom
	err := r.db.Where("order_id = ?", orderID).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		room = entity.ChatRoom{OrderID: orderID}
		if err := r.db.Create(&room).Error; err != nil {
			return nil, err
		}
		return &room, nil
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// ดึงห้องตาม roomID
func (r *ChatRepository) FindRoomByID(roomID uint) (*entity.ChatRoom, error) {
	var room entity.ChatRoom
	if err := r.db.First(&room, roomID).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// ---------------------- Messages ----------------------

// ดึงข้อความทั้งหมดในห้อง
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

// บันทึกข้อความใหม่
func (r *ChatRepository) CreateMessage(msg *entity.Message) error {
	return r.db.Create(msg).Error
}

// ---------------------- Orders ----------------------

// ดึง order พร้อม RiderWork เพื่อใช้ตรวจสิทธิ์
func (r *ChatRepository) FindOrderWithRider(orderID uint) (*entity.Order, error) {
    var order entity.Order
    err := r.db.
        Preload("RiderWork.Rider.User"). // ✅ preload ให้ Rider มี User
        First(&order, orderID).Error
    return &order, err
}

