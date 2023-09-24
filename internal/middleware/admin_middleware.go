package middleware

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/handlers/admin"
	"ibuddy_bot/internal/handlers/user"
	"ibuddy_bot/internal/models"
)

func AdminMiddleware(
	adminHandler *admin.Handler,
	userHandler *user.Handler,
) func(context.Context, *tgbotapi.Update, *models.User) {
	return func(ctx context.Context, update *tgbotapi.Update, user *models.User) {
		if user.IsAdmin() && adminHandler.IsAdminUpdate(update) {
			adminHandler.HandleUpdate(ctx, update)
		} else {
			userHandler.HandleUpdate(ctx, update, user)
		}
	}
}
