package admin

import (
	"context"
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleAdminChatsCommand(ctx context.Context, message *tgbotapi.Message) {
	chats, _ := h.storage.ListChats(ctx)
	buttons := make([][]tgbotapi.InlineKeyboardButton, len(chats))

	for i, chat := range chats {
		chatTitle := chat.Title
		if chatTitle == "" {
			chatTitle = "[empty title]"
		}

		userMention := chat.Username
		if userMention == "" {
			userMention = strconv.FormatInt(chat.UserId, 10)
		}
		chatTitle = fmt.Sprintf("%s: %s", userMention, chatTitle)
		data := fmt.Sprintf("%s%s", UserChatDataPrefix, chat.Id.Hex())
		buttons[i] = []tgbotapi.InlineKeyboardButton{
			{
				Text:         chatTitle,
				CallbackData: &data,
			},
		}
	}

	if len(buttons) == 0 {
		h.newSystemReply(message, "No chats found")

		return
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, "Chats")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	_, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}
