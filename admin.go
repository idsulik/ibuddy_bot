package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/database"
	"log"
	"strconv"
	"strings"
)

func handleAdminCommand(message *tgbotapi.Message) {
	if message.From.UserName != adminUser {
		handleUnknownCommand(message)
		return
	}

	switch message.CommandArguments() {
	case "users":
		handleAdminUsersCommand(message)
	case "chats":
		handleAdminChatsCommand(message)
	default:
		handleAdminDefaultCommand(message)
	}
}

func handleAdminDefaultCommand(message *tgbotapi.Message) {
	text := fmt.Sprintf("`/admin users`\n`/admin chats`\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	bot.Send(msg)
}

func handleAdminUsersCommand(message *tgbotapi.Message) {
	users, _ := db.ListUsers()

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(users))

	for i, user := range users {
		chatsData := fmt.Sprintf("user_chats: %d", user.Id)
		banData := fmt.Sprintf("user_ban: %d", user.Id)
		unbanData := fmt.Sprintf("user_unban: %d", user.Id)
		var banUnbanBtn tgbotapi.InlineKeyboardButton

		if user.IsBanned() {
			banUnbanBtn = tgbotapi.InlineKeyboardButton{
				Text:         "[unban]",
				CallbackData: &unbanData,
			}
		} else {
			banUnbanBtn = tgbotapi.InlineKeyboardButton{
				Text:         "[ban]",
				CallbackData: &banData,
			}
		}

		buttons[i] = []tgbotapi.InlineKeyboardButton{{
			Text:         user.Username,
			CallbackData: &user.Username,
		}, {
			Text:         "[chats]",
			CallbackData: &chatsData,
		}, banUnbanBtn}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Users")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	bot.Send(msg)
}

func handleAdminChatsCommand(message *tgbotapi.Message) {
	chats, _ := db.ListChats()
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

	msg := tgbotapi.NewMessage(message.Chat.ID, "Chats")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	_, err := bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func handleUserChatsButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != adminUser {
		handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_chats: ", "", 1), 10, 64)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := db.GetUserById(userId)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	chats, err := db.ListUserChats(user.Id)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

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
		newSystemReply(callbackQuery.Message, "No chats found")

		return
	}

	text := fmt.Sprintf("@%s chats", user.Username)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	_, err = bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func handleUserChatButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != adminUser {
		handleUnknownCommand(callbackQuery.Message)
		return
	}

	chatId, err := primitive.ObjectIDFromHex(strings.Replace(callbackQuery.Data, "user_chat: ", "", 1))

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	var limit int64 = 50
	messages, _ := db.ListChatMessages(chatId, &limit)
	ReverseSlice(messages)

	items := make([]string, len(messages))
	for i, msg := range messages {
		if msg.Role == database.RoleUser {
			items[i] = fmt.Sprintf("%s:\n%s", getUserMention(msg.UserId, msg.Username), msg.Text)
		} else {
			items[i] = fmt.Sprintf("`Assistant's answer`:\n%s", msg.Text)
		}
	}

	_, err = newReplyWithFallback(callbackQuery.Message, strings.Join(items, "\n\n"), tgbotapi.ModeMarkdownV2)

	if err != nil {
		log.Println(err)
	}
}

func handleUserBanButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != adminUser {
		handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_ban: ", "", 1), 10, 64)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := db.GetUserById(userId)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	banReason := "..."
	user.BanReason = &banReason

	_, err = db.UpdateUser(&user)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	text := fmt.Sprintf("User @%s banned with reason `%s`", user.Username, banReason)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	bot.Send(msg)
}

func handleUserUnbanButton(callbackQuery *tgbotapi.CallbackQuery) {
	if callbackQuery.From.UserName != adminUser {
		handleUnknownCommand(callbackQuery.Message)
		return
	}

	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, "user_unban: ", "", 1), 10, 64)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := db.GetUserById(userId)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user.BanReason = nil

	_, err = db.UpdateUser(&user)

	if err != nil {
		newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	text := fmt.Sprintf("User @%s unbanned", user.Username)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyToMessageID = callbackQuery.Message.MessageID

	bot.Send(msg)
}
