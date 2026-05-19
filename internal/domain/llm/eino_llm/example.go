package eino_llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"

	log "xiaozhi-esp32-server-golang/logger"
)

// ExampleConfig là config ví dụ.
var ExampleConfig = map[string]interface{}{
	"type":       "eino_llm",
	"model_name": "gpt-3.5-turbo",
	"api_key":    "your-api-key-here",
	"base_url":   "https://api.openai.com/v1",
	"max_tokens": 500,
	"streamable": true,
}

// ExampleUsage minh họa cách dùng EinoLLMProvider.
func ExampleUsage() {
	// 1. Ví dụ config OpenAI
	openaiConfig := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-3.5-turbo",
		"api_key":    "your-openai-api-key",
		"base_url":   "https://api.openai.com/v1",
		"max_tokens": 500,
		"streamable": true,
	}

	// 2. Ví dụ config Ollama
	ollamaConfig := map[string]interface{}{
		"type":       "ollama",
		"model_name": "llama2",
		"base_url":   "http://localhost:11434",
		"max_tokens": 500,
		"streamable": true,
	}

	// 3. Tạo provider
	openaiProvider, err := NewEinoLLMProvider(openaiConfig)
	if err != nil {
		log.Errorf("Tạo provider OpenAI thất bại: %v", err)
		return
	}

	ollamaProvider, err := NewEinoLLMProvider(ollamaConfig)
	if err != nil {
		log.Errorf("Tạo provider Ollama thất bại: %v", err)
		return
	}

	// 4. Dùng type message native của Eino
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "Bạn là một trợ lý hữu ích",
		},
		{
			Role:    schema.User,
			Content: "Hãy giới thiệu về framework Eino",
		},
	}

	// 5. Hội thoại cơ bản
	fmt.Println("=== Hội thoại cơ bản OpenAI ===")
	responseChan := openaiProvider.ResponseWithContext(context.Background(), "example_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	fmt.Println("=== Hội thoại cơ bản Ollama ===")
	responseChan = ollamaProvider.ResponseWithContext(context.Background(), "example_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	// 6. Ví dụ tool call
	tools := []*schema.ToolInfo{
		{
			Name:        "get_weather",
			ParamsOneOf: &schema.ParamsOneOf{
				// Định nghĩa tham số tool
			},
		},
	}

	fmt.Println("=== Hội thoại có tool call ===")
	toolResponseChan := openaiProvider.ResponseWithContext(context.Background(), "example_session", messages, tools)
	for resp := range toolResponseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	// 7. Ví dụ gọi dạng chain
	fmt.Println("=== Ví dụ gọi dạng chain ===")
	enhancedProvider := openaiProvider.
		WithMaxTokens(1000).
		WithStreamable(false)

	fmt.Printf("Loại provider: %s\n", enhancedProvider.GetProviderType())
	fmt.Printf("Thông tin model: %+v\n", enhancedProvider.GetModelInfo())
}

// ExampleAdvancedUsage là ví dụ dùng nâng cao.
func ExampleAdvancedUsage() {
	config := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-4",
		"api_key":    "your-api-key",
		"max_tokens": 1000,
		"streamable": true,
	}

	provider, err := NewEinoLLMProvider(config)
	if err != nil {
		log.Errorf("Tạo provider thất bại: %v", err)
		return
	}

	// Dùng context control
	ctx := context.Background()
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "Hãy viết một bài dài về AI",
		},
	}

	fmt.Println("=== Hội thoại có context control ===")
	responseChan := provider.ResponseWithContext(ctx, "advanced_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	// Dùng trực tiếp Eino ChatModel
	chatModel := provider.GetChatModel()
	result, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Errorf("Gọi trực tiếp ChatModel thất bại: %v", err)
		return
	}

	fmt.Printf("Kết quả gọi trực tiếp: %s\n", result.Content)
}

// ExampleMultiProvider là ví dụ đa provider.
func ExampleMultiProvider() {
	providers := make(map[string]*EinoLLMProvider)

	// Tạo nhiều provider
	configs := map[string]map[string]interface{}{
		"openai": {
			"type":       "openai",
			"model_name": "gpt-3.5-turbo",
			"api_key":    "your-openai-key",
		},
		"ollama": {
			"type":       "ollama",
			"model_name": "llama2",
			"base_url":   "http://localhost:11434",
		},
	}

	for name, config := range configs {
		provider, err := NewEinoLLMProvider(config)
		if err != nil {
			log.Errorf("Tạo provider %s thất bại: %v", name, err)
			continue
		}
		providers[name] = provider
	}

	// Dùng các provider khác nhau xử lý cùng request
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "nội dung，nội dung",
		},
	}

	for name, provider := range providers {
		fmt.Printf("=== %s nội dung ===\n", name)
		responseChan := provider.ResponseWithContext(context.Background(), "multi_session", messages, nil)
		for resp := range responseChan {
			if resp.Content != "" {
				fmt.Print(resp.Content)
			}
			if len(resp.ToolCalls) > 0 {
				fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
			}
		}
		fmt.Println()
	}
}

// ExampleWithTools nội dung
func ExampleWithTools() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Tạo provider thất bại: %v", err)
		return
	}

	// nội dungEinonội dung
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "nội dung？nội dung。",
		},
	}

	// nội dungEinonội dung
	tools := []*schema.ToolInfo{
		{
			Name:        "get_weather",
			ParamsOneOf: &schema.ParamsOneOf{
				// nội dung
				// nội dung，nội dung
			},
		},
	}

	fmt.Println("=== nội dung ===")

	// nội dungEinonội dung
	fmt.Println("--- Einonội dung ---")
	responseChan := provider.ResponseWithContext(context.Background(), "tool_session", messages, tools)
	for resp := range responseChan {
		fmt.Printf("nội dung: %+v\n", resp)
	}
}

// MultiProviderExample nội dung
func MultiProviderExample() {
	// OpenAInội dung
	fmt.Println("=== OpenAI nội dung ===")
	openaiConfig := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-3.5-turbo",
		"api_key":    "your-openai-api-key",
		"base_url":   "https://api.openai.com/v1",
		"max_tokens": 500,
	}

	openaiProvider, err := NewEinoLLMProvider(openaiConfig)
	if err != nil {
		log.Errorf("Tạo provider OpenAI thất bại: %v", err)
		return
	}

	fmt.Printf("Loại provider: %s\n", openaiProvider.GetProviderType())

	// Ollamanội dung
	fmt.Println("\n=== Ollama nội dung ===")
	ollamaConfig := map[string]interface{}{
		"type":       "ollama",
		"model_name": "llama2",
		"base_url":   "http://localhost:11434",
		"max_tokens": 500,
	}

	ollamaProvider, err := NewEinoLLMProvider(ollamaConfig)
	if err != nil {
		log.Errorf("Tạo provider Ollama thất bại: %v", err)
		return
	}

	fmt.Printf("Loại provider: %s\n", ollamaProvider.GetProviderType())

	// nội dungEinonội dung
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "nội dung。",
		},
	}

	// nội dung
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("\n--- OpenAI nội dung ---")
	openaiResponse := openaiProvider.ResponseWithContext(ctx, "openai_session", messages, nil)
	for resp := range openaiResponse {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}

	fmt.Println("\n--- Ollama nội dung ---")
	ollamaResponse := ollamaProvider.ResponseWithContext(ctx, "ollama_session", messages, nil)
	for resp := range ollamaResponse {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()
}

// EinoFrameworkAdvantages Einonội dung
func EinoFrameworkAdvantages() string {
	return `
Einonội dung：

1. **nội dung**
   - nội dung（ChatModel, Tool, ChatTemplate, Retrievernội dung）
   - nội dung
   - nội dung

2. **nội dung**
   - nội dung
   - nội dung、nội dung、nội dung
   - nội dung、nội dung、nội dung

3. **nội dung**
   - nội dung
   - nội dung
   - nội dung
   - nội dung

4. **nội dung**
   - nội dung
   - nội dung（OnStart, OnEnd, OnErrornội dung）
   - nội dung、nội dung、nội dung

5. **nội dung**
   - nội dung
   - nội dung
   - nội dung
   - nội dung

nội dung：

**nội dung**：
- nội dungEinonội dungOpenAInội dungOllama
- nội dungtypenội dung
- nội dungEino ChatModelnội dung

**Einonội dung**：
- nội dung*schema.Messagenội dung
- nội dung*schema.ToolInfonội dung
- nội dungEinonội dung，nội dung

**nội dung**：
- nội dung (WithMaxTokens, WithStreamable)
- nội dung
- nội dung
- nội dungLLMProvidernội dung

**nội dung**：
- nội dung
- nội dung
- nội dung
- nội dung

nội dungEinonội dung，nội dungLLMnội dung。
`
}

// BasicUsageExample nội dung
func BasicUsageExample() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Tạo provider thất bại: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// nội dung
	enhancedProvider := provider.
		WithMaxTokens(2000).
		WithStreamable(true)

	// nội dungEino ChatModel
	chatModel := enhancedProvider.GetChatModel()
	fmt.Printf("nội dungChatModel: %+v\n", chatModel)

	// nội dung
	providerType := enhancedProvider.GetProviderType()
	fmt.Printf("Loại provider: %s\n", providerType)

	// nội dung
	modelInfo := enhancedProvider.GetModelInfo()
	fmt.Printf("nội dungThông tin model: %+v\n", modelInfo)

	// nội dung - nội dungEinonội dung
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "nội dung，nội dungGonội dungAInội dung。",
		},
		{
			Role:    schema.User,
			Content: "nội dungEinonội dung。",
		},
	}

	// nội dung
	responseChan := enhancedProvider.ResponseWithContext(ctx, "basic_example", messages, nil)
	fmt.Printf("nội dung:\n")
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()
}

// EinoNativeExample Einonội dungAPInội dung
func EinoNativeExample() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Tạo provider thất bại: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// nội dungEinonội dung
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "nội dungAInội dung。",
		},
		{
			Role:    schema.User,
			Content: "nội dungEinonội dung。",
		},
	}

	fmt.Println("=== Einonội dungAPInội dung ===")

	// 1. nội dungEinoResponse
	fmt.Println("--- EinoResponse ---")
	responseChan := provider.ResponseWithContext(ctx, "eino_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	// 2. nội dungEinoResponseWithTools
	fmt.Println("\n--- EinoResponseWithTools ---")
	tools := []*schema.ToolInfo{
		{
			Name:        "search_docs",
			ParamsOneOf: &schema.ParamsOneOf{
				// Định nghĩa tham số tool
			},
		},
	}

	toolResponseChan := provider.ResponseWithContext(ctx, "eino_tools_session", messages, tools)
	for resp := range toolResponseChan {
		if resp.Content != "" {
			fmt.Printf("nội dung: %s\n", resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
}

func main() {
	BasicUsageExample()
}
