package service

import "github.com/Kash4299/todo-chat-app/internal/model"

type IUserService interface {
	Register(user *model.User) error
	Login(email, password string) (*model.User, error)
}

type ITodoService interface {
	Create(todo *model.Todo) error
	GetByID(id uint) (*model.Todo, error)
	GetByUserID(userID uint) ([]model.Todo, error)
	Update(todo *model.Todo) error
	Delete(id uint) error
}

type IChatService interface {
	SaveMessage(msg *model.ChatMessage) error
	GetMessagesByRoomID(roomID string) ([]model.ChatMessage, error)
	RegisterClient(roomID string, client chan *model.ChatMessage)
	UnregisterClient(roomID string, client chan *model.ChatMessage)
	BroadcastToRoom(roomID string, msg *model.ChatMessage)
}
