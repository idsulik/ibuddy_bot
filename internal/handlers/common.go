package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/models"
	"log"
	"strings"
)

func (h *UpdateHandler) newSystemReply(message *tgbotapi.Message, text string) (tgbotapi.Message, error) {
	msgConfig := h.newSystemMessage(message.Chat.ID, text)

	return h.bot.Send(msgConfig)
}

func (h *UpdateHandler) newSystemMessage(chatId int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("`%s`", text))
	msg.ParseMode = tgbotapi.ModeMarkdown

	return msg
}

func (h *UpdateHandler) pinMessage(chatId int64, messageId int) {
	_, err := h.bot.Send(tgbotapi.PinChatMessageConfig{
		ChatID:              chatId,
		MessageID:           messageId,
		DisableNotification: true,
	})

	if err != nil {
		log.Println(err)
	}
}

func (h *UpdateHandler) newReplyWithFallback(message *tgbotapi.Message, responseText string, parseMode string) (tgbotapi.Message, error) {
	if len(responseText) > 4096 {
		responseText = responseText[:4096]
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = parseMode
	msg.ReplyToMessageID = message.MessageID

	res, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)

		if strings.Contains(err.Error(), "can't parse entities") {
			if parseMode == tgbotapi.ModeMarkdownV2 {
				return h.newReplyWithFallback(
					message,
					responseText,
					tgbotapi.ModeMarkdown,
				)
			} else {
				return h.newReplyWithFallback(
					message,
					responseText,
					"",
				)
			}
		}
	}

	return res, err
}

func (h *UpdateHandler) getCurrentUser(message *tgbotapi.Message) models.User {
	userId := message.From.ID
	username := message.From.UserName
	lang := message.From.LanguageCode

	if message.From.IsBot {
		if message.ReplyToMessage != nil {
			userId = message.ReplyToMessage.From.ID
			username = message.ReplyToMessage.From.UserName
			lang = message.ReplyToMessage.From.LanguageCode
		}
	}

	user, _ := h.storage.GetOrCreateUser(userId, &models.User{
		Id:       userId,
		Username: username,
	})

	user.Lang = lang

	return user
}

func (h *UpdateHandler) changeUserActiveChat(user *models.User, chatId primitive.ObjectID) {
	user.ActiveChatId = &chatId
	h.storage.UpdateUser(user)
}
