package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"ibuddy_bot/internal/handlers"
	"ibuddy_bot/internal/storage/mongodb"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	telegramTokenEnvName = "TELEGRAM_TOKEN"
	chatgptKeyEnvName    = "CHATGPT_KEY"
	mongoDbUri           = "MONGODB_URI"
	debugEnvName         = "DEBUG"
	adminUserEnvName     = "ADMIN_USER"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	telegramToken := os.Getenv(telegramTokenEnvName)
	chatgptKey := os.Getenv(chatgptKeyEnvName)
	mongodbUri := os.Getenv(mongoDbUri)
	debug := os.Getenv(debugEnvName) == "true"
	adminUser := os.Getenv(adminUserEnvName)
	client := openai.NewClient(chatgptKey)
	bot, err := tgbotapi.NewBotAPI(telegramToken)

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

	defer storage.Disconnect(ctx)

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	updateHandler := handlers.NewUpdateHandler(bot, client, storage, telegramToken, adminUser)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updateChan := bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case update := <-updateChan:
				updateHandler.HandleUpdate(&update)
			case <-ctx.Done():
				return
			}
		}
	}()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	cancel()

	fmt.Println("Adios!")
}
