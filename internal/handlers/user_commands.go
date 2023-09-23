package handlers

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/internal/localization"
	"ibuddy_bot/internal/models"
	"ibuddy_bot/internal/util"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	maxChatMessages = 20

	maximumContextLengthError = "maximum context length"
)

func (h *UpdateHandler) handleMessage(message *tgbotapi.Message) {
	if message.PinnedMessage != nil {
		log.Printf("Pinned message: %s", message.PinnedMessage.Text)
		return
	}

	user := h.getCurrentUser(message)

	msg := h.newSystemMessage(message.Chat.ID, localization.GetLocalizedText(user.Lang, localization.TextLoading))
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
	}
	msg.ReplyToMessageID = message.MessageID

	loadingMsg, _ := h.bot.Send(msg)

	if user.IsBanned() {
		msg := h.newSystemMessage(message.Chat.ID, localization.GetLocalizedText(user.Lang, localization.UserBanned, *user.BanReason))
		msg.ReplyToMessageID = message.MessageID

		_, err := h.bot.Send(msg)

		if err != nil {
			log.Println(err)
		}

		return
	}

	messageText := strings.TrimSpace(message.Text)
	if message.IsCommand() {
		h.handleCommandMessage(message)
	} else {
		if len(messageText) < 2 {
			msg := h.newSystemMessage(message.Chat.ID, localization.GetLocalizedText(user.Lang, localization.TooShortMessage))
			msg.ReplyToMessageID = message.MessageID
			h.bot.Send(msg)

			return
		}

		if user.ActiveChatId == nil {
			res, err := h.storage.CreateChat(models.Chat{
				UserId:   user.Id,
				Username: user.Username,
				Title:    messageText,
			})

			if err != nil {
				log.Println(err)
			} else {
				v, _ := res.InsertedID.(primitive.ObjectID)
				h.changeUserActiveChat(&user, v)
				h.pinMessage(message.Chat.ID, message.MessageID)
			}
		}

		isVoiceText := false
		voiceText := h.extractVoiceText(message)

		if voiceText != "" {
			isVoiceText = true
			messageText = voiceText
		}

		var limit int64 = maxChatMessages
		activeChatMessages, _ := h.storage.ListChatMessages(*user.ActiveChatId, &limit)
		util.ReverseSlice(activeChatMessages)

		messages := make([]openai.ChatCompletionMessage, len(activeChatMessages))

		for i, msg := range activeChatMessages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Text,
			}
		}

		_, err := h.storage.InsertMessage(models.Message{
			Id:       message.MessageID,
			ChatId:   *user.ActiveChatId,
			UserId:   message.From.ID,
			Username: message.From.UserName,
			Role:     models.RoleUser,
			Text:     messageText,
		})

		if err != nil {
			log.Println(err)
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: messageText,
		})

		resp, err := h.clientOpenAI.CreateChatCompletion(
			context.Background(),
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

		_, err = h.storage.InsertMessage(models.Message{
			Id:         msg.MessageID,
			ChatId:     *user.ActiveChatId,
			ReplyToId:  &message.MessageID,
			UserId:     h.bot.Self.ID,
			Username:   h.bot.Self.UserName,
			Role:       models.RoleAssistant,
			Text:       responseText,
			Additional: resp,
		})
	}

	_, err := h.bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, loadingMsg.MessageID))

	if err != nil {
		log.Println(err)
	}
}

func (h *UpdateHandler) extractVoiceText(message *tgbotapi.Message) string {
	fileId := ""
	if message.Voice != nil {
		fileId = message.Voice.FileID
	} else if message.Audio != nil {
		fileId = message.Audio.FileID
	} else {
		return ""
	}

	file, err := h.bot.GetFile(tgbotapi.FileConfig{
		FileID: fileId,
	})

	fileUrl := file.Link(h.telegramToken)
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

	resp, err := h.clientOpenAI.CreateTranscription(
		context.Background(),
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

func (h *UpdateHandler) handleStartCommand(message *tgbotapi.Message) {
	user := h.getCurrentUser(message)
	msg := h.newSystemMessage(message.Chat.ID, localization.GetLocalizedText(user.Lang, localization.WelcomeMessage))
	msg.ReplyToMessageID = message.MessageID
	h.bot.Send(msg)
}

func (h *UpdateHandler) handleNewCommand(message *tgbotapi.Message) {
	user := h.getCurrentUser(message)

	user.ActiveChatId = nil

	h.storage.UpdateUser(&user)

	msg := h.newSystemMessage(message.Chat.ID, "New context started")
	msg.ReplyToMessageID = message.MessageID

	h.bot.Send(msg)
	h.bot.Send(tgbotapi.UnpinAllChatMessagesConfig{
		ChatID: message.Chat.ID,
	})
}
func (h *UpdateHandler) handleImageCommand(message *tgbotapi.Message) {
	prompt := strings.TrimSpace(strings.ReplaceAll(message.Text, "/image", ""))

	if len(prompt) < 3 {
		msg := h.newSystemMessage(message.Chat.ID, "Please write more information")
		msg.ReplyToMessageID = message.MessageID
		h.bot.Send(msg)
	} else {
		resp, err := h.clientOpenAI.CreateImage(
			context.Background(),
			openai.ImageRequest{
				Prompt:         prompt,
				Size:           openai.CreateImageSize256x256,
				ResponseFormat: openai.CreateImageResponseFormatURL,
				N:              2,
				User:           strconv.FormatInt(message.From.ID, 10),
			},
		)

		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)

			msg := h.newSystemMessage(message.Chat.ID, "Failed, try again")
			msg.ReplyToMessageID = message.MessageID
			h.bot.Send(msg)
			return
		}

		files := make([]interface{}, len(resp.Data))
		for i, url := range resp.Data {
			files[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url.URL))
		}

		mediaGroup := tgbotapi.NewMediaGroup(message.Chat.ID, files)

		h.bot.SendMediaGroup(mediaGroup)
	}
}

func (h *UpdateHandler) handleChatsCommand(message *tgbotapi.Message) {
	chats, _ := h.storage.ListUserChats(h.getCurrentUser(message).Id)

	if len(chats) == 0 {
		h.newSystemReply(message, "No chats found")
		return
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, len(chats))

	for i, chat := range chats {
		chatIdHex := chat.Id.Hex()
		buttons[i] = []tgbotapi.InlineKeyboardButton{{
			Text:         chat.Title,
			CallbackData: &chatIdHex,
		}}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Click on chat you want to switch")
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg.ReplyToMessageID = message.MessageID

	_, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func (h *UpdateHandler) handleHistoryCommand(message *tgbotapi.Message) {
	user := h.getCurrentUser(message)

	if user.ActiveChatId == nil {
		h.newSystemReply(message, "There is no active chat")

		return
	}

	var limit int64 = 10
	messages, _ := h.storage.ListChatMessages(*user.ActiveChatId, &limit)
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

func (h *UpdateHandler) handleUnknownCommand(message *tgbotapi.Message) {
	msg := h.newSystemMessage(message.Chat.ID, "Unknown command")
	msg.ReplyToMessageID = message.MessageID
	_, err := h.bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}
