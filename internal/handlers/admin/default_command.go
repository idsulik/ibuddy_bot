package admin

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleDefaultCommand(message *tgbotapi.Message) {
	text := fmt.Sprintf("`/admin users`\n`/admin chats`\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	h.bot.Send(msg)
}
