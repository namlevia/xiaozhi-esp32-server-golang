package controllers

import "strings"

// VoiceInfo mô tả một giọng TTS Qwen.
type VoiceInfo struct {
	Value       string   `json:"value"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
}

// ModelVoiceMap ánh xạ họ model sang danh sách giọng được hỗ trợ.
var ModelVoiceMap = map[string][]VoiceInfo{
	"qwen3-tts-flash": {
		{Value: "Cherry", Label: "Cherry", Description: "Giọng nữ tươi sáng, tích cực và tự nhiên"},
		{Value: "Serena", Label: "Serena", Description: "Giọng nữ dịu dàng"},
		{Value: "Ethan", Label: "Ethan", Description: "Giọng nam phổ thông chuẩn, ấm áp và giàu năng lượng"},
		{Value: "Chelsie", Label: "Chelsie", Description: "Giọng nữ phong cách nhân vật ảo"},
		{Value: "Momo", Label: "Momo", Description: "Giọng nữ nhí nhảnh, vui vẻ"},
		{Value: "Vivian", Label: "Vivian", Description: "Giọng nữ cá tính, dễ thương"},
		{Value: "Moon", Label: "Moon", Description: "Giọng nam phóng khoáng, tự tin"},
		{Value: "Maia", Label: "Maia", Description: "Giọng nữ tri thức và dịu dàng"},
		{Value: "Kai", Label: "Kai", Description: "Giọng nam êm tai, thư giãn"},
		{Value: "Nofish", Label: "Nofish", Description: "Giọng nam cá tính, chất thiết kế"},
		{Value: "Bella", Label: "Bella", Description: "Giọng bé gái dễ thương"},
		{Value: "Jennifer", Label: "Jennifer", Description: "Giọng nữ tiếng Anh Mỹ chất lượng thương hiệu"},
		{Value: "Ryan", Label: "Ryan", Description: "Giọng nam giàu nhịp điệu và kịch tính"},
		{Value: "Katerina", Label: "Katerina", Description: "Giọng nữ trưởng thành, giàu âm sắc"},
		{Value: "Aiden", Label: "Aiden", Description: "Giọng nam tiếng Anh Mỹ trẻ trung"},
		{Value: "Eldric Sage", Label: "Eldric Sage", Description: "Giọng nam lớn tuổi, trầm ổn và thông thái"},
		{Value: "Mia", Label: "Mia", Description: "Giọng nữ ngoan hiền, nhẹ nhàng"},
		{Value: "Mochi", Label: "Mochi", Description: "Giọng thiếu niên thông minh, lanh lợi"},
		{Value: "Bellona", Label: "Bellona", Description: "Giọng nữ vang, rõ chữ và sống động"},
		{Value: "Vincent", Label: "Vincent", Description: "Giọng nam khàn, giàu chất giang hồ"},
		{Value: "Bunny", Label: "Bunny", Description: "Giọng bé gái cực kỳ dễ thương"},
		{Value: "Neil", Label: "Neil", Description: "Giọng nam dẫn tin chuyên nghiệp"},
		{Value: "Elias", Label: "Elias", Description: "Giọng giảng viên nghiêm túc, giàu tự sự"},
		{Value: "Arthur", Label: "Arthur", Description: "Giọng nam mộc mạc, từng trải"},
		{Value: "Nini", Label: "Nini", Description: "Giọng nữ mềm mại, gần gũi"},
		{Value: "Ebona", Label: "Ebona", Description: "Giọng nữ lớn tuổi hơi rùng rợn"},
		{Value: "Seren", Label: "Seren", Description: "Giọng nữ ôn hòa, thư giãn, hỗ trợ ngủ"},
		{Value: "Pip", Label: "Pip", Description: "Giọng bé trai tinh nghịch"},
		{Value: "Stella", Label: "Stella", Description: "Giọng nữ ngọt ngào, giàu chính nghĩa"},
		{Value: "Bodega", Label: "Bodega", Description: "Giọng nam Tây Ban Nha nhiệt tình"},
		{Value: "Sonrisa", Label: "Sonrisa", Description: "Giọng nữ Latin nhiệt tình, vui vẻ"},
		{Value: "Alek", Label: "Alek", Description: "Giọng nam Nga lạnh lùng nhưng ấm áp"},
		{Value: "Dolce", Label: "Dolce", Description: "Giọng nam Ý thư thái"},
		{Value: "Sohee", Label: "Sohee", Description: "Giọng nữ Hàn dịu dàng, giàu cảm xúc"},
		{Value: "Ono Anna", Label: "Ono Anna", Description: "Giọng nữ Nhật tinh nghịch"},
		{Value: "Lenn", Label: "Lenn", Description: "Giọng nam Đức lý trí, có nét nổi loạn"},
		{Value: "Emilien", Label: "Emilien", Description: "Giọng nam Pháp lãng mạn"},
		{Value: "Andre", Label: "Andre", Description: "Giọng nam trầm, tự nhiên và điềm đạm"},
		{Value: "Radio Gol", Label: "Radio Gol", Description: "Giọng bình luận bóng đá giàu chất thơ"},
		{Value: "Jada", Label: "Jada", Description: "Giọng nữ Thượng Hải năng động"},
		{Value: "Dylan", Label: "Dylan", Description: "Giọng nam Bắc Kinh trẻ trung"},
		{Value: "Li", Label: "Li", Description: "Giọng nam giáo viên yoga kiên nhẫn"},
		{Value: "Marcus", Label: "Marcus", Description: "Giọng nam Thiểm Tây đậm chất địa phương"},
		{Value: "Roy", Label: "Roy", Description: "Giọng nam Mân Nam hài hước, thẳng thắn"},
		{Value: "Peter", Label: "Peter", Description: "Giọng nam Thiên Tân phong cách tấu hài"},
		{Value: "Sunny", Label: "Sunny", Description: "Giọng nữ Tứ Xuyên ngọt ngào"},
		{Value: "Eric", Label: "Eric", Description: "Giọng nam Thành Đô đời thường, linh hoạt"},
		{Value: "Rocky", Label: "Rocky", Description: "Giọng nam Quảng Đông hài hước"},
		{Value: "Kiki", Label: "Kiki", Description: "Giọng nữ Quảng Đông ngọt ngào"},
	},

	"qwen-tts": {
		{Value: "Cherry", Label: "Cherry", Description: "Giọng nữ tươi sáng, tích cực và tự nhiên"},
		{Value: "Serena", Label: "Serena", Description: "Giọng nữ dịu dàng"},
		{Value: "Ethan", Label: "Ethan", Description: "Giọng nam phổ thông chuẩn, ấm áp và giàu năng lượng"},
		{Value: "Chelsie", Label: "Chelsie", Description: "Giọng nữ phong cách nhân vật ảo"},
		{Value: "Momo", Label: "Momo", Description: "Giọng nữ nhí nhảnh, vui vẻ"},
	},
}

// normalizeModel chuẩn hóa tên model cụ thể thành khóa họ model.
// Ví dụ: qwen3-tts-flash-2025-11-27 -> qwen3-tts-flash
//
//	qwen-tts-2025-05-22       -> qwen-tts
func normalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if strings.HasPrefix(model, "qwen3-tts-flash") {
		return "qwen3-tts-flash"
	}
	if strings.HasPrefix(model, "qwen-tts") {
		return "qwen-tts"
	}
	return model
}

// GetVoicesByModel lấy danh sách giọng được hỗ trợ theo tên model.
func GetVoicesByModel(model string) []VoiceInfo {
	key := normalizeModel(model)
	if voices, ok := ModelVoiceMap[key]; ok {
		return voices
	}
	return nil
}

// IsVoiceSupported kiểm tra model chỉ định có hỗ trợ giọng đã chọn hay không.
func IsVoiceSupported(model, voice string) bool {
	if voice == "" {
		return false
	}
	for _, v := range GetVoicesByModel(model) {
		if v.Value == voice {
			return true
		}
	}
	return false
}
