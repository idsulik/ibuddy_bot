package handlers

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/storage"
	"log"
	"strings"
)

type UpdateHandler struct {
	bot           *tgbotapi.BotAPI
	clientOpenAI  *openai.Client
	storage       storage.Storage
	telegramToken string
	adminUser     string
}

func (h *UpdateHandler) HandleUpdate(update *tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		h.handleCallbackQuery(update.CallbackQuery)
	} else {
		log.Println("Unknown update!")
	}
}

func (h *UpdateHandler) handleCommandMessage(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		h.handleStartCommand(message)
	case "image":
		h.handleImageCommand(message)
	case "new":
		h.handleNewCommand(message)
	case "chats":
		h.handleChatsCommand(message)
	case "history":
		h.handleHistoryCommand(message)
	case "admin":
		h.handleAdminCommand(message)
	default:
		h.handleUnknownCommand(message)
	}
}

func (h *UpdateHandler) handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	switch {
	case primitive.IsValidObjectID(callbackQuery.Data):
		h.handleChatSwitchButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_chats"):
		h.handleUserChatsButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_chat"):
		h.handleUserChatButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_ban"):
		h.handleUserBanButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_unban"):
		h.handleUserUnbanButton(callbackQuery)
	}
}
func (h *UpdateHandler) handleChatSwitchButton(callbackQuery *tgbotapi.CallbackQuery) {
	chatId, err := primitive.ObjectIDFromHex(callbackQuery.Data)

	if err != nil {
		log.Println(err)

		return
	}

	chat, err := h.storage.GetChatById(chatId)

	callback := tgbotapi.NewCallback(callbackQuery.ID, chat.Title)

	if _, err := h.bot.Request(callback); err != nil {
		panic(err)
	}

	user := h.getCurrentUser(callbackQuery.Message)
	msg, err := h.bot.Send(
		h.newSystemMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("Active chat: %s", chat.Title)),
	)

	if err != nil {
		log.Println(err)

		return
	}

	h.changeUserActiveChat(&user, chatId)
	h.pinMessage(msg.Chat.ID, msg.MessageID)
}

func NewUpdateHandler(bot *tgbotapi.BotAPI, client *openai.Client, storage storage.Storage, telegramToken string, adminUser string) *UpdateHandler {
	return &UpdateHandler{
		bot:           bot,
		clientOpenAI:  client,
		storage:       storage,
		telegramToken: telegramToken,
		adminUser:     adminUser,
	}
}
