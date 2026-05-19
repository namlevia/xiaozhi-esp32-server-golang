package eventbus

const (
	TopicAddMessage = "add_message"
	TopicSessionEnd = "session_end"
	TopicExitChat   = "exit_chat" // Sự kiện thoát chat

	// Sự kiện liên quan lịch sử chat, đã deprecated và thống nhất dùng TopicAddMessage.
	// Deprecated: dùng TopicAddMessage thay thế.
	TopicChatHistoryUserMessage      = "chat_history_user_message"      // Message user sau ASR, đã deprecated
	TopicChatHistoryAssistantMessage = "chat_history_assistant_message" // Phản hồi bot sau LLM+TTS, đã deprecated
)
