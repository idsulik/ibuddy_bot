package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

func (h *Handler) handleImageCommand(ctx context.Context, message *tgbotapi.Message) {
	prompt := strings.TrimSpace(strings.ReplaceAll(message.Text, "/image", ""))

	if len(prompt) < 3 {
		msg := h.newSystemMessage(message.Chat.ID, "Please write more information")
		msg.ReplyToMessageID = message.MessageID
		h.bot.Send(msg)
	} else {
		resp, err := h.client.CreateImage(
			ctx,
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

			msg := h.newSystemMessage(message.Chat.ID, "Failed, try again")
			msg.ReplyToMessageID = message.MessageID
			h.bot.Send(msg)
			return
		}

		files := make([]interface{}, len(resp.Data))
		for i, url := range resp.Data {
			files[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url.URL))
		}

		mediaGroup := tgbotapi.NewMediaGroup(message.Chat.ID, files)

		h.bot.SendMediaGroup(mediaGroup)
	}
}
