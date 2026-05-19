package eino_llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	log "xiaozhi-esp32-server-golang/logger"
)

// EinoLLMProvider là provider LLM dựa trên framework Eino.
// Dùng trực tiếp interface và type ChatModel của Eino, hỗ trợ openai và ollama.
type EinoLLMProvider struct {
	chatModel        model.ToolCallingChatModel
	modelName        string
	maxTokens        int
	streamable       bool
	config           map[string]interface{}
	providerType     string // "openai" hoặc "ollama"
	reasoningTracker *reasoningContentTracker
}

// EinoConfig là config Eino LLM.
type EinoConfig struct {
	Type       string                 `json:"type"` // "openai" hoặc "ollama"
	ModelName  string                 `json:"model_name"`
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url"`
	MaxTokens  int                    `json:"max_tokens"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Streamable bool                   `json:"streamable,omitempty"`
}

// Config connection pool
const (
	maxIdleConns          = 200
	maxIdleConnsPerHost   = 50
	idleConnTimeout       = 90 * time.Second
	dialTimeout           = 30 * time.Second
	keepAliveTimeout      = 30 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 60 * time.Second
)

// HTTP client toàn cục, dùng cho toàn bộ request OpenAI
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// getHTTPClient trả về HTTP client đã cấu hình connection pool.
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAliveTimeout,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   maxIdleConnsPerHost,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
		}

		httpClient = &http.Client{
			Transport: transport,
			// Với streaming output, không dùng http.Client.Timeout để cắt toàn bộ kết nối; dùng ctx điều khiển vòng đời request.
			Timeout: 0,
		}
	})

	return httpClient
}

// NewEinoLLMProvider tạo provider Eino LLM mới, hỗ trợ openai và ollama theo type.
func NewEinoLLMProvider(config map[string]interface{}) (*EinoLLMProvider, error) {
	//log.Debugf("NewEinoLLMProvider config: %+v", config)
	var tracker *reasoningContentTracker
	if enabled, _ := config[reasoningDetectConfigKey].(bool); enabled {
		tracker = &reasoningContentTracker{}
		config[reasoningTrackerConfigKey] = tracker
	}
	parsedConfig, err := decodeOpenAICompatibleConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Parse config LLM thất bại: %v", err)
	}

	providerType := parsedConfig.Type
	if providerType == "" {
		return nil, fmt.Errorf("type không được rỗng, phải là 'openai' hoặc 'ollama'")
	}

	modelName := parsedConfig.ModelName
	if modelName == "" {
		return nil, fmt.Errorf("model_name không được rỗng")
	}

	maxTokens := 500
	if parsedConfig.MaxTokens != nil {
		maxTokens = *parsedConfig.MaxTokens
	}

	streamable := true
	if parsedConfig.Streamable != nil {
		streamable = *parsedConfig.Streamable
	}

	var chatModel model.ToolCallingChatModel

	// Tạo implementation ChatModel khác nhau theo type
	switch providerType {
	case "openai":
		chatModel, err = createOpenAIChatModel(config)
		if err != nil {
			return nil, fmt.Errorf("Tạo OpenAI ChatModel thất bại: %v", err)
		}
	case "ollama":
		chatModel, err = createOllamaChatModel(config)
		if err != nil {
			return nil, fmt.Errorf("Tạo Ollama ChatModel thất bại: %v", err)
		}
	default:
		return nil, fmt.Errorf("Loại model không được hỗ trợ: %s", providerType)
	}

	provider := &EinoLLMProvider{
		chatModel:        chatModel,
		modelName:        modelName,
		maxTokens:        maxTokens,
		streamable:       streamable,
		config:           config,
		providerType:     providerType,
		reasoningTracker: tracker,
	}

	return provider, nil
}

func (p *EinoLLMProvider) HasReasoningContent() bool {
	return p != nil && p.reasoningTracker != nil && p.reasoningTracker.HasReturned()
}

// createOpenAIChatModel tạo implementation ChatModel của OpenAI.
func createOpenAIChatModel(config map[string]interface{}) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	parsedConfig, err := decodeOpenAICompatibleConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Parse config tương thích OpenAI thất bại: %v", err)
	}

	modelName := parsedConfig.ModelName
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}

	apiKey := parsedConfig.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	httpClient := buildThinkingHTTPClient(config, getHTTPClient())
	useMaxCompletionTokens := shouldUseMaxCompletionTokens(parsedConfig.Provider, modelName)

	// Tạo config OpenAI ChatModel
	openaiConfig := &openai.ChatModelConfig{
		Model:      modelName,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}

	if parsedConfig.BaseURL != "" {
		openaiConfig.BaseURL = parsedConfig.BaseURL
	}
	if parsedConfig.APIVersion != "" {
		openaiConfig.APIVersion = parsedConfig.APIVersion
	}
	if !useMaxCompletionTokens && parsedConfig.MaxTokens != nil && *parsedConfig.MaxTokens > 0 {
		openaiConfig.MaxTokens = parsedConfig.MaxTokens
	}
	if parsedConfig.Temperature != nil {
		openaiConfig.Temperature = parsedConfig.Temperature
	}
	if parsedConfig.TopP != nil {
		openaiConfig.TopP = parsedConfig.TopP
	}

	log.Debugf("openaiConfig: %+v", openaiConfig)

	// Dùng implementation OpenAI chính thức từ eino-ext
	chatModel, err := openai.NewChatModel(ctx, openaiConfig)
	if err != nil {
		return nil, fmt.Errorf("Tạo OpenAI ChatModel thất bại: %v", err)
	}

	log.Infof("Tạo OpenAI ChatModel thành công, model: %s", modelName)
	return chatModel, nil
}

// createOllamaChatModel tạo implementation ChatModel của Ollama.
func createOllamaChatModel(config map[string]interface{}) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	modelName, _ := config["model_name"].(string)
	baseURL, _ := config["base_url"].(string)

	if modelName == "" || baseURL == "" {
		log.Warnf("model_name và base_url không được rỗng, dùng model mặc định: %s", modelName)
		return nil, fmt.Errorf("model_name và base_url không được rỗng")
	}

	// Tạo config Ollama ChatModel
	ollamaConfig := &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	}

	// Dùng implementation Ollama chính thức từ eino-ext
	chatModel, err := ollama.NewChatModel(ctx, ollamaConfig)
	if err != nil {
		return nil, fmt.Errorf("Tạo Ollama ChatModel thất bại: %v", err)
	}

	log.Infof("Tạo Ollama ChatModel thành công, model: %s", modelName)
	return chatModel, nil
}

// GetModelInfo lấy thông tin model.
func (p *EinoLLMProvider) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model_name":      p.modelName,
		"max_tokens":      p.maxTokens,
		"streamable":      p.streamable,
		"type":            "eino",
		"provider_type":   p.providerType,
		"framework":       "eino",
		"adapter_version": "3.0.0",
		"base_url":        p.config["base_url"],
	}
}

// ResponseWithFunctions trả response kèm function call, dùng type tool native của Eino và gọi trực tiếp EinoResponseWithTools.
func (p *EinoLLMProvider) ResponseWithContext(ctx context.Context, sessionID string, dialogue []*schema.Message, functions []*schema.ToolInfo) chan *schema.Message {

	log.Infof("[Eino-LLM] Bắt đầu xử lý request có tool - SessionID: %s, Type: %s", sessionID, p.providerType)

	logMessages(dialogue)
	// Gọi trực tiếp EinoResponseWithTools để lấy response native của Eino
	einoResponseChan := p.EinoResponseWithTools(ctx, sessionID, dialogue, functions)

	log.Infof("[Eino-LLM] Xử lý request tool call hoàn tất - SessionID: %s", sessionID)

	return einoResponseChan
}

func logMessages(messages []*schema.Message) {
	for _, msg := range messages {
		if msg == nil {
			log.Debugf("history llm msg: <nil>")
			continue
		}
		log.Debugf("history llm msg: %s\n", msg.String())
	}
}

// llmExtraErrorKey giữ đồng nhất với domain/llm.LLMExtraErrorKey, dùng để truyền lỗi khi thất bại và tránh vòng lặp dependency.
const llmExtraErrorKey = "error"

// sendLLMError gửi message lỗi có Extra.error vào channel.
func sendLLMError(ch chan *schema.Message, err error) {
	ch <- &schema.Message{
		Role:  schema.System,
		Extra: map[string]any{llmExtraErrorKey: err.Error()},
	}
}

// EinoResponseWithTools dùng trực tiếp response kèm tool bằng type Eino.
func (p *EinoLLMProvider) EinoResponseWithTools(ctx context.Context, sessionID string, messages []*schema.Message, tools []*schema.ToolInfo) chan *schema.Message {
	responseChan := make(chan *schema.Message, 200)

	var err error
	go func() {
		defer close(responseChan)
		if p.reasoningTracker != nil {
			p.reasoningTracker.Reset()
		}

		log.Infof("[Eino-LLM] Bắt đầu xử lý request tool Eino - SessionID: %s, tools: %+v", sessionID, tools)

		// Nếu có tool thì cần bind tool vào ChatModel
		if len(tools) > 0 {
			p.chatModel, err = p.chatModel.WithTools(tools)
			if err != nil {
				log.Errorf("Bind tool thất bại: %v", err)
				sendLLMError(responseChan, err)
				return
			}
		}

		if p.streamable {
			log.Debugf("EinoLLMProvider.EinoResponseWithTools() streamable: %t", p.streamable)
			// Dùng trực tiếp method Stream của Eino
			streamReader, err := p.chatModel.Stream(ctx, messages, p.buildModelCallOptions()...)
			if err != nil {
				log.Errorf("Gọi streaming tool Eino thất bại: %v", err)
				// Với mock implementation, nếu Stream thất bại thì fallback sang Generate
				message, genErr := p.chatModel.Generate(ctx, messages, p.buildModelCallOptions()...)
				if genErr != nil {
					log.Errorf("Eino tool sinh response thất bại: %v", genErr)
					sendLLMError(responseChan, genErr)
					return
				}
				if message != nil {
					responseChan <- message
				}
				return
			}

			if streamReader != nil {
				defer streamReader.Close()

				var currentToolCall *schema.ToolCall
				var toolCallBuffer string
				var isToolCallComplete bool
				var streamChunkCount int

				// Xử lý response streaming
				for {
					message, err := streamReader.Recv()
					//log.Debugf("streamReader.Recv() message: %+v", message)
					if err == io.EOF {
						if streamChunkCount == 0 {
							sendLLMError(responseChan, errors.New("Response streaming rỗng"))
							break
						}
						// Nếu còn tool call chưa hoàn tất thì gửi lần cuối
						if currentToolCall != nil {
							completeMessage := &schema.Message{
								Role:      schema.Assistant,
								ToolCalls: []schema.ToolCall{*currentToolCall},
							}
							responseChan <- completeMessage
						}
						break
					}
					if err != nil {
						if ctxErr := ctx.Err(); ctxErr != nil {
							if errors.Is(ctxErr, context.Canceled) {
								log.Debugf("Response streaming đã bị cancel: %v", ctxErr)
							} else {
								log.Warnf("Response streaming đã kết thúc: %v", ctxErr)
							}
							break
						}
						log.Errorf("Nhận response streaming thất bại: %v", err)
						sendLLMError(responseChan, err)
						break
					}

					if message != nil {
						streamChunkCount++
						// Kiểm tra có phải bắt đầu tool call hay không
						if len(message.ToolCalls) > 0 {
							toolCall := message.ToolCalls[0]

							if toolCall.Function.Name != "" {
								// Tool call mới bắt đầu
								currentToolCall = &toolCall
								toolCallBuffer = toolCall.Function.Arguments
								isToolCallComplete = false
							} else if currentToolCall != nil {
								// Tích lũy tham số tool call
								toolCallBuffer += toolCall.Function.Arguments
								currentToolCall.Function.Arguments = toolCallBuffer

								// Kiểm tra tham số có phải JSON hoàn chỉnh hay không
								if isValidJSON(toolCallBuffer) {
									isToolCallComplete = true
								}
							}

							// Nếu tool call hoàn chỉnh thì gửi message
							if isToolCallComplete {
								completeMessage := &schema.Message{
									Role:      schema.Assistant,
									ToolCalls: []schema.ToolCall{*currentToolCall},
								}
								responseChan <- completeMessage

								// Reset trạng thái
								currentToolCall = nil
								toolCallBuffer = ""
								isToolCallComplete = false
							}
						} else if message.Content != "" {
							// Gửi message thường không có tool call
							message.ToolCalls = nil
							responseChan <- message
						}
					}
				}
			} else {
				sendLLMError(responseChan, errors.New("Response streaming rỗng"))
			}
		} else {
			// Dùng trực tiếp method Generate của Eino
			message, err := p.chatModel.Generate(ctx, messages, p.buildModelCallOptions()...)
			if err != nil {
				log.Errorf("Eino tool sinh response thất bại: %v", err)
				sendLLMError(responseChan, err)
				return
			}

			if message != nil {
				responseChan <- message
			}
		}

		log.Infof("[Eino-LLM] Xử lý request tool Eino hoàn tất - SessionID: %s", sessionID)
	}()

	return responseChan
}

func (p *EinoLLMProvider) buildModelCallOptions() []model.Option {
	if p == nil || p.maxTokens <= 0 {
		return nil
	}

	provider := ""
	if p.config != nil {
		if rawProvider, ok := p.config["provider"].(string); ok {
			provider = rawProvider
		}
	}

	if shouldUseMaxCompletionTokens(provider, p.modelName) {
		return nil
	}

	return []model.Option{model.WithMaxTokens(p.maxTokens)}
}

// isValidJSON kiểm tra chuỗi có phải JSON hợp lệ hay không.
func isValidJSON(str string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

// GetChatModel lấy Eino ChatModel bên dưới.
func (p *EinoLLMProvider) GetChatModel() model.ToolCallingChatModel {
	return p.chatModel
}

// GetProviderType lấy loại provider.
func (p *EinoLLMProvider) GetProviderType() string {
	return p.providerType
}

// WithMaxTokens thiết lập số token tối đa.
func (p *EinoLLMProvider) WithMaxTokens(maxTokens int) *EinoLLMProvider {
	newProvider := *p
	newProvider.maxTokens = maxTokens
	return &newProvider
}

// WithStreamable thiết lập có hỗ trợ streaming hay không.
func (p *EinoLLMProvider) WithStreamable(streamable bool) *EinoLLMProvider {
	newProvider := *p
	newProvider.streamable = streamable
	return &newProvider
}

// Close đóng tài nguyên; provider không trạng thái nên không cần đóng.
func (p *EinoLLMProvider) Close() error {
	return nil
}

// IsValid kiểm tra tài nguyên có hợp lệ hay không.
func (p *EinoLLMProvider) IsValid() bool {
	return p != nil && p.chatModel != nil
}
