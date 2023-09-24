package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/localization"
)

func (h *Handler) handleStartCommand(message *tgbotapi.Message) {
	user := h.getCurrentUser()
	msg := h.newSystemMessage(message.Chat.ID, localization.GetLocalizedText(user.Lang, localization.WelcomeMessage))
	msg.ReplyToMessageID = message.MessageID
	h.bot.Send(msg)
}
