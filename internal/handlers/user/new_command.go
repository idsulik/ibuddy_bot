package user

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleNewCommand(ctx context.Context, message *tgbotapi.Message) {
	user := h.getCurrentUser()

	user.ActiveChatId = nil

	h.storage.UpdateUser(ctx, user)

	msg := h.newSystemMessage(message.Chat.ID, "New context started")
	msg.ReplyToMessageID = message.MessageID

	h.bot.Send(msg)
	h.bot.Send(
		tgbotapi.UnpinAllChatMessagesConfig{
			ChatID: message.Chat.ID,
		},
	)
}
