package database

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type User struct {
	Id           int64               `bson:"id"`
	Username     string              `bson:"username"`
	ActiveChatId *primitive.ObjectID `bson:"active_chat_id"`
	BanReason    *string             `bson:"ban_reason"`
}

func (u User) IsBanned() bool {
	return u.BanReason != nil
}

type Chat struct {
	Id     primitive.ObjectID `bson:"_id,omitempty"`
	UserId int64              `bson:"user_id"`
	Title  string             `bson:"title"`
}

type Message struct {
	ChatId     primitive.ObjectID `bson:"chat_id"`
	ReplyTo    primitive.ObjectID `bson:"reply_to"`
	UserId     int64              `bson:"user_id"`
	Role       string             `bson:"role"`
	Text       string             `bson:"text"`
	Additional interface{}        `bson:"additional"`
}
