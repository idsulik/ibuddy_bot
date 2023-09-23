package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Chat struct {
	Id       primitive.ObjectID `bson:"_id,omitempty"`
	UserId   int64              `bson:"user_id"`
	Username string             `bson:"username"`
	Title    string             `bson:"title"`
}
