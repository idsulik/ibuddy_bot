package models

import "go.mongodb.org/mongo-driver/bson/primitive"

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
