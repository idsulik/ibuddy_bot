package util

import (
	"fmt"
)

func GetUserMention(userId int64, username string) string {
	if username == "" {
		username = "user"
	}
	return fmt.Sprintf("[%s](tg://user?id=%d)", username, userId)
}

func ReverseSlice[T any](items []T) {
	for i := 0; i < len(items)/2; i++ {
		j := len(items) - i - 1
		items[i], items[j] = items[j], items[i]
	}
}
