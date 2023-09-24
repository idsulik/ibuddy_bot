package tgbotclient

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ibuddy_bot/internal/localization"
)

type TgBotClient struct {
	*tgbotapi.BotAPI
}

func NewTgBotClient(botToken string, debug bool) (*TgBotClient, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)

	if err != nil {
		log.Panic(err)
		return nil, err
	}

	bot.Debug = debug

	return &TgBotClient{bot}, nil
}

func (h *TgBotClient) DeleteMessage(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	return h.BotAPI.Send(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
}

func (h *TgBotClient) SendLoadingReply(msg *tgbotapi.Message, userLang string) (tgbotapi.Message, error) {
	loadingMsgConfig := h.NewSystemMessage(
		msg.Chat.ID,
		localization.GetLocalizedText(userLang, localization.TextLoading),
	)
	loadingMsgConfig.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
	}
	loadingMsgConfig.ReplyToMessageID = msg.MessageID

	return h.BotAPI.Send(loadingMsgConfig)
}

func (h *TgBotClient) NewSystemReply(message *tgbotapi.Message, text string) (tgbotapi.Message, error) {
	msgConfig := h.NewSystemMessage(message.Chat.ID, text)

	return h.Send(msgConfig)
}

func (h *TgBotClient) NewSystemMessage(chatId int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("`%s`", text))
	msg.ParseMode = tgbotapi.ModeMarkdown

	return msg
}

func (h *TgBotClient) PinMessage(chatId int64, messageId int) (tgbotapi.Message, error) {
	msg, err := h.Send(
		tgbotapi.PinChatMessageConfig{
			ChatID:              chatId,
			MessageID:           messageId,
			DisableNotification: true,
		},
	)

	return msg, err
}

func (h *TgBotClient) NewReplyWithFallback(
	message *tgbotapi.Message,
	responseText string,
	parseMode string,
) (tgbotapi.Message, error) {
	if len(responseText) > 4096 {
		responseText = responseText[:4096]
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = parseMode
	msg.ReplyToMessageID = message.MessageID

	res, err := h.Send(msg)

	if err != nil {
		log.Println(err)

		if strings.Contains(err.Error(), "can't parse entities") {
			if parseMode == tgbotapi.ModeMarkdownV2 {
				return h.NewReplyWithFallback(
					message,
					responseText,
					tgbotapi.ModeMarkdown,
				)
			} else {
				return h.NewReplyWithFallback(
					message,
					responseText,
					"",
				)
			}
		}
	}

	return res, err
}
