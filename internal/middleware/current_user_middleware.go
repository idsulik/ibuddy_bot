package middleware

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/storage"
)

func extractFrom(update *tgbotapi.Update) *tgbotapi.User {
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From
	}
	return update.Message.From
}

func extractReplyToMessage(update *tgbotapi.Update) *tgbotapi.Message {
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.ReplyToMessage
	}
	return update.Message.ReplyToMessage
}

func CurrentUserMiddleware(
	storage storage.Storage,
	next func(context.Context, *tgbotapi.Update, *models.User),
) func(context.Context, *tgbotapi.Update) {
	return func(ctx context.Context, update *tgbotapi.Update) {
		from := extractFrom(update)

		userId := from.ID
		username := from.UserName
		lang := from.LanguageCode

		if from.IsBot {
			replyToMessage := extractReplyToMessage(update)
			if replyToMessage != nil {
				userId = replyToMessage.From.ID
				username = replyToMessage.From.UserName
				lang = replyToMessage.From.LanguageCode
			}
		}

		user, err := storage.GetOrCreateUser(
			ctx,
			userId, &models.User{
				Id:       userId,
				Username: username,
			},
		)
		if err != nil {
			log.Println(err)
			return
		}

		user.Lang = lang
		next(ctx, update, &user)
	}
}
