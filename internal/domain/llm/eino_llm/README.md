# Eino LLM Provider - triển khai đa provider thống nhất

## Tổng quan

`EinoLLMProvider` là implementation LLM dựa trên CloudWeGo Eino, hỗ trợ nhiều provider như OpenAI và Ollama. Implementation này dùng type/interface native của Eino để cung cấp API nhất quán.

## Tính năng chính

### Hỗ trợ nhiều provider

- **OpenAI**: hỗ trợ các model tương thích OpenAI.
- **Ollama**: hỗ trợ model mã nguồn mở chạy local.
- **Interface thống nhất**: mọi provider dùng cùng API.

### Native Eino

- Dùng trực tiếp `*schema.Message` và `*schema.ToolInfo`.
- Hỗ trợ tool call native của Eino.
- Hỗ trợ streaming và non-streaming.

## Ví dụ cấu hình

```go
config := map[string]interface{}{
    "type": "openai",
    "model_name": "gpt-4o-mini",
    "api_key": "your-api-key",
    "base_url": "https://api.openai.com/v1",
}
```

## Ghi chú

- Giữ nguyên các key như `type`, `model_name`, `api_key`, `base_url`.
- `ResponseWithContext` là entrypoint chính cho request LLM có context.
- `ResponseWithFunctions` chuyển tiếp sang `EinoResponseWithTools` để tái sử dụng logic tool call.
