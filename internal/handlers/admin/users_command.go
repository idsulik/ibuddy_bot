package admin

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleUsersCommand(ctx context.Context, message *tgbotapi.Message) {
	users, err := h.storage.ListUsers(ctx)
	if err != nil {
		log.Println("Error fetching users:", err)
		return
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(users))

	for i, user := range users {
		chatsData := fmt.Sprintf("%s%d", UserChatsDataPrefix, user.Id)
		banData := fmt.Sprintf("%s%d", UserBanDataPrefix, user.Id)
		unbanData := fmt.Sprintf("%s%d", UserUnbanDataPrefix, user.Id)
		userIdData := fmt.Sprintf("%s%d", UserInfoDataPrefix, user.Id)

		var banUnbanBtn tgbotapi.InlineKeyboardButton

		if user.IsBanned() {
			banUnbanBtn = tgbotapi.NewInlineKeyboardButtonData("[unban]", unbanData)
		} else {
			banUnbanBtn = tgbotapi.NewInlineKeyboardButtonData("[ban]", banData)
		}

		buttons[i] = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(user.Username, userIdData),
			tgbotapi.NewInlineKeyboardButtonData("[chats]", chatsData),
			banUnbanBtn,
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Users")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	_, err = h.bot.Send(msg)
	if err != nil {
		log.Println("Error sending message:", err)
	}
}
