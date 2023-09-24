package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"ibuddy_bot/internal/handlers/admin"
	"ibuddy_bot/internal/handlers/user"
	"ibuddy_bot/internal/middleware"
	"ibuddy_bot/internal/storage/mongodb"
	"ibuddy_bot/pkg/openaiclient"
	"ibuddy_bot/pkg/tgbotclient"
)

const (
	telegramTokenEnvName = "TELEGRAM_TOKEN"
	chatgptKeyEnvName    = "CHATGPT_KEY"
	mongoDbUri           = "MONGODB_URI"
	debugEnvName         = "DEBUG"
	adminUserEnvName     = "ADMIN_USER"

	workerCount = 3
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Println("Error loading .env file")
	}

	telegramToken := os.Getenv(telegramTokenEnvName)
	chatgptKey := os.Getenv(chatgptKeyEnvName)
	mongodbUri := os.Getenv(mongoDbUri)
	debug := os.Getenv(debugEnvName) == "true"
	adminUser := os.Getenv(adminUserEnvName)
	openAiClient := openaiclient.NewOpenAiClient(chatgptKey)
	tgBotClient, err := tgbotclient.NewTgBotClient(telegramToken, debug)

	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	storage, err := mongodb.New(ctx, mongodbUri)

	if err != nil {
		log.Fatal(err)
	}

	err = mongodb.Init(ctx, storage)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", tgBotClient.Self.UserName)

	adminHandler := admin.NewHandler(tgBotClient, openAiClient, storage)
	userHandler := user.NewHandler(tgBotClient, openAiClient, storage)

	adminMiddleware := middleware.AdminMiddleware(adminHandler, userHandler)
	banCheckMiddleware := middleware.BanCheckMiddleware(tgBotClient, adminMiddleware)
	currentUserMiddleware := middleware.CurrentUserMiddleware(storage, adminUser, banCheckMiddleware)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updateChan := tgBotClient.GetUpdatesChan(u)

	for i := 0; i < workerCount; i++ {
		go func(ctx context.Context) {
			for {
				select {
				case update := <-updateChan:
					currentUserMiddleware(ctx, &update)
				case <-ctx.Done():
					return
				}
			}
		}(ctx)
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	storage.Disconnect(ctx)
	cancel()

	fmt.Println("Adios!")
}
