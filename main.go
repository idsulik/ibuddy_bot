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
	telegramTokenEnvName = "TELEGRAM_TOKEN"
	chatgptKeyEnvName    = "CHATGPT_KEY"
	debugEnvName         = "DEBUG"
)

var (
	bot           *tgbotapi.BotAPI
	client        *openai.Client
	telegramToken string
)

func main() {
	var err error
	telegramToken = os.Getenv(telegramTokenEnvName)
	chatgptKey := os.Getenv(chatgptKeyEnvName)
	debug := os.Getenv(debugEnvName) == "True"
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
	} else {
		fileId := ""
		if message.Voice != nil {
			fileId = message.Voice.FileID
		} else {
			fileId = message.Audio.FileID
		}

		isVoiceText := false
		text := message.Text
		if fileId != "" {
			file, err := bot.GetFile(tgbotapi.FileConfig{
				FileID: fileId,
			})

			fileUrl := file.Link(telegramToken)
			localFile, err := downloadFileByUrl(fileUrl)
			mp3FilePath, err := convertOggToMp3(localFile.Name())

			resp, err := client.CreateTranscription(
				context.Background(),
				openai.AudioRequest{
					Model:    openai.Whisper1,
					FilePath: mp3FilePath,
				},
			)

			os.Remove(mp3FilePath)

			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)

				msg = tgbotapi.NewMessage(message.Chat.ID, "Failed, try again")
				msg.ReplyToMessageID = message.MessageID
				bot.Send(msg)
				return
			}

			if resp.Text != "" {
				text = resp.Text
				isVoiceText = true
			}
		}

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
			Content: text,
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

		responseText := resp.Choices[0].Message.Content
		if isVoiceText {
			responseText = fmt.Sprintf("voice text: %s\n\n%s", text, responseText)
		}
		msg = tgbotapi.NewMessage(message.Chat.ID, responseText)
		msg.ReplyToMessageID = message.MessageID

		bot.Send(msg)
	}

	bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, loadingMsg.MessageID))
}

func downloadFileByUrl(url string) (*os.File, error) {
	file, err := os.CreateTemp(os.TempDir(), "voice")

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
