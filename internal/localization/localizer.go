package localization

import "fmt"

const (
	TextLoading     = "loading"
	UserBanned      = "userBanned"
	TooShortMessage = "tooShortMessage"
	WelcomeMessage  = "welcomeMessage"
)

var (
	messages = map[string]map[string]string{
		"en": {
			TextLoading:     "Loading...",
			UserBanned:      "You're banned: %s",
			TooShortMessage: "Too short message",
			WelcomeMessage:  "Welcome!\nSend message to start conversation\nSend `/new` to clear current thread\nSend `/image {description}` to generate images",
		},
		"ru": {
			TextLoading:     "Идет загрузка...",
			UserBanned:      "Вы были забанены: %s",
			TooShortMessage: "Слишком короткое сообщение",
			WelcomeMessage:  "Добро пожаловать! ",
		},
	}
)

func GetLocalizedText(lang string, textId string, args ...interface{}) string {
	if _, ok := messages[lang][textId]; !ok {
		lang = "en"
	}

	message, ok := messages[lang][textId]

	if !ok {
		return textId
	}

	if len(args) == 0 {
		return message
	}

	return fmt.Sprintf(message, args)
}
