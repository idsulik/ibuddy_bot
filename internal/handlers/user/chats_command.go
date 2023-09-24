package user

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleChatsCommand(ctx context.Context, message *tgbotapi.Message) {
	chats, _ := h.storage.ListUserChats(ctx, h.getCurrentUser().Id)

	if len(chats) == 0 {
		h.newSystemReply(message, "No chats found")
		return
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(chats))

	for i, chat := range chats {
		chatIdHex := chat.Id.Hex()
		buttons[i] = []tgbotapi.InlineKeyboardButton{
			{
				Text:         chat.Title,
				CallbackData: &chatIdHex,
			},
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Click on chat you want to switch")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	_, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}
