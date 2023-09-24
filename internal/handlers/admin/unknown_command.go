package admin

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleUnknownCommand(message *tgbotapi.Message) {
	msg := h.newSystemMessage(message.Chat.ID, "Unknown command")
	msg.ReplyToMessageID = message.MessageID
	_, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}
