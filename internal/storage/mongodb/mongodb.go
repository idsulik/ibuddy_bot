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
		err = db.client.Database(databaseName).CreateCollection(context.Background(), usersCollectionName)
	}
	if !collectionMap[chatsCollectionName] {
		err = db.client.Database(databaseName).CreateCollection(context.Background(), chatsCollectionName)
	}
	if !collectionMap[messagesCollectionName] {
		err = db.client.Database(databaseName).CreateCollection(context.Background(), messagesCollectionName)
	}

	return err
}

func (db *Mongo) Disconnect(ctx context.Context) error {
	return db.client.Disconnect(ctx)
}

func (db *Mongo) GetUserById(userId int64) (models.User, error) {
	var result models.User

	err := db.client.Database(databaseName).Collection(usersCollectionName).FindOne(
		context.Background(),
		bson.M{"id": userId},
	).Decode(&result)

	return result, err
}

func (db *Mongo) GetOrCreateUser(userId int64, newUser *models.User) (models.User, error) {
	user, err := db.GetUserById(userId)

	if errors.Is(err, mongo.ErrNoDocuments) {
		db.CreateUser(newUser)
		user, err = db.GetUserById(userId)
	}

	return user, err
}

func (db *Mongo) CreateUser(user *models.User) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).InsertOne(context.Background(), user)
}

func (db *Mongo) UpdateUser(user *models.User) (*mongo.UpdateResult, error) {
	return db.client.Database(databaseName).Collection(usersCollectionName).ReplaceOne(
		context.Background(),
		bson.M{"id": user.Id},
		user,
	)
}

func (db *Mongo) GetChatById(chatId primitive.ObjectID) (models.Chat, error) {
	var result models.Chat

	err := db.client.Database(databaseName).Collection(chatsCollectionName).FindOne(
		context.Background(),
		bson.M{"_id": chatId},
	).Decode(&result)

	return result, err
}

func (db *Mongo) ListUserChats(id int64) ([]models.Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		context.Background(),
		bson.M{"user_id": id},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]models.Chat, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Mongo) ListChatMessages(id primitive.ObjectID, limit *int64) ([]models.Message, error) {
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

	items := make([]models.Message, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Mongo) InsertMessage(message models.Message) (*primitive.ObjectID, error) {
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

func (db *Mongo) CreateChat(chat models.Chat) (*mongo.InsertOneResult, error) {
	return db.client.Database(databaseName).Collection(chatsCollectionName).InsertOne(
		context.Background(),
		chat,
	)
}

func (db *Mongo) ListUsers() ([]models.User, error) {
	cur, err := db.client.Database(databaseName).Collection(usersCollectionName).Find(
		context.Background(),
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]models.User, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}

func (db *Mongo) ListChats() ([]models.Chat, error) {
	cur, err := db.client.Database(databaseName).Collection(chatsCollectionName).Find(
		context.Background(),
		bson.M{},
	)

	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	items := make([]models.Chat, 0)
	err = cur.All(context.Background(), &items)

	return items, err
}
