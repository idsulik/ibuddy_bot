package database

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName           = "ibuddy"
	usersCollectionName    = "users"
	chatsCollectionName    = "chats"
	messagesCollectionName = "messages"
)

type Database struct {
	client *mongo.Client
}

func (db *Database) Init() error {
	var err error
	err = db.client.Database(databaseName).CreateCollection(context.Background(), usersCollectionName)
	err = db.client.Database(databaseName).CreateCollection(context.Background(), chatsCollectionName)
	err = db.client.Database(databaseName).CreateCollection(context.Background(), messagesCollectionName)

	return err
}

func (db *Database) Disconnect(ctx context.Context) error {
	return db.client.Disconnect(ctx)
}

func (db *Database) GetUserById(userId int64) (User, error) {
	var result User

	err := db.client.Database(databaseName).Collection(usersCollectionName).FindOne(
		context.Background(),
		bson.M{"id": userId},
	).Decode(&result)

	return result, err
}

func (db *Database) GetOrCreateUser(userId int64, newUser *User) (User, error) {
	user, err := db.GetUserById(userId)

	if errors.Is(err, mongo.ErrNoDocuments) {
		db.CreateUser(newUser)
		user, err = db.GetUserById(userId)
	}

	return user, err
}

func (db *Database) CreateUser(user *User) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).InsertOne(context.Background(), user)
}

func (db *Database) UpdateUser(user *User) (*mongo.UpdateResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).ReplaceOne(
		context.Background(),
		bson.M{"id": user.Id},
		user,
	)
}

func (db *Database) GetChatById(chatId primitive.ObjectID) (Chat, error) {
	var result Chat

	err := db.client.Database(databaseName).Collection(chatsCollectionName).FindOne(
		context.Background(),
		bson.M{"_id": chatId},
	).Decode(&result)

	return result, err
}

func (db *Database) ListUserChats(id int64) ([]Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		context.Background(),
		bson.M{"user_id": id},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]Chat, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Database) ListChatMessages(id primitive.ObjectID, limit *int64) ([]Message, error) {
	cur, err := db.client.Database(databaseName).Collection(messagesCollectionName).Find(
		context.Background(),
		bson.M{"chat_id": id},
		&options.FindOptions{
			Limit: limit,
			Sort:  bson.M{"_id": -1},
		},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]Message, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Database) InsertMessage(message Message) (*primitive.ObjectID, error) {
	res, err := db.client.Database(databaseName).Collection(messagesCollectionName).InsertOne(
		context.Background(),
		message,
	)

	if err != nil {
		return nil, err
	}

	id, _ := res.InsertedID.(primitive.ObjectID)

	return &id, nil
}

func (db *Database) CreateChat(chat Chat) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(chatsCollectionName).InsertOne(
		context.Background(),
		chat,
	)
}

func (db *Database) ListUsers() ([]User, error) {
	cur, err := db.client.Database(databaseName).Collection(usersCollectionName).Find(
		context.Background(),
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]User, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Database) ListChats() ([]Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		context.Background(),
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]Chat, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func New(ctx context.Context, uri string) (*Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		return nil, err
	}

	return &Database{
		client: client,
	}, nil
}
