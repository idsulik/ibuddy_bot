package storage

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"ibuddy_bot/internal/models"
)

type Storage interface {
	Disconnect(ctx context.Context) error
	GetUserById(userId int64) (models.User, error)
	GetOrCreateUser(userId int64, newUser *models.User) (models.User, error)
	CreateUser(user *models.User) (*mongo.InsertOneResult, error)
	UpdateUser(user *models.User) (*mongo.UpdateResult, error)
	GetChatById(chatId primitive.ObjectID) (models.Chat, error)
	ListUserChats(id int64) ([]models.Chat, error)
	ListChatMessages(id primitive.ObjectID, limit *int64) ([]models.Message, error)
	InsertMessage(message models.Message) (*primitive.ObjectID, error)
	CreateChat(chat models.Chat) (*mongo.InsertOneResult, error)
	ListUsers() ([]models.User, error)
	ListChats() ([]models.Chat, error)
}
