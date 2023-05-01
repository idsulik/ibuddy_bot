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
	Id       primitive.ObjectID `bson:"_id,omitempty"`
	UserId   int64              `bson:"user_id"`
	Username string             `bson:"username"`
	Title    string             `bson:"title"`
}

type Message struct {
	Id         int                `bson:"id"`
	ChatId     primitive.ObjectID `bson:"chat_id"`
	ReplyToId  *int               `bson:"reply_to_id"`
	UserId     int64              `bson:"user_id"`
	Username   string             `bson:"username"`
	Role       string             `bson:"role"`
	Text       string             `bson:"text"`
	Additional interface{}        `bson:"additional"`
}
