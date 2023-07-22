package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"ibuddy_bot/database"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const (
	telegramTokenEnvName = "TELEGRAM_TOKEN"
	chatgptKeyEnvName    = "CHATGPT_KEY"
	mongoDbUri           = "MONGODB_URI"
	debugEnvName         = "DEBUG"
	adminUserEnvName     = "ADMIN_USER"

	maxChatMessages = 30
)

var (
	bot           *tgbotapi.BotAPI
	client        *openai.Client
	db            *database.Database
	telegramToken string
	adminUser     string
)

func main() {
	err := godotenv.Load()
	if err != nil {
		//log.Fatal("Error loading .env file")
	}

	telegramToken = os.Getenv(telegramTokenEnvName)
	chatgptKey := os.Getenv(chatgptKeyEnvName)
	mongodbUri := os.Getenv(mongoDbUri)
	debug := os.Getenv(debugEnvName) == "true"
	adminUser = os.Getenv(adminUserEnvName)
	client = openai.NewClient(chatgptKey)
	bot, err = tgbotapi.NewBotAPI(telegramToken)

	if err != nil {
		panic(err)
	}

	db, err = database.New(context.Background(), mongodbUri)

	if err != nil {
		panic(err)
	}

	db.Init()
	defer db.Disconnect(context.Background())

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	for update := range bot.GetUpdatesChan(u) {
		if update.Message != nil {
			go handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(update.CallbackQuery)
		} else {
			log.Println("Unknown update!")
		}
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
	//time for cleanup before exit
	fmt.Println("Adios!")
}

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	switch {
	case primitive.IsValidObjectID(callbackQuery.Data):
		handleChatSwitchButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_chats"):
		handleUserChatsButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_chat"):
		handleUserChatButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_ban"):
		handleUserBanButton(callbackQuery)
	case strings.HasPrefix(callbackQuery.Data, "user_unban"):
		handleUserUnbanButton(callbackQuery)

	}
}

func getUserMention(userId int64, username string) string {
	if username == "" {
		username = "user"
	}
	return fmt.Sprintf("[%s](tg://user?id=%d)", username, userId)
}

func handleChatSwitchButton(callbackQuery *tgbotapi.CallbackQuery) {
	chatId, err := primitive.ObjectIDFromHex(callbackQuery.Data)

	if err != nil {
		log.Println(err)

		return
	}

	chat, err := db.GetChatById(chatId)

	callback := tgbotapi.NewCallback(callbackQuery.ID, chat.Title)

	if _, err := bot.Request(callback); err != nil {
		panic(err)
	}

	user := getCurrentUser(callbackQuery.Message)
	msg, err := bot.Send(
		newSystemMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("Active chat: %s", chat.Title)),
	)

	if err != nil {
		log.Println(err)

		return
	}

	changeUserActiveChat(&user, chatId)
	pinMessage(msg.Chat.ID, msg.MessageID)
}

func pinMessage(chatId int64, messageId int) {
	_, err := bot.Send(tgbotapi.PinChatMessageConfig{
		ChatID:              chatId,
		MessageID:           messageId,
		DisableNotification: true,
	})

	if err != nil {
		log.Println(err)
	}
}

func handleMessage(message *tgbotapi.Message) {
	if message.PinnedMessage != nil {
		log.Printf("Pinned message: %s", message.PinnedMessage.Text)
		return
	}

	user := getCurrentUser(message)

	msg := newSystemMessage(message.Chat.ID, getLocalizedText(user.Lang, TextLoading))
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
	}
	msg.ReplyToMessageID = message.MessageID

	loadingMsg, _ := bot.Send(msg)

	if user.IsBanned() {
		msg := newSystemMessage(message.Chat.ID, getLocalizedText(user.Lang, UserBanned, *user.BanReason))
		msg.ReplyToMessageID = message.MessageID

		_, err := bot.Send(msg)

		if err != nil {
			log.Println(err)
		}

		return
	}

	messageText := strings.TrimSpace(message.Text)
	if message.IsCommand() {
		handleCommandMessage(message)
	} else {
		if len(messageText) < 2 {
			msg := newSystemMessage(message.Chat.ID, getLocalizedText(user.Lang, TooShortMessage))
			msg.ReplyToMessageID = message.MessageID
			bot.Send(msg)

			return
		}

		if user.ActiveChatId == nil {
			res, err := db.CreateChat(database.Chat{
				UserId:   user.Id,
				Username: user.Username,
				Title:    messageText,
			})

			if err != nil {
				log.Println(err)
			} else {
				v, _ := res.InsertedID.(primitive.ObjectID)
				changeUserActiveChat(&user, v)
				pinMessage(message.Chat.ID, message.MessageID)
			}
		}

		isVoiceText := false
		voiceText := extractVoiceText(message)

		if voiceText != "" {
			isVoiceText = true
			messageText = voiceText
		}

		var limit int64 = maxChatMessages
		activeChatMessages, _ := db.ListChatMessages(*user.ActiveChatId, &limit)
		ReverseSlice(activeChatMessages)

		messages := make([]openai.ChatCompletionMessage, len(activeChatMessages))

		for i, msg := range activeChatMessages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Text,
			}
		}

		_, err := db.InsertMessage(database.Message{
			Id:       message.MessageID,
			ChatId:   *user.ActiveChatId,
			UserId:   message.From.ID,
			Username: message.From.UserName,
			Role:     database.RoleUser,
			Text:     messageText,
		})

		if err != nil {
			log.Println(err)
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: messageText,
		})

		resp, err := client.CreateChatCompletion(
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

			newSystemReply(message, "Failed, try again")

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

		msg, err := newReplyWithFallback(message, responseText, tgbotapi.ModeMarkdownV2)

		if err != nil {
			log.Println(err)
		}

		_, err = db.InsertMessage(database.Message{
			Id:         msg.MessageID,
			ChatId:     *user.ActiveChatId,
			ReplyToId:  &message.MessageID,
			UserId:     bot.Self.ID,
			Username:   bot.Self.UserName,
			Role:       database.RoleAssistant,
			Text:       responseText,
			Additional: resp,
		})
	}

	_, err := bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, loadingMsg.MessageID))

	if err != nil {
		log.Println(err)
	}
}

func newReplyWithFallback(message *tgbotapi.Message, responseText string, parseMode string) (tgbotapi.Message, error) {
	if len(responseText) > 4096 {
		responseText = responseText[:4096]
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = parseMode
	msg.ReplyToMessageID = message.MessageID

	res, err := bot.Send(msg)

	if err != nil {
		log.Println(err)

		if strings.Contains(err.Error(), "can't parse entities") {
			if parseMode == tgbotapi.ModeMarkdownV2 {
				return newReplyWithFallback(
					message,
					responseText,
					tgbotapi.ModeMarkdown,
				)
			} else {
				return newReplyWithFallback(
					message,
					responseText,
					"",
				)
			}
		}
	}

	return res, err
}

func changeUserActiveChat(user *database.User, chatId primitive.ObjectID) {
	user.ActiveChatId = &chatId
	db.UpdateUser(user)
}

func handleCommandMessage(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		handleStartCommand(message)
	case "image":
		handleImageCommand(message)
	case "new":
		handleNewCommand(message)
	case "chats":
		handleChatsCommand(message)
	case "history":
		handleHistoryCommand(message)
	case "admin":
		handleAdminCommand(message)
	default:
		handleUnknownCommand(message)
	}
}

func handleUnknownCommand(message *tgbotapi.Message) {
	msg := newSystemMessage(message.Chat.ID, "Unknown command")
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func handleNewCommand(message *tgbotapi.Message) {
	user := getCurrentUser(message)

	user.ActiveChatId = nil

	db.UpdateUser(&user)

	msg := newSystemMessage(message.Chat.ID, "New context started")
	msg.ReplyToMessageID = message.MessageID

	bot.Send(msg)
	bot.Send(tgbotapi.UnpinAllChatMessagesConfig{
		ChatID: message.Chat.ID,
	})
}

func handleStartCommand(message *tgbotapi.Message) {
	user := getCurrentUser(message)
	msg := newSystemMessage(message.Chat.ID, getLocalizedText(user.Lang, WelcomeMessage))
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func getCurrentUser(message *tgbotapi.Message) database.User {
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

	user, _ := db.GetOrCreateUser(userId, &database.User{
		Id:       userId,
		Username: username,
	})

	user.Lang = lang

	return user
}

func handleChatsCommand(message *tgbotapi.Message) {
	chats, _ := db.ListUserChats(getCurrentUser(message).Id)

	if len(chats) == 0 {
		newSystemReply(message, "No chats found")
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

	_, err := bot.Send(msg)

	if err != nil {
		log.Println(err)
	}
}

func handleHistoryCommand(message *tgbotapi.Message) {
	user := getCurrentUser(message)

	if user.ActiveChatId == nil {
		newSystemReply(message, "There is no active chat")

		return
	}

	var limit int64 = 10
	messages, _ := db.ListChatMessages(*user.ActiveChatId, &limit)
	ReverseSlice(messages)

	items := make([]string, len(messages))
	for i, msg := range messages {
		if msg.Role == database.RoleUser {
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
	_, err := newReplyWithFallback(message, text, tgbotapi.ModeMarkdownV2)

	if err != nil {
		log.Println(err)
	}
}

func newSystemReply(message *tgbotapi.Message, text string) (tgbotapi.Message, error) {
	msgConfig := newSystemMessage(message.Chat.ID, text)

	return bot.Send(msgConfig)
}

func newSystemMessage(chatId int64, text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("`%s`", text))
	msg.ParseMode = tgbotapi.ModeMarkdown

	return msg
}

func extractVoiceText(message *tgbotapi.Message) string {
	fileId := ""
	if message.Voice != nil {
		fileId = message.Voice.FileID
	} else if message.Audio != nil {
		fileId = message.Audio.FileID
	} else {
		return ""
	}

	file, err := bot.GetFile(tgbotapi.FileConfig{
		FileID: fileId,
	})

	fileUrl := file.Link(telegramToken)
	localFile, err := downloadFileByUrl(fileUrl)
	defer os.Remove(localFile.Name())
	mp3FilePath, err := convertOggToMp3(localFile.Name())
	defer os.Remove(mp3FilePath)

	if err != nil {
		fmt.Printf("error: %v\n", err)

		msg := newSystemMessage(message.Chat.ID, "Failed, try again")
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return ""
	}

	resp, err := client.CreateTranscription(
		context.Background(),
		openai.AudioRequest{
			Model:    openai.Whisper1,
			FilePath: mp3FilePath,
		},
	)

	if err != nil {
		fmt.Printf("error: %v\n", err)

		msg := newSystemMessage(message.Chat.ID, "Failed, try again")
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
		return ""
	}

	return resp.Text
}

func handleImageCommand(message *tgbotapi.Message) {
	prompt := strings.TrimSpace(strings.ReplaceAll(message.Text, "/image", ""))

	if len(prompt) < 3 {
		msg := newSystemMessage(message.Chat.ID, "Please write more information")
		msg.ReplyToMessageID = message.MessageID
		bot.Send(msg)
	} else {
		resp, err := client.CreateImage(
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

			msg := newSystemMessage(message.Chat.ID, "Failed, try again")
			msg.ReplyToMessageID = message.MessageID
			bot.Send(msg)
			return
		}

		files := make([]interface{}, len(resp.Data))
		for i, url := range resp.Data {
			files[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(url.URL))
		}

		mediaGroup := tgbotapi.NewMediaGroup(message.Chat.ID, files)

		bot.SendMediaGroup(mediaGroup)
	}
}
