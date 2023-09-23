package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/util"
	"log"
	"strconv"
	"strings"
)

func (h *UpdateHandler) handleAdminCommand(message *tgbotapi.Message) {
	if message.From.UserName != h.adminUser {
		h.handleUnknownCommand(message)
		return
	}

	switch message.CommandArguments() {
	case "users":
		h.handleAdminUsersCommand(message)
	case "chats":
		h.handleAdminChatsCommand(message)
	default:
		h.handleAdminDefaultCommand(message)
	}
}

func (h *UpdateHandler) handleAdminDefaultCommand(message *tgbotapi.Message) {
	text := fmt.Sprintf("`/admin users`\n`/admin chats`\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	h.bot.Send(msg)
}

func (h *UpdateHandler) handleAdminUsersCommand(message *tgbotapi.Message) {
	users, err := h.storage.ListUsers()
	if err != nil {
		log.Println("Error fetching users:", err)
		return
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(users))

	for i, user := range users {
		chatsData := fmt.Sprintf("user_chats: %d", user.Id)
		banData := fmt.Sprintf("user_ban: %d", user.Id)
		unbanData := fmt.Sprintf("user_unban: %d", user.Id)
		userIdData := fmt.Sprintf("user_info: %d", user.Id)

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

func (h *UpdateHandler) handleAdminChatsCommand(message *tgbotapi.Message) {
	chats, _ := h.storage.ListChats()
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
		data := fmt.Sprintf("user_chat: %s", chat.Id.Hex())
		buttons[i] = []tgbotapi.InlineKeyboardButton{{
			Text:         chatTitle,
			CallbackData: &data,
		}}
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

func (h *UpdateHandler) handleUserChatsButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != h.adminUser {
		h.handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_chats: ", "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	chats, err := h.storage.ListUserChats(user.Id)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(chats))

	for i, chat := range chats {
		chatTitle := chat.Title
		if chatTitle == "" {
			chatTitle = "[empty title]"
		}
		data := fmt.Sprintf("user_chat: %s", chat.Id.Hex())
		buttons[i] = []tgbotapi.InlineKeyboardButton{{
			Text:         chatTitle,
			CallbackData: &data,
		}}
	}

	if len(buttons) == 0 {
		h.newSystemReply(callbackQuery.Message, "No chats found")

		return
	}

	text := fmt.Sprintf("@%s chats", user.Username)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	_, err = h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func (h *UpdateHandler) handleUserChatButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != h.adminUser {
		h.handleUnknownCommand(callbackQuery.Message)
		return
	}

	chatId, err := primitive.ObjectIDFromHex(strings.Replace(callbackQuery.Data, "user_chat: ", "", 1))

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	var limit int64 = 50
	messages, _ := h.storage.ListChatMessages(chatId, &limit)
	util.ReverseSlice(messages)

	items := make([]string, len(messages))
	for i, msg := range messages {
		if msg.Role == models.RoleUser {
			items[i] = fmt.Sprintf("%s:\n%s", util.GetUserMention(msg.UserId, msg.Username), msg.Text)
		} else {
			items[i] = fmt.Sprintf("`Assistant's answer`:\n%s", msg.Text)
		}
	}

	_, err = h.newReplyWithFallback(callbackQuery.Message, strings.Join(items, "\n\n"), tgbotapi.ModeMarkdownV2)

	if err != nil {
		log.Println(err)
	}
}

func (h *UpdateHandler) handleUserBanButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != h.adminUser {
		h.handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_ban: ", "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	banReason := "..."
	user.BanReason = &banReason

	_, err = h.storage.UpdateUser(&user)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	text := fmt.Sprintf("User @%s banned with reason `%s`", user.Username, banReason)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	h.bot.Send(msg)
}

func (h *UpdateHandler) handleUserUnbanButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != h.adminUser {
		h.handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_unban: ", "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user.BanReason = nil

	_, err = h.storage.UpdateUser(&user)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	text := fmt.Sprintf("User @%s unbanned", user.Username)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	h.bot.Send(msg)
}
