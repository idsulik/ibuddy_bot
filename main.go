package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const (
	telegramToken = "TELEGRAM_TOKEN"
	chatgptKey    = "CHATGPT_KEY"
	debug         = "DEBUG"
)

var (
	bot    *tgbotapi.BotAPI
	client *openai.Client
)

func main() {
	var err error
	telegramToken := os.Getenv(telegramToken)
	chatgptKey := os.Getenv(chatgptKey)
	debug := os.Getenv(debug) == "True"
	bot, err = tgbotapi.NewBotAPI(telegramToken)
	client = openai.NewClient(chatgptKey)

	if err != nil {
		panic(err)
	}

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	for update := range bot.GetUpdatesChan(u) {
		if update.Message == nil {
			log.Println("No message", update)
			continue
		}
		go handleMessage(update.Message)
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	//time for cleanup before exit
	fmt.Println("Adios!")
}

func handleMessage(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	msg := tgbotapi.NewMessage(message.Chat.ID, "Loading...")
	msg.ReplyToMessageID = message.MessageID

	loadingMsg, _ := bot.Send(msg)

	if strings.HasPrefix(message.Text, "/image") {
		prompt := strings.TrimSpace(strings.ReplaceAll(message.Text, "/image", ""))

		if len(prompt) < 3 {
			msg = tgbotapi.NewMessage(message.Chat.ID, "Please write more information")
			msg.ReplyToMessageID = message.MessageID
			bot.Send(msg)
		} else {
			resp, err := client.CreateImage(
				context.Background(),
				openai.ImageRequest{
					Prompt:         prompt,
					Size:           openai.CreateImageSize256x256,
					ResponseFormat: openai.CreateImageResponseFormatURL,
					N:              2,
					User:           strconv.FormatInt(message.From.ID, 10),
				},
			)

			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)

				msg = tgbotapi.NewMessage(message.Chat.ID, "Failed, try again")
				msg.ReplyToMessageID = message.MessageID
				bot.Send(msg)
				return
			}

			files := make([]interface{}, len(resp.Data))
			for i, url := range resp.Data {
				files[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url.URL))
			}

			mediaGroup := tgbotapi.NewMediaGroup(message.Chat.ID, files)

			bot.SendMediaGroup(mediaGroup)
		}
	} else if message.Voice != nil || message.Audio != nil {
		fileId := ""
		if message.Voice != nil {
			fileId = message.Voice.FileID
		} else {
			fileId = message.Audio.FileID
		}

		file, err := bot.GetFile(tgbotapi.FileConfig{
			FileID: fileId,
		})

		fileUrl := file.Link(telegramToken)

		localFile, err := downloadFileByUrl(fileUrl)

		resp, err := client.CreateTranscription(
			context.Background(),
			openai.AudioRequest{
				Model:    openai.Whisper1,
				FilePath: localFile.Name(),
			},
		)

		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)

			msg = tgbotapi.NewMessage(message.Chat.ID, "Failed, try again")
			msg.ReplyToMessageID = message.MessageID
			bot.Send(msg)
			return
		}

		msg = tgbotapi.NewMessage(message.Chat.ID, resp.Text)
		msg.ReplyToMessageID = message.MessageID

		bot.Send(msg)
	} else {
		messages := make([]openai.ChatCompletionMessage, 0)

		if message.ReplyToMessage != nil {
			if message.ReplyToMessage.From.IsBot {
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: message.ReplyToMessage.Text,
				})
			}
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message.Text,
		})

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:     openai.GPT3Dot5Turbo,
				Messages:  messages,
				MaxTokens: 300,
				User:      strconv.FormatInt(message.From.ID, 10),
			},
		)

		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)

			msg = tgbotapi.NewMessage(message.Chat.ID, "Failed, try again")
			msg.ReplyToMessageID = message.MessageID
			bot.Send(msg)
			return
		}

		msg = tgbotapi.NewMessage(message.Chat.ID, resp.Choices[0].Message.Content)
		msg.ReplyToMessageID = message.MessageID

		bot.Send(msg)
	}

	bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, loadingMsg.MessageID))
}

func downloadFileByUrl(url string) (*os.File, error) {
	file, err := os.CreateTemp(os.TempDir(), "voice_*.wav")

	if err != nil {
		return nil, err
	}
	defer file.Close()

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)

	return file, err
}
