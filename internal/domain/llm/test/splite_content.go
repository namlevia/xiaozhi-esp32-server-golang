package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	// Định nghĩa tập dấu câu
	punctuationMap = map[rune]bool{
		'。':  true,
		'？':  true,
		'！':  true,
		'；':  true,
		'：':  true,
		'\n': true,
		'.':  true,
		'?':  true,
		'!':  true,
		';':  true,
		':':  true,
	}

	// Object pool để tái sử dụng
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// Slice pool để lưu kết quả
	runeSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]rune, 0, 1024)
			return &slice
		},
	}

	// Precompile regex
	numberPrefixRegex = regexp.MustCompile(`(?m)^[\s]*\d{1,3}\.$`)
)

// Dùng kiểm tra ký tự nhanh thay cho regex
func isNumberPrefix(text []rune, pos int) bool {
	if pos <= 0 || text[pos] != '.' {
		return false
	}

	// Tìm ngược về đầu dòng hoặc ký tự xuống dòng
	start := pos - 1
	digitCount := 0
	foundDigit := false

	// Bỏ qua ký tự trắng trước dấu chấm
	for start >= 0 && (text[start] == ' ' || text[start] == '\t') {
		start--
	}

	// Đếm chữ số
	for start >= 0 && text[start] >= '0' && text[start] <= '9' {
		digitCount++
		foundDigit = true
		if digitCount > 3 { // Hơn 3 chữ số không phải số thứ tự hợp lệ
			return false
		}
		start--
	}

	// Kiểm tra trước chữ số là ký tự trắng hoặc đầu dòng
	if start >= 0 && text[start] != ' ' && text[start] != '\t' && text[start] != '\n' {
		return false
	}

	return foundDigit
}

// Xóa ký tự trắng đầu/cuối
func trimSpaceRunes(text []rune) []rune {
	start, end := 0, len(text)-1

	for start <= end && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n') {
		start++
	}

	for end >= start && (text[end] == ' ' || text[end] == '\t' || text[end] == '\n') {
		end--
	}

	if start > end {
		return nil
	}
	return text[start : end+1]
}

func findLastPunctuation(text []rune) int {
	// Tìm dấu câu cuối cùng từ sau ra trước
	lastPos := -1
	for i := len(text) - 1; i >= 0; i-- {
		// Kiểm tra có phải dấu câu hay không
		if punctuationMap[text[i]] {
			// Nếu là dấu chấm, kiểm tra có phải một phần của số thứ tự hay không
			if text[i] == '.' && isNumberPrefix(text, i) {
				continue
			}
			return i
		}
	}
	return lastPos
}

func findNextSplitPoint(text []rune, startPos int, maxLen int) int {
	// Tính vị trí kết thúc tìm kiếm
	endPos := startPos + maxLen
	if endPos > len(text) {
		endPos = len(text)
	}

	// Tìm từ trước ra sau
	for i := startPos; i < endPos; i++ {
		// Kiểm tra có phải ký tự xuống dòng hay không, đồng thời kiểm tra dòng tiếp theo có phải số thứ tự hay không
		if text[i] == '\n' {
			nextPos := i + 1
			// Bỏ qua ký tự trắng
			for nextPos < endPos && (text[nextPos] == ' ' || text[nextPos] == '\t') {
				nextPos++
			}
			// Kiểm tra có phải bắt đầu số thứ tự hay không
			if nextPos < endPos-2 && text[nextPos] >= '0' && text[nextPos] <= '9' {
				return i
			}
			continue
		}

		// Dùng map kiểm tra có phải dấu câu hay không
		if punctuationMap[text[i]] {
			return i
		}
	}

	// Nếu không tìm thấy trong phạm vi maxLen, thử tìm trong phạm vi lớn hơn
	if endPos < len(text) {
		for i := endPos; i < len(text); i++ {
			if text[i] == '\n' || punctuationMap[text[i]] {
				return i
			}
		}
	}

	return -1
}

func extractSmartSentences(text string, minLen, maxLen int) (sentences []string, remaining string) {
	// Preallocate dung lượng slice hợp lý
	estimatedCount := len(text) / 50
	if estimatedCount < 10 {
		estimatedCount = 10
	}
	sentences = make([]string, 0, estimatedCount)

	// Chuyển thành slice rune một lần
	currentRunes := []rune(text)
	startPos := 0

	// Lấy object tái sử dụng từ object pool
	builder := builderPool.Get().(*strings.Builder)
	defer builderPool.Put(builder)
	builder.Grow(maxLen * 2)

	// Lấy slice rune tạm
	tempRunesPtr := runeSlicePool.Get().(*[]rune)
	tempRunes := (*tempRunesPtr)[:0]
	defer runeSlicePool.Put(tempRunesPtr)

	for startPos < len(currentRunes) {
		// Bỏ qua ký tự trắng ở đầu
		for startPos < len(currentRunes) && (currentRunes[startPos] == ' ' || currentRunes[startPos] == '\t' || currentRunes[startPos] == '\n') {
			startPos++
		}

		if startPos >= len(currentRunes) {
			break
		}

		// Tìm điểm tách tiếp theo
		splitPos := findNextSplitPoint(currentRunes, startPos, maxLen)
		if splitPos == -1 {
			// Không tìm thấy điểm tách, dùng text còn lại làm remaining
			segment := trimSpaceRunes(currentRunes[startPos:])
			if len(segment) > 0 {
				remaining = string(segment)
			}
			break
		}

		// Trích đoạn hiện tại
		builder.Reset()
		tempRunes = tempRunes[:0]

		// Thu thập và xử lý đoạn hiện tại
		segment := trimSpaceRunes(currentRunes[startPos : splitPos+1])

		// Kiểm tra đoạn có đạt độ dài tối thiểu và kết thúc bằng dấu câu hay không
		if len(segment) >= minLen && punctuationMap[segment[len(segment)-1]] {
			sentences = append(sentences, string(segment))
		} else {
			// Nếu không thỏa điều kiện thì thêm vào remaining
			if len(segment) > 0 {
				if len(remaining) > 0 {
					remaining += " "
				}
				remaining += string(segment)
			}
		}

		startPos = splitPos + 1
	}

	return sentences, remaining
}

func main() {
	text := `Hừ, em biết anh lại đang qua loa với em! Lần nào hỏi anh cũng nói không, có phải anh không thích em nữa không? Em giận rồi đó! Em không chơi với anh nữa! Trừ khi... anh hứa lát nữa đưa em đi chợ đêm ăn đậu hũ ngọt nhé~ còn phải nắm tay em đi dạo phố, cả đường phải chọc em cười, làm em vui bay lên trời! Nếu không em thật sự sẽ không thèm để ý anh nữa đâu~`
	sentences, remaining := extractSmartSentences(text, 3, 200)
	for i, sentence := range sentences {
		fmt.Printf("\nCâu %d:\n%s\n", i+1, sentence)
	}
	if remaining != "" {
		fmt.Printf("\nCòn lại:\n%s\n", remaining)
	}
}
