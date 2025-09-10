package services

import (
	"backend/entity"
	"backend/repository"
	"errors"

	"gorm.io/gorm"
)

type ChatService struct {
	Repo *repository.ChatRepository
	DB   *gorm.DB
}

func NewChatService(db *gorm.DB, repo *repository.ChatRepository) *ChatService {
	return &ChatService{Repo: repo, DB: db}
}

// ---------------------- Rooms ----------------------

// หา/สร้างห้องตาม orderID
func (s *ChatService) GetOrCreateRoom(orderID uint) (*entity.ChatRoom, error) {
	return s.Repo.FindOrCreateRoom(orderID)
}

// ดึงห้องตาม roomID
func (s *ChatService) GetRoomByID(roomID uint) (*entity.ChatRoom, error) {
	return s.Repo.FindRoomByID(roomID)
}

// ---------------------- Messages ----------------------

// ดึงข้อความในห้อง
func (s *ChatService) GetMessages(roomID uint) ([]entity.Message, error) {
	return s.Repo.FindMessagesByRoom(roomID)
}

// ส่งข้อความใหม่
func (s *ChatService) SendMessage(roomID, userID, typeMsgID uint, body string) (*entity.Message, error) {
	if body == "" {
		return nil, errors.New("message body cannot be empty")
	}

	msg := &entity.Message{
		Body:          body,
		UserSenderID:  userID,
		TypeMessageID: typeMsgID,
		RoomID:        roomID,
	}

	if err := s.Repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// ---------------------- Permissions ----------------------

// ตรวจสอบว่า user มีสิทธิ์เข้าถึงห้อง (customer + rider)
func (s *ChatService) CanAccessRoom(userID, orderID uint) (bool, error) {
    order, err := s.Repo.FindOrderWithRider(orderID)
    if err != nil {
        return false, err
    }

    // ลูกค้าเจ้าของ order
    if order.UserID == userID {
        return true, nil
    }

    // Rider ของ order → เทียบ userID
    for _, rw := range order.RiderWork {
        if rw.Rider.UserID == userID {
            return true, nil
        }
    }

    return false, nil
}

