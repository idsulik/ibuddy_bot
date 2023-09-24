package storage

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"ibuddy_bot/internal/models"
)

type Storage interface {
	Disconnect(ctx context.Context) error
	GetUserById(ctx context.Context, userId int64) (models.User, error)
	GetOrCreateUser(ctx context.Context, userId int64, newUser *models.User) (models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error)
	UpdateUser(ctx context.Context, user *models.User) (*mongo.UpdateResult, error)
	GetChatById(ctx context.Context, chatId primitive.ObjectID) (models.Chat, error)
	ListUserChats(ctx context.Context, id int64) ([]models.Chat, error)
	ListChatMessages(ctx context.Context, id primitive.ObjectID, limit *int64) ([]models.Message, error)
	InsertMessage(ctx context.Context, message models.Message) (*primitive.ObjectID, error)
	CreateChat(ctx context.Context, chat models.Chat) (*mongo.InsertOneResult, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	ListChats(ctx context.Context) ([]models.Chat, error)
}
