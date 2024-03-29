package models

import (
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	Admin        bool
	Model        *string `bson:"model"`
	MaxTokens    int     `bson:"max_tokens"`
}

func (u *User) IsBanned() bool {
	return u.BanReason != nil
}

func (u *User) IsAdmin() bool {
	return u.Admin
}

func (u *User) GetMaxTokens() int {
	if u.MaxTokens == 0 {
		return defaultMaxTokens
	}

	return u.MaxTokens
}

func (u *User) GetModel() string {
	if u.Model != nil {
		return *u.Model
	}

	return openai.GPT3Dot5Turbo
}
