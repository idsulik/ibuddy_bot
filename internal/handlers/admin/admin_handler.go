package admin

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/storage"
	"ibuddy_bot/internal/util"
	"ibuddy_bot/pkg/openaiclient"
	"ibuddy_bot/pkg/tgbotclient"
)

const (
	UsersCommand = "users"
	ChatsCommand = "chats"
)

const (
	UserChatsDataPrefix = "admin:user_chats:"
	UserChatDataPrefix  = "admin:user_chat:"
	UserInfoDataPrefix  = "admin:user_info:"
	UserBanDataPrefix   = "admin:user_ban:"
	UserUnbanDataPrefix = "admin:user_unban:"
)

type Handler struct {
	bot           *tgbotclient.TgBotClient
	client        *openaiclient.OpenAiClient
	storage       storage.Storage
	telegramToken string
	adminUser     string
}

func NewHandler(
	bot *tgbotclient.TgBotClient,
	client *openaiclient.OpenAiClient,
	storage storage.Storage,
	telegramToken string,
	adminUser string,
) *Handler {
	return &Handler{
		bot:           bot,
		client:        client,
		storage:       storage,
		telegramToken: telegramToken,
		adminUser:     adminUser,
	}
}

func (h *Handler) IsAdminUpdate(update *tgbotapi.Update) bool {
	msg := update.Message
	if update.CallbackQuery != nil {
		msg = update.CallbackQuery.Message
	}

	if msg.IsCommand() && strings.HasPrefix(msg.Command(), "admin") {
		return true
	}

	return update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, "admin:")
}

func (h *Handler) HandleUpdate(ctx context.Context, update *tgbotapi.Update) {
	message := update.Message

	if update.Message != nil {
		h.handleMessage(ctx, message)
	} else if update.CallbackQuery != nil {
		h.handleCallbackQuery(ctx, update.CallbackQuery)
	} else {
		log.Println("Unknown update!")
	}
}

func (h *Handler) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	switch message.CommandArguments() {
	case UsersCommand:
		h.handleUsersCommand(ctx, message)
	case ChatsCommand:
		h.handleAdminChatsCommand(ctx, message)
	default:
		h.handleDefaultCommand(message)
	}
}

func (h *Handler) handleCallbackQuery(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	switch {
	case strings.HasPrefix(callbackQuery.Data, UserChatsDataPrefix):
		h.handleUserChatsButton(ctx, callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, UserChatDataPrefix):
		h.handleUserChatButton(ctx, callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, UserBanDataPrefix):
		h.handleUserBanButton(ctx, callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, UserUnbanDataPrefix):
		h.handleUserUnbanButton(ctx, callbackQuery)
	}
}

func (h *Handler) handleUserChatsButton(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, UserChatsDataPrefix, "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(ctx, userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	chats, err := h.storage.ListUserChats(ctx, user.Id)

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
		data := fmt.Sprintf("%s%s", UserChatDataPrefix, chat.Id.Hex())
		buttons[i] = []tgbotapi.InlineKeyboardButton{
			{
				Text:         chatTitle,
				CallbackData: &data,
			},
		}
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

func (h *Handler) handleUserChatButton(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	chatId, err := primitive.ObjectIDFromHex(strings.Replace(callbackQuery.Data, UserChatDataPrefix, "", 1))

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	var limit int64 = 50
	messages, _ := h.storage.ListChatMessages(ctx, chatId, &limit)
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

func (h *Handler) handleUserBanButton(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, UserBanDataPrefix, "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(ctx, userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	banReason := "..."
	user.BanReason = &banReason

	_, err = h.storage.UpdateUser(ctx, &user)

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

func (h *Handler) handleUserUnbanButton(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	userId, err := strconv.ParseInt(strings.Replace(callbackQuery.Data, UserUnbanDataPrefix, "", 1), 10, 64)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user, err := h.storage.GetUserById(ctx, userId)

	if err != nil {
		h.newSystemReply(callbackQuery.Message, err.Error())

		return
	}

	user.BanReason = nil

	_, err = h.storage.UpdateUser(ctx, &user)

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

func (h *Handler) newSystemReply(message *tgbotapi.Message, s string) (tgbotapi.Message, error) {
	return h.bot.NewSystemReply(message, s)
}

func (h *Handler) newReplyWithFallback(
	message *tgbotapi.Message,
	responseText string,
	parseMode string,
) (tgbotapi.Message, error) {
	return h.bot.NewReplyWithFallback(message, responseText, parseMode)
}

func (h *Handler) newSystemMessage(chatId int64, text string) tgbotapi.MessageConfig {
	return h.bot.NewSystemMessage(chatId, text)
}
