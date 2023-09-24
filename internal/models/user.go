package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	RoleUser         = "user"
	RoleAssistant    = "assistant"
	defaultMaxTokens = 300
)

type User struct {
	Id           int64               `bson:"id"`
	Username     string              `bson:"username"`
	ActiveChatId *primitive.ObjectID `bson:"active_chat_id"`
	BanReason    *string             `bson:"ban_reason"`
	Lang         string
	MaxTokens    int `bson:"max_tokens"`
}

func (u User) IsBanned() bool {
	return u.BanReason != nil
}

func (u User) GetMaxTokens() int {
	if u.MaxTokens == 0 {
		return defaultMaxTokens
	}

	return u.MaxTokens
}
