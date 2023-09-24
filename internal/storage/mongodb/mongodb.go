package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"ibuddy_bot/internal/models"
)

const (
	databaseName           = "ibuddy"
	usersCollectionName    = "users"
	chatsCollectionName    = "chats"
	messagesCollectionName = "messages"
)

type Mongo struct {
	client *mongo.Client
}

func New(ctx context.Context, uri string) (*Mongo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		return nil, err
	}

	return &Mongo{
		client: client,
	}, nil
}

func Init(ctx context.Context, db *Mongo) error {
	var err error
	collections, err := db.client.Database(databaseName).ListCollectionNames(ctx, bson.M{})

	if err != nil {
		return err
	}

	collectionMap := make(map[string]bool)
	for _, name := range collections {
		collectionMap[name] = true
	}

	if !collectionMap[usersCollectionName] {
		err = db.client.Database(databaseName).CreateCollection(ctx, usersCollectionName)
	}
	if !collectionMap[chatsCollectionName] {
		err = db.client.Database(databaseName).CreateCollection(ctx, chatsCollectionName)
	}
	if !collectionMap[messagesCollectionName] {
		err = db.client.Database(databaseName).CreateCollection(ctx, messagesCollectionName)
	}

	return err
}

func (db *Mongo) Disconnect(ctx context.Context) error {
	return db.client.Disconnect(ctx)
}

func (db *Mongo) GetUserById(ctx context.Context, userId int64) (models.User, error) {
	var result models.User

	err := db.client.Database(databaseName).Collection(usersCollectionName).FindOne(
		ctx,
		bson.M{"id": userId},
	).Decode(&result)

	return result, err
}

func (db *Mongo) GetOrCreateUser(ctx context.Context, userId int64, newUser *models.User) (models.User, error) {
	user, err := db.GetUserById(ctx, userId)

	if errors.Is(err, mongo.ErrNoDocuments) {
		db.CreateUser(ctx, newUser)
		user, err = db.GetUserById(ctx, userId)
	}

	return user, err
}

func (db *Mongo) CreateUser(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).InsertOne(ctx, user)
}

func (db *Mongo) UpdateUser(ctx context.Context, user *models.User) (*mongo.UpdateResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).ReplaceOne(
		ctx,
		bson.M{"id": user.Id},
		user,
	)
}

func (db *Mongo) GetChatById(ctx context.Context, chatId primitive.ObjectID) (models.Chat, error) {
	var result models.Chat

	err := db.client.Database(databaseName).Collection(chatsCollectionName).FindOne(
		ctx,
		bson.M{"_id": chatId},
	).Decode(&result)

	return result, err
}

func (db *Mongo) ListUserChats(ctx context.Context, id int64) ([]models.Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		ctx,
		bson.M{"user_id": id},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	items := make([]models.Chat, 0)
	err = cur.All(ctx, &items)

	return items, err
}

func (db *Mongo) ListChatMessages(ctx context.Context, id primitive.ObjectID, limit *int64) ([]models.Message, error) {
	cur, err := db.client.Database(databaseName).Collection(messagesCollectionName).Find(
		ctx,
		bson.M{"chat_id": id},
		&options.FindOptions{
			Limit: limit,
			Sort:  bson.M{"_id": -1},
		},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	items := make([]models.Message, 0)
	err = cur.All(ctx, &items)

	return items, err
}

func (db *Mongo) InsertMessage(ctx context.Context, message models.Message) (*primitive.ObjectID, error) {
	res, err := db.client.Database(databaseName).Collection(messagesCollectionName).InsertOne(
		ctx,
		message,
	)

	if err != nil {
		return nil, err
	}

	id, _ := res.InsertedID.(primitive.ObjectID)

	return &id, nil
}

func (db *Mongo) CreateChat(ctx context.Context, chat models.Chat) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(chatsCollectionName).InsertOne(
		ctx,
		chat,
	)
}

func (db *Mongo) ListUsers(ctx context.Context) ([]models.User, error) {
	cur, err := db.client.Database(databaseName).Collection(usersCollectionName).Find(
		ctx,
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	items := make([]models.User, 0)
	err = cur.All(ctx, &items)

	return items, err
}

func (db *Mongo) ListChats(ctx context.Context) ([]models.Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		ctx,
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)

	items := make([]models.Chat, 0)
	err = cur.All(ctx, &items)

	return items, err
}
