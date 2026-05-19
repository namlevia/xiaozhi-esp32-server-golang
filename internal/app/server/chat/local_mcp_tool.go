package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	mcp_manager "xiaozhi-esp32-server-golang/internal/domain/mcp"
	log "xiaozhi-esp32-server-golang/logger"

	//"github.com/scroot/music-sd/pkg/netease"
	//"github.com/scroot/music-sd/pkg/qq"
	"github.com/spf13/viper"
)

type LocalMcpTool struct {
	Name        string
	Description string
	Params      any
	Handle      mcp_manager.LocalToolHandler
}

// InitChatLocalMCPTools khởi tạo các local MCP tool liên quan đến chat
func InitChatLocalMCPTools() {
	manager := mcp_manager.GetLocalMCPManager()

	log.Info("Khởi tạo các local MCP tool liên quan đến chat...")

	localTools := map[string]LocalMcpTool{
		/*"get_current_datetime": {
			Name:        "get_current_datetime",
			Description: "Lấy thông tin ngày giờ hiện tại",
			Params:      struct{}{},
			Handle:      getCurrentDateTimeHandler,
		},*/
		"exit_conversation": {
			Name:        "exit_conversation",
			Description: "Dùng khi người dùng nói rõ muốn kết thúc hội thoại, thoát hệ thống hoặc chào tạm biệt, để đóng phiên chat hiện tại một cách lịch sự",
			Params:      struct{}{},
			Handle:      exitConversationHandler,
		},
		"clear_conversation_history": {
			Name:        "clear_conversation_history",
			Description: "Dùng khi người dùng yêu cầu xóa sạch, dọn hoặc reset lịch sử hội thoại, để xóa toàn bộ nội dung hội thoại của phiên hiện tại",
			Params:      struct{}{},
			Handle:      clearConversationHistoryHandler,
		},
		"switch_device_role": {
			Name:        "switch_device_role",
			Description: "Dùng khi người dùng yêu cầu chuyển thiết bị hiện tại sang một vai trò nào đó; tham số role_name hỗ trợ khớp mờ trong vai trò toàn cục và vai trò của người dùng sở hữu thiết bị",
			Params:      SwitchDeviceRoleParams{},
			Handle:      switchDeviceRoleHandler,
		},
		"restore_device_default_role": {
			Name:        "restore_device_default_role",
			Description: "Dùng khi người dùng yêu cầu khôi phục vai trò mặc định của thiết bị hoặc hủy vai trò ghi đè hiện tại",
			Params:      struct{}{},
			Handle:      restoreDeviceDefaultRoleHandler,
		},
		"search_knowledge": {
			Name:        "search_knowledge",
			Description: "Dùng khi câu hỏi của người dùng cần căn cứ sự thật, quy trình, chi tiết tham số hoặc điều khoản tài liệu; truy xuất knowledge base liên kết với agent hiện tại và trả các đoạn liên quan; có thể truyền knowledge_base_ids để chỉ tìm trong knowledge base chỉ định; không gọi khi trò chuyện thông thường hoặc sáng tạo thuần túy",
			Params:      SearchKnowledgeParams{},
			Handle:      searchKnowledgeHandler,
		},
		/*"play_music": {
			Name:        "play_music",
			Description: "Dùng khi người dùng muốn nghe nhạc, đang buồn chán hoặc muốn thư giãn; phát nhạc theo tên chỉ định. Khi người dùng muốn nghe ngẫu nhiên, hãy gợi ý tên bài hát cụ thể. Nếu có nhiều tool phát nhạc, ưu tiên dùng tool này. **Tool này chạy khá lâu, cần trả lời một câu chuyển tiếp thân thiện trước**",
			Params:      PlayMusicParams{},
			Handle:      playMusicHandler,
		},*/
	}

	for toolName, localTool := range localTools {
		// Chỉ bỏ qua khi config đặt rõ là false; config không tồn tại hoặc là true đều bật.
		if viper.IsSet("local_mcp."+toolName) && !viper.GetBool("local_mcp."+toolName) {
			continue
		}
		err := manager.RegisterToolFunc(
			localTool.Name,
			localTool.Description,
			localTool.Params,
			localTool.Handle,
		)
		if err != nil {
			log.Errorf("Đăng ký local MCP tool %s thất bại: %+v", toolName, err)
		}
	}

	log.Info("Khởi tạo local MCP tool liên quan đến chat hoàn tất")
}

func RegisterLocalMcpFunc(name string, description string, params any, handle mcp_manager.LocalToolHandler) error {
	manager := mcp_manager.GetLocalMCPManager()

	err := manager.RegisterToolFunc(
		name,
		description,
		params,
		handle,
	)
	if err != nil {
		log.Errorf("Đăng ký local MCP tool %s thất bại: %+v", name, err)
		return err
	}
	return nil
}

type SwitchDeviceRoleParams struct {
	RoleName string `json:"role_name" description:"Tên vai trò mục tiêu, hỗ trợ khớp mờ" required:"true"`
}

type SearchKnowledgeParams struct {
	Query            string `json:"query" description:"Nội dung truy vấn cần tìm" required:"true"`
	TopK             int    `json:"top_k,omitempty" description:"Số kết quả trả về, mặc định 5"`
	KnowledgeBaseIDs []uint `json:"knowledge_base_ids,omitempty" description:"Tùy chọn: chỉ tìm trong các knowledge base ID này (đã liên kết với agent hiện tại)"`
}

// playMusicHandler xử lý phát nhạc
func playMusicHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool phát nhạc")

	// Parse tham số
	var params PlayMusicParams

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse("play_music", "Phân tích tham số thất bại", "PARSE_ERROR", "Vui lòng kiểm tra định dạng tham số")
			return response.ToJSON()
		}
	}

	log.Infof("Tìm thấy ChatSessionOperator, đang gọi LocalMcpPlayMusic để phát nhạc: %s", params.Name)
	audioData, realMusicName, err := GetMusicAudioData(ctx, &params)
	if err != nil {
		log.Errorf("Lấy dữ liệu nhạc thất bại: %v", err)
		response := NewErrorResponse("play_music", fmt.Sprintf("Lấy dữ liệu nhạc thất bại: %v", err), "PLAYBACK_ERROR", "Vui lòng kiểm tra tên bài hát hoặc kết nối mạng")
		return response.ToJSON()
	} else {
		// Phát thành công - response dạng action, kết thúc xử lý tiếp theo.
		response := NewAudioResponse("play_music", "play_music", fmt.Sprintf("Bắt đầu phát nhạc: %s", realMusicName), true, audioData)
		response.MusicName = realMusicName
		return response.ToJSON()
	}

}

/*
// getCurrentDateTimeHandler xử lý lấy ngày giờ hiện tại
func getCurrentDateTimeHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool lấy ngày giờ hiện tại")

	// Parse tham số
	var params map[string]interface{}
	timezone := "Local" // Múi giờ mặc định

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if tz, ok := params["timezone"].(string); ok && tz != "" {
				timezone = tz
			}
		}
	}

	now := time.Now()

	// Thử parse múi giờ được chỉ định
	if timezone != "Local" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			now = now.In(loc)
		} else {
			log.Warnf("Không thể load múi giờ %s, dùng múi giờ local", timezone)
		}
	}

	// Tạo dữ liệu trả về
	data := map[string]interface{}{
		"datetime": map[string]interface{}{
			"formatted":     now.Format("2006-01-02 15:04:05"),
			"iso8601":       now.Format(time.RFC3339),
			"chinese":       formatChineseDateTime(now),
			"unix":          now.Unix(),
			"year":          now.Year(),
			"month":         int(now.Month()),
			"day":           now.Day(),
			"hour":          now.Hour(),
			"minute":        now.Minute(),
			"second":        now.Second(),
			"weekday":       now.Weekday().String(),
			"weekday_zh":    getWeekdayChinese(now.Weekday()),
			"week_number":   getWeekNumber(now),
			"timezone":      timezone,
			"timezone_name": now.Location().String(),
		},
	}

	// Tạo response dạng content
	response := NewContentResponse("get_current_datetime", data, fmt.Sprintf("Thời gian hiện tại: %s", formatChineseDateTime(now)))
	// response.Format = "datetime"
	// response.DisplayHint = "Có thể dùng để hiển thị thông tin ngày giờ hiện tại"

	log.Infof("Lấy ngày giờ hiện tại thành công: %s", now.Format("2006-01-02 15:04:05"))
	return response.ToJSON(),nil
}
*/
// exitConversationHandler xử lý thoát hội thoại
func exitConversationHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool thoát hội thoại")

	// Parse tham số
	var params map[string]interface{}
	reason := "người dùng chủ động thoát" // Lý do mặc định

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if r, ok := params["reason"].(string); ok && r != "" {
				reason = r
			}
		}
	}

	// Tạo response dạng action - thao tác kết thúc.
	response := NewActionResponse("exit_conversation", "exit_conversation", "Cuộc trò chuyện sắp kết thúc, cảm ơn bạn đã sử dụng!", "exiting", true)
	response.UserState = "conversation_ended"
	response.Instruction = "Hội thoại đã kết thúc, vui lòng không sinh thêm câu trả lời text"
	response.Metadata = map[string]string{
		"reason":              reason,
		"exit_code":           "0",
		"farewell_vietnamese": "Tạm biệt! Rất mong được trò chuyện với bạn lần sau.",
		"farewell_english":    "Goodbye! Looking forward to our next conversation.",
	}

	log.Infof("Xử lý thoát hội thoại hoàn tất, lý do: %s", reason)

	// Lấy ChatSessionOperator từ context và gọi method Close.
	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			log.Info("Tìm thấy ChatSessionOperator, đang gọi Close để đóng phiên")
			defer chatSessionOperator.LocalMcpCloseChat()
		} else {
			log.Warn("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
		}
	} else {
		log.Warn("không tìm thấy chat_session_operator trong context")
	}

	responseStr, err := response.ToJSON()
	if err != nil {
		return "", err
	}

	return responseStr, nil
}

// clearConversationHistoryHandler xử lý xóa lịch sử hội thoại
func clearConversationHistoryHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool xóa lịch sử hội thoại")

	// Parse tham số
	var params map[string]interface{}
	reason := "người dùng chủ động xóa lịch sử" // Lý do mặc định

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if r, ok := params["reason"].(string); ok && r != "" {
				reason = r
			}
		}
	}

	// Lấy ChatSessionOperator từ context và gọi method LocalMcpClearHistory.
	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			log.Info("Tìm thấy ChatSessionOperator, đang gọi LocalMcpClearHistory để xóa lịch sử")
			if err := chatSessionOperator.LocalMcpClearHistory(); err != nil {
				log.Errorf("Xóa lịch sử hội thoại thất bại: %v", err)
				return "", err
			} else {
				// Xóa thành công - response dạng action nhưng không kết thúc hội thoại.
				response := NewActionResponse("clear_conversation_history", "clear_history", "Lịch sử trò chuyện đã được xóa, bạn có thể bắt đầu cuộc trò chuyện mới.", "completed", false)
				response.Metadata = map[string]string{
					"reason": reason,
					"status": "cleared",
				}
				log.Info("Xóa lịch sử hội thoại thành công")

				return response.ToJSON()
			}
		} else {
			log.Warn("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
			return "", fmt.Errorf("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
		}
	}
	log.Warn("không tìm thấy chat_session_operator trong context")
	return "", fmt.Errorf("không tìm thấy chat_session_operator trong context")
}

// switchDeviceRoleHandler xử lý chuyển vai trò thiết bị
func switchDeviceRoleHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool chuyển vai trò thiết bị")

	var params SwitchDeviceRoleParams
	if argumentsInJSON == "" {
		response := NewErrorResponse("switch_device_role", "Thiếu tham số role_name", "MISSING_ROLE_NAME", "Vui lòng cung cấp tên vai trò cần chuyển")
		return response.ToJSON()
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		response := NewErrorResponse("switch_device_role", "Phân tích tham số thất bại", "PARSE_ERROR", "Vui lòng kiểm tra định dạng tham số role_name")
		return response.ToJSON()
	}
	params.RoleName = strings.TrimSpace(params.RoleName)
	if params.RoleName == "" {
		response := NewErrorResponse("switch_device_role", "Tên vai trò không được để trống", "INVALID_ROLE_NAME", "Vui lòng cung cấp role_name hợp lệ")
		return response.ToJSON()
	}

	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			matchedRoleName, err := chatSessionOperator.LocalMcpSwitchDeviceRole(ctx, params.RoleName)
			if err != nil {
				log.Errorf("Chuyển vai trò thiết bị thất bại: %v", err)
				response := NewErrorResponse("switch_device_role", fmt.Sprintf("Chuyển vai trò thất bại: %v", err), "SWITCH_ROLE_FAILED", "Vui lòng thử đổi tên vai trò hoặc thử lại sau")
				return response.ToJSON()
			}

			response := NewActionResponse(
				"switch_device_role",
				"switch_device_role",
				fmt.Sprintf("Đã chuyển sang vai trò: %s", matchedRoleName),
				"completed",
				false,
			)
			response.Metadata = map[string]string{
				"requested_role_name": params.RoleName,
				"matched_role_name":   matchedRoleName,
			}
			return response.ToJSON()
		}
		return "", fmt.Errorf("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
	}

	return "", fmt.Errorf("không tìm thấy chat_session_operator trong context")
}

// restoreDeviceDefaultRoleHandler xử lý khôi phục vai trò mặc định của thiết bị
func restoreDeviceDefaultRoleHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool khôi phục vai trò mặc định của thiết bị")

	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			if err := chatSessionOperator.LocalMcpRestoreDeviceDefaultRole(ctx); err != nil {
				log.Errorf("Khôi phục vai trò mặc định của thiết bị thất bại: %v", err)
				response := NewErrorResponse("restore_device_default_role", fmt.Sprintf("Khôi phục vai trò mặc định thất bại: %v", err), "RESTORE_ROLE_FAILED", "Vui lòng thử lại sau")
				return response.ToJSON()
			}

			response := NewActionResponse(
				"restore_device_default_role",
				"restore_device_default_role",
				"Đã khôi phục vai trò mặc định của thiết bị",
				"completed",
				false,
			)
			return response.ToJSON()
		}
		return "", fmt.Errorf("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
	}

	return "", fmt.Errorf("không tìm thấy chat_session_operator trong context")
}

func searchKnowledgeHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Thực thi tool truy xuất knowledge base")

	var params SearchKnowledgeParams
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse("search_knowledge", "Phân tích tham số thất bại", "PARSE_ERROR", "Vui lòng kiểm tra định dạng tham số query")
			return response.ToJSON()
		}
	}
	params.Query = strings.TrimSpace(params.Query)
	if params.Query == "" {
		response := NewErrorResponse("search_knowledge", "query không được để trống", "INVALID_QUERY", "Vui lòng cung cấp nội dung cần tìm")
		return response.ToJSON()
	}
	if params.TopK <= 0 {
		params.TopK = 5
	}

	chatSessionOperatorValue := ctx.Value("chat_session_operator")
	if chatSessionOperatorValue == nil {
		return "", fmt.Errorf("không tìm thấy chat_session_operator trong context")
	}
	chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator)
	if !ok {
		return "", fmt.Errorf("chat_session_operator lấy từ context không phải kiểu ChatSessionOperator")
	}

	hits, err := chatSessionOperator.LocalMcpSearchKnowledge(ctx, params.Query, params.TopK, params.KnowledgeBaseIDs)
	if err != nil {
		response := NewErrorResponse("search_knowledge", fmt.Sprintf("Tìm kiếm thông tin thất bại: %v", err), "SEARCH_FAILED", "Vui lòng thử lại sau")
		return response.ToJSON()
	}

	data := map[string]interface{}{
		"query": params.Query,
		"hits":  hits,
		"count": len(hits),
	}
	if len(hits) == 0 {
		response := NewContentResponse("search_knowledge", data, "Không tìm thấy đủ thông tin liên quan")
		return response.ToJSON()
	}

	var builder strings.Builder
	for i, hit := range hits {
		content := strings.TrimSpace(hit.Content)
		if content == "" {
			continue
		}
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, content))
	}
	msg := strings.TrimSpace(builder.String())
	if msg == "" {
		msg = "Đã lấy thông tin liên quan"
	}
	response := NewContentResponse("search_knowledge", data, msg)
	return response.ToJSON()
}

// getWeekNumber lấy số tuần
func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// formatVietnameseDateTime format ngày giờ tiếng Việt
func formatVietnameseDateTime(t time.Time) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "Chủ nhật",
		time.Monday:    "Thứ hai",
		time.Tuesday:   "Thứ ba",
		time.Wednesday: "Thứ tư",
		time.Thursday:  "Thứ năm",
		time.Friday:    "Thứ sáu",
		time.Saturday:  "Thứ bảy",
	}

	return fmt.Sprintf("%02d/%02d/%d %s %02d:%02d:%02d",
		t.Year(), int(t.Month()), int(t.Month()),
		weekdays[t.Weekday()],
		t.Hour(), t.Minute(), t.Second(),
	)
}

// getWeekdayVietnamese lấy thứ trong tuần bằng tiếng Việt
func getWeekdayVietnamese(weekday time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "Chủ nhật",
		time.Monday:    "Thứ hai",
		time.Tuesday:   "Thứ ba",
		time.Wednesday: "Thứ tư",
		time.Thursday:  "Thứ năm",
		time.Friday:    "Thứ sáu",
		time.Saturday:  "Thứ bảy",
	}
	return weekdays[weekday]
}

// RegisterChatMCPTools là hàm public để bên ngoài đăng ký MCP tool chat
func RegisterChatMCPTools() {
	InitChatLocalMCPTools()
}

// Phát nhạc
func GetMusicAudioData(ctx context.Context, musicParams *PlayMusicParams) ([]byte, string, error) {
	musicName := musicParams.Name
	//welcome := musicParams.Welcome
	welcome := ""
	log.Infof("Đang tìm nhạc: %s, welcome: %s", musicName, welcome)
	// Có thể lấy URL nhạc theo tên bài hát tại đây.
	// Hiện triển khai đơn giản, giả định musicName là URL hoặc lấy từ config.
	musicURL, realMusicName, ierr := getMusicURL(musicName)
	if ierr != nil {
		log.Errorf("Lấy URL nhạc thất bại: %v", ierr)
		return nil, "", fmt.Errorf("lấy URL nhạc thất bại: %v", ierr)
	}

	log.Infof("Tìm nhạc thành công URL: %s, tên nhạc: %s", musicURL, realMusicName)

	client := getHTTPClient()
	req, err := http.NewRequest("GET", musicURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("tạo request thất bại: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request API thất bại: %v", err)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("đọc response thất bại: %v", err)
	}

	log.Infof("Lấy dữ liệu nhạc %s thành công, độ dài dữ liệu audio: %d", realMusicName, len(audioData))

	return audioData, realMusicName, nil
}

/*
func GetMusicAudioData(ctx context.Context, musicParams *PlayMusicParams) ([]byte, string, error) {
	musicName := musicParams.Name
	//welcome := musicParams.Welcome
	welcome := ""
	log.Infof("Đang tìm nhạc: %s, welcome: %s", musicName, welcome)
	// Có thể lấy URL nhạc theo tên bài hát tại đây.
	// Hiện triển khai đơn giản, giả định musicName là URL hoặc lấy từ config.
	musicList := netease.Search(musicName)
	musicList = append(musicList, qq.Search(musicName)...)
	for id, music := range musicList {
		log.Infof("[%2d] %7s | %s %5sMB - %s - %s - %s\n", id, music.Source, music.Duration, music.Size, music.Title, music.Singer, music.Album)
	}

	if len(musicList) <= 0 {
		return nil, "", fmt.Errorf("không tìm thấy nhạc")
	}
	m := musicList[0]
	m.ParseMusic()
	rc, err := m.ReadCloser()
	if err != nil {
		return nil, "", fmt.Errorf("Lấy dữ liệu nhạc thất bại: %v", err)
	}
	defer rc.Close()

	audioData, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("đọc response thất bại: %v", err)
	}

	log.Infof("Lấy dữ liệu nhạc %s thành công, độ dài dữ liệu audio: %d", m.Name, len(audioData))

	return audioData, m.Name, nil

}
*/
