package main

import (
	"fmt"
)

func containsRune(slice []rune, target rune) bool {
	for _, r := range slice {
		if r == target {
			return true
		}
	}
	return false
}

func extractSmartSentences(text string, minLen, maxLen int) (sentences []string, remaining string) {
	// Tập dấu phân tách hợp lệ, có thể mở rộng tùy chỉnh.
	splitTokens := []rune{'。', '！', '？', '；', '\n', '.', '!', '?', ';'}

	current := []rune(text)
	for len(current) >= minLen {
		// Tính kích thước cửa sổ hiện tại
		windowSize := maxLen
		if windowSize > len(current) {
			windowSize = len(current)
		}

		// Tìm điểm tách trong cửa sổ hợp lệ
		splitPos := -1
		for i := windowSize - 1; i >= minLen-1; i-- {
			if containsRune(splitTokens, current[i]) {
				splitPos = i
				break
			}
		}

		if splitPos == -1 {
			break // Không tìm thấy điểm tách hợp lệ
		}

		// Tách và lưu câu hợp lệ
		sentences = append(sentences, string(current[:splitPos+1]))
		current = current[splitPos+1:]
	}

	return
}

func main() {
	text := "Chào mọi người! Hôm nay thời tiết đẹp. Chúng ta cùng học xử lý ngôn ngữ tự nhiên. Ví dụ này minh họa chức năng tách văn bản."
	sentences, remaining := extractSmartSentences(text, 3, 20)
	fmt.Println(sentences)
	fmt.Println(remaining)
}
