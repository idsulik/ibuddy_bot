package main

import "ibuddy_bot/database"

func ReverseSlice(s []database.Message) {
	for i := 0; i < len(s)/2; i++ {
		j := len(s) - i - 1
		s[i], s[j] = s[j], s[i]
	}
}
