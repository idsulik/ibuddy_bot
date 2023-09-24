package user

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/util"
)

func (h *Handler) handleHistoryCommand(ctx context.Context, message *tgbotapi.Message) {
	user := h.getCurrentUser()

	if user.ActiveChatId == nil {
		h.newSystemReply(message, "There is no active chat")

		return
	}

	var limit int64 = 10
	messages, _ := h.storage.ListChatMessages(ctx, *user.ActiveChatId, &limit)
	util.ReverseSlice(messages)

	items := make([]string, len(messages))
	for i, msg := range messages {
		if msg.Role == models.RoleUser {
			items[i] = fmt.Sprintf("*Your message*:\n%s", msg.Text)
		} else {
			items[i] = fmt.Sprintf("`Assistant's answer`:\n%s", msg.Text)
		}
	}

	var text string

	if len(items) == 10 {
		text = fmt.Sprintf("Last %d messages:\n %s", limit, strings.Join(items, "\n\n"))
	} else {
		text = strings.Join(items, "\n\n")
	}
	_, err := h.newReplyWithFallback(message, text, tgbotapi.ModeMarkdownV2)

	if err != nil {
		log.Println(err)
	}
}
