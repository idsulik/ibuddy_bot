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
	"strings"
)

const (
	telegramToken = "TELEGRAM_TOKEN"
	chatgptKey    = "CHATGPT_KEY"
	debug         = "DEBUG"
)

func main() {
	telegramToken := os.Getenv(telegramToken)
	chatgptKey := os.Getenv(chatgptKey)
	debug := os.Getenv(debug) == "True"
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	client := openai.NewClient(chatgptKey)

	if err != nil {
		panic(err)
	}

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Loading...")
			msg.ReplyToMessageID = update.Message.MessageID

			loadingMsg, _ := bot.Send(msg)

			if strings.Contains(strings.ToLower(update.Message.Text), "generate image") {
				resp, err := client.CreateImage(
					context.Background(),
					openai.ImageRequest{
						Prompt:         update.Message.Text,
						Size:           openai.CreateImageSize256x256,
						ResponseFormat: openai.CreateImageResponseFormatURL,
						N:              2,
					},
				)

				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)

					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Failed, try again")
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
					return
				}

				files := make([]interface{}, len(resp.Data))
				for i, url := range resp.Data {
					files[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url.URL))
				}

				mediaGroup := tgbotapi.NewMediaGroup(update.Message.Chat.ID, files)

				bot.SendMediaGroup(mediaGroup)
			} else if update.Message.Voice != nil || update.Message.Audio != nil {
				fileId := ""
				if update.Message.Voice != nil {
					fileId = update.Message.Voice.FileID
				} else {
					fileId = update.Message.Audio.FileID
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

					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Failed, try again")
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
					return
				}

				msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp.Text)
				msg.ReplyToMessageID = update.Message.MessageID

				bot.Send(msg)
			} else {
				resp, err := client.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model: openai.GPT3Dot5Turbo,
						Messages: []openai.ChatCompletionMessage{
							{
								Role:    openai.ChatMessageRoleUser,
								Content: update.Message.Text,
							},
						},
					},
				)

				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)

					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Failed, try again")
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
					return
				}

				msg = tgbotapi.NewMessage(update.Message.Chat.ID, resp.Choices[0].Message.Content)
				msg.ReplyToMessageID = update.Message.MessageID

				bot.Send(msg)
			}

			bot.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, loadingMsg.MessageID))
		}
	}
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
