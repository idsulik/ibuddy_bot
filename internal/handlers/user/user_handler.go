package user

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/localization"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/storage"
	"ibuddy_bot/internal/util"
	"ibuddy_bot/pkg/openaiclient"
	"ibuddy_bot/pkg/tgbotclient"
)

type Handler struct {
	bot           *tgbotclient.TgBotClient
	client        *openaiclient.OpenAiClient
	storage       storage.Storage
	telegramToken string
	currentUser   *models.User
}

func NewHandler(
	bot *tgbotclient.TgBotClient,
	client *openaiclient.OpenAiClient,
	storage storage.Storage,
) *Handler {
	return &Handler{
		bot:     bot,
		client:  client,
		storage: storage,
	}
}

func (h *Handler) HandleUpdate(ctx context.Context, update *tgbotapi.Update, currentUser *models.User) {
	h.setCurrentUser(currentUser)

	if update.Message != nil {
		if update.Message.IsCommand() {
			h.handleCommandMessage(ctx, update.Message)
		} else {
			h.handleMessage(ctx, update.Message)
		}
	} else if update.CallbackQuery != nil {
		h.handleCallbackQuery(ctx, update.CallbackQuery)
	} else {
		log.Println("Unknown update!")
	}
}

const (
	maxChatMessages           = 20
	maximumContextLengthError = "maximum context length"
)

func (h *Handler) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	if message.PinnedMessage != nil {
		log.Printf("Pinned message: %s", message.PinnedMessage.Text)
		return
	}

	var err error
	user := h.getCurrentUser()

	loadingMsg, _ := h.bot.SendLoadingReply(message, h.getCurrentUser().Lang)
	defer h.bot.DeleteMessage(&loadingMsg)

	messageText := strings.TrimSpace(message.Text)
	if len(messageText) < 2 {
		msg := h.newSystemMessage(
			message.Chat.ID,
			localization.GetLocalizedText(user.Lang, localization.TooShortMessage),
		)
		msg.ReplyToMessageID = message.MessageID
		_, err := h.bot.Send(msg)
		if err != nil {
			log.Println(err)
		}

		return
	}

	if user.ActiveChatId == nil {
		res, err := h.storage.CreateChat(
			ctx,
			models.Chat{
				UserId:   user.Id,
				Username: user.Username,
				Title:    messageText,
			},
		)

		if err != nil {
			log.Println(err)
		} else {
			v, _ := res.InsertedID.(primitive.ObjectID)
			h.changeUserActiveChat(ctx, user, v)
			_, err := h.bot.PinMessage(message.Chat.ID, message.MessageID)
			if err != nil {
				log.Println(err)
			}
		}
	}

	isVoiceText := false
	voiceText := h.extractVoiceText(ctx, message)

	if voiceText != "" {
		isVoiceText = true
		messageText = voiceText
	}

	var limit int64 = maxChatMessages
	activeChatMessages, _ := h.storage.ListChatMessages(ctx, *user.ActiveChatId, &limit)
	util.ReverseSlice(activeChatMessages)

	messages := make([]openai.ChatCompletionMessage, len(activeChatMessages))

	for i, msg := range activeChatMessages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Text,
		}
	}

	_, err = h.storage.InsertMessage(
		ctx,
		models.Message{
			Id:       message.MessageID,
			ChatId:   *user.ActiveChatId,
			UserId:   message.From.ID,
			Username: message.From.UserName,
			Role:     models.RoleUser,
			Text:     messageText,
		},
	)

	if err != nil {
		log.Println(err)
	}

	messages = append(
		messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: messageText,
		},
	)

	resp, err := h.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:     openai.GPT3Dot5Turbo,
			Messages:  messages,
			MaxTokens: user.GetMaxTokens(),
			User:      strconv.FormatInt(user.Id, 10),
		},
	)

	if err != nil {
		log.Println(err)

		if strings.Contains(err.Error(), maximumContextLengthError) {
			_, err = h.newSystemReply(message, fmt.Sprintf("Start new context with /new command"))
		} else {
			_, err = h.newSystemReply(message, "Failed, try again")
		}

		if err != nil {
			log.Println(err)
		}

		return
	}

	responseText := resp.Choices[0].Message.Content

	if err != nil {
		log.Println(err)
	}

	if isVoiceText {
		responseText = fmt.Sprintf("**voice text**:\n```\n%s\n```\n\n%s", messageText, responseText)
	}

	msg, err := h.newReplyWithFallback(message, responseText, tgbotapi.ModeMarkdownV2)

	if err != nil {
		log.Println(err)
	}

	_, err = h.storage.InsertMessage(
		ctx,
		models.Message{
			Id:         msg.MessageID,
			ChatId:     *user.ActiveChatId,
			ReplyToId:  &message.MessageID,
			UserId:     h.bot.Self.ID,
			Username:   h.bot.Self.UserName,
			Role:       models.RoleAssistant,
			Text:       responseText,
			Additional: resp,
		},
	)
}

func (h *Handler) extractVoiceText(ctx context.Context, message *tgbotapi.Message) string {
	fileId := ""
	if message.Voice != nil {
		fileId = message.Voice.FileID
	} else if message.Audio != nil {
		fileId = message.Audio.FileID
	} else {
		return ""
	}

	fileUrl, err := h.bot.GetFileDirectURL(fileId)
	localFile, err := util.DownloadFileByUrl(fileUrl)
	defer os.Remove(localFile.Name())
	mp3FilePath, err := util.ConvertOggToMp3(localFile.Name())
	defer os.Remove(mp3FilePath)

	if err != nil {
		fmt.Printf("error: %v\n", err)

		msg := h.newSystemMessage(message.Chat.ID, "Failed, try again")
		msg.ReplyToMessageID = message.MessageID
		h.bot.Send(msg)
		return ""
	}

	resp, err := h.client.CreateTranscription(
		ctx,
		openai.AudioRequest{
			Model:    openai.Whisper1,
			FilePath: mp3FilePath,
		},
	)

	if err != nil {
		fmt.Printf("error: %v\n", err)

		msg := h.newSystemMessage(message.Chat.ID, "Failed, try again")
		msg.ReplyToMessageID = message.MessageID
		h.bot.Send(msg)
		return ""
	}

	return resp.Text
}

func (h *Handler) handleCommandMessage(ctx context.Context, message *tgbotapi.Message) {
	loadingMsg, _ := h.bot.SendLoadingReply(message, h.getCurrentUser().Lang)
	defer h.bot.DeleteMessage(&loadingMsg)

	switch message.Command() {
	case "start":
		h.handleStartCommand(message)
	case "image":
		h.handleImageCommand(ctx, message)
	case "new":
		h.handleNewCommand(ctx, message)
	case "chats":
		h.handleChatsCommand(ctx, message)
	case "history":
		h.handleHistoryCommand(ctx, message)
	default:
		h.handleUnknownCommand(message)
	}
}

func (h *Handler) handleCallbackQuery(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	switch {
	case primitive.IsValidObjectID(callbackQuery.Data):
		h.handleChatSwitchButton(ctx, callbackQuery)
	}
}

func (h *Handler) handleChatSwitchButton(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) {
	chatId, err := primitive.ObjectIDFromHex(callbackQuery.Data)

	if err != nil {
		log.Println(err)

		return
	}

	chat, err := h.storage.GetChatById(ctx, chatId)

	callback := tgbotapi.NewCallback(callbackQuery.ID, chat.Title)

	if _, err := h.bot.Request(callback); err != nil {
		panic(err)
	}

	user := h.getCurrentUser()
	msg, err := h.bot.Send(
		h.newSystemMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("Active chat: %s", chat.Title)),
	)

	if err != nil {
		log.Println(err)

		return
	}

	h.changeUserActiveChat(ctx, user, chatId)
	h.bot.PinMessage(msg.Chat.ID, msg.MessageID)
}

func (h *Handler) changeUserActiveChat(ctx context.Context, user *models.User, chatId primitive.ObjectID) {
	user.ActiveChatId = &chatId
	h.storage.UpdateUser(ctx, user)
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

func (h *Handler) setCurrentUser(user *models.User) {
	h.currentUser = user
}

func (h *Handler) getCurrentUser() *models.User {
	return h.currentUser
}
