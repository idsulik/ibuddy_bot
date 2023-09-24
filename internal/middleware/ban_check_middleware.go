package middleware

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/localization"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/pkg/tgbotclient"
)

func BanCheckMiddleware(
	tgBotClient *tgbotclient.TgBotClient,
	next func(context.Context, *tgbotapi.Update, *models.User),
) func(context.Context, *tgbotapi.Update, *models.User) {
	return func(ctx context.Context, update *tgbotapi.Update, user *models.User) {
		if user.IsBanned() {
			var chatId int64

			if update.CallbackQuery != nil {
				chatId = update.CallbackQuery.Message.Chat.ID
			} else {
				chatId = update.Message.Chat.ID
			}

			msg := tgBotClient.NewSystemMessage(
				chatId,
				localization.GetLocalizedText(user.Lang, localization.UserBanned, *user.BanReason),
			)

			if update.Message != nil {
				msg.ReplyToMessageID = update.Message.MessageID
			}

			_, err := tgBotClient.Send(msg)

			if err != nil {
				log.Println(err)
			}
		} else {
			next(ctx, update, user)
		}
	}
}
