// services/chat_service.go
package services

import (
	"backend/entity"
	"backend/repository"
)

type ChatService struct {
	repo *repository.ChatRepository
}

func NewChatService(repo *repository.ChatRepository) *ChatService {
	return &ChatService{repo}
}

func (s *ChatService) CreateRoom(orderID uint) (*entity.ChatRoom, error) {
	room := &entity.ChatRoom{OrderID: orderID}
	err := s.repo.CreateRoom(room)
	return room, err
}

func (s *ChatService) GetRoomsByUser(userID uint) ([]entity.ChatRoom, error) {
	return s.repo.FindRoomsByUser(userID)
}

func (s *ChatService) GetMessages(roomID uint) ([]entity.Message, error) {
	return s.repo.FindMessagesByRoom(roomID)
}

func (s *ChatService) SendMessage(roomID, userID, typeID uint, body string) (*entity.Message, error) {
	msg := &entity.Message{
		Body:          body,
		UserSenderID:  userID,
		TypeMessageID: typeID,
		RoomID:        roomID,
	}
	err := s.repo.CreateMessage(msg)
	return msg, err
}
