package chat

import (
	"strings"
	"unicode"

	"github.com/spf13/viper"
)

// removePunctuation xóa dấu câu khỏi text
func removePunctuation(text string) string {
	// Tạo string builder
	var builder strings.Builder
	builder.Grow(len(text))

	for _, r := range text {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// isWakeupWord kiểm tra text có phải wakeup word không
func isWakeupWord(text string) bool {
	wakeupWords := viper.GetStringSlice("wakeup_words")
	for _, word := range wakeupWords {
		if text == word {
			return true
		}
	}
	return false
}
