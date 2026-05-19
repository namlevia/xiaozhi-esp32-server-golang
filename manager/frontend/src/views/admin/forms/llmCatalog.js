const defaultOption = { label: 'Mặc định', value: 'default' }
const enableOption = { label: 'Bật', value: 'enabled' }
const disableOption = { label: 'Tắt', value: 'disabled' }
const clearHistoryOptions = [
  { label: 'Mặc định', value: 'default' },
  { label: 'Xóa', value: true },
  { label: 'Giữ', value: false }
]

function withDefault(options) {
  return [defaultOption, ...options]
}

function createModel(value, thinking, extra = {}) {
  return {
    value,
    label: value,
    thinking,
    ...extra
  }
}

const openAIReasoningStandard = withDefault([
  { label: 'Rất thấp', value: 'minimal' },
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' }
])

const openAIReasoningCodex = withDefault([
  { label: 'Tắt', value: 'none' },
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' }
])

const openAIReasoningCodexMax = withDefault([
  { label: 'Tắt', value: 'none' },
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' },
  { label: 'Rất cao', value: 'xhigh' }
])

const openAIReasoningLegacy = withDefault([
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' }
])

const openAIReasoningHighOnly = withDefault([
  { label: 'Cao', value: 'high' }
])

const booleanThinkingOptions = withDefault([
  enableOption,
  disableOption
])

const doubaoReasoningOptions = withDefault([
  { label: 'Tắt', value: 'minimal' },
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' }
])

const anthropicAdaptiveOptions = [
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' },
  { label: 'Rất cao', value: 'max' }
]

const openAIReasoningLatest = withDefault([
  { label: 'Tắt', value: 'none' },
  { label: 'Thấp', value: 'low' },
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' },
  { label: 'Rất cao', value: 'xhigh' }
])

const openAIReasoningLatestPro = withDefault([
  { label: 'Trung bình', value: 'medium' },
  { label: 'Cao', value: 'high' },
  { label: 'Rất cao', value: 'xhigh' }
])

const openAIReasoningRequest = {
  allowMaxTokens: false,
  allowTemperature: false,
  allowTopP: false
}

const anthropicManualThinking = {
  label: 'Suy luận sâu',
  options: withDefault([{ label: 'Suy luận thủ công', value: 'enabled' }]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled']
}

const anthropicAdaptiveThinking = {
  label: 'Suy luận sâu',
  options: withDefault([
    { label: 'Suy luận thủ công', value: 'enabled' },
    { label: 'Suy luận thích ứng', value: 'adaptive' }
  ]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled'],
  showEffortFor: ['adaptive'],
  effortOptions: anthropicAdaptiveOptions
}

const zhipuThinkingConfig = {
  label: 'Suy luận sâu',
  options: booleanThinkingOptions,
  showClearThinkingFor: ['enabled'],
  clearThinkingOptions: clearHistoryOptions
}

const aliyunThinkingConfig = {
  label: 'Suy luận sâu',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 1,
  budgetStep: 256
}

const siliconflowThinkingConfig = {
  label: 'Suy luận sâu',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 128,
  budgetMax: 32768,
  budgetStep: 128
}

const providerTypeMap = {
  openai: 'openai',
  ollama: 'ollama',
  azure: 'openai',
  anthropic: 'openai',
  zhipu: 'openai',
  aliyun: 'openai',
  doubao: 'openai',
  siliconflow: 'openai',
  deepseek: 'openai',
  gemini: 'openai',
  openrouter: 'openai',
  '9router': 'openai',
  groq: 'openai',
  together: 'openai',
  mistral: 'openai',
  xai: 'openai',
  perplexity: 'openai',
  dify: 'dify',
  coze: 'coze'
}

const knownProviders = new Set(Object.keys(providerTypeMap))

const editableBaseURLProviders = new Set(['openai', 'ollama', 'azure', 'gemini', 'openrouter', '9router', 'groq', 'together', 'mistral', 'xai', 'perplexity', 'dify', 'coze'])

const catalog = {
  openai: {
    quickUrl: 'https://api.openai.com/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Mặc định ưu tiên dùng alias ổn định chính thức; nếu cần khóa hành vi, có thể nhập thủ công model snapshot ID chính xác.',
    models: [
      createModel('gpt-5.4', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'Cường độ suy luận', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-mini', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-nano', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-pro', { label: 'Cường độ suy luận', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat-latest', false, { hint: 'Alias riêng cho ChatGPT, phù hợp tương thích workflow cũ; tích hợp mới nên ưu tiên dòng GPT-5.* chính.' }),
      createModel('gpt-5-pro', { label: 'Cường độ suy luận', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('gpt-5', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5.3-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-codex', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1', { label: 'Cường độ suy luận', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-mini', { label: 'Cường độ suy luận', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-max', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'Cường độ suy luận',
      options: openAIReasoningCodex,
      hint: 'Model tùy chỉnh không nằm trong danh sách đã biết, đã fallback về cấu hình reasoning_effort chung; hiệu lực phụ thuộc model thực tế.'
    }
  },
  ollama: {
    quickUrl: 'http://127.0.0.1:11434/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Ollama dùng dịch vụ model cục bộ hoặc riêng tư; danh sách model và địa chỉ đều có thể tự tùy chỉnh.',
    models: [],
    fallbackThinking: null
  },
  azure: {
    quickUrl: 'https://your-resource-name.openai.azure.com/openai/v1/',
    modelPlaceholder: 'Vui lòng chọn tên model chính thức hoặc nhập tên deployment tùy chỉnh',
    modelHint: 'Với Azure, trường này là tên deployment; tên trong danh sách chủ yếu để tham chiếu năng lực của model nền bên dưới.',
    models: [
      createModel('gpt-5.4', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'Cường độ suy luận', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-chat', false, { hint: 'Các model chat trong tài liệu Azure thường được truy cập qua tên deployment; khả dụng hay không còn phụ thuộc khu vực và quota.' }),
      createModel('gpt-5.3-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat', { label: 'Cường độ suy luận', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-pro', { label: 'Cường độ suy luận', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'Cường độ suy luận',
      options: openAIReasoningCodex,
      hint: 'Nếu deployment Azure tùy chỉnh không nằm trong danh sách model đã biết, hệ thống sẽ fallback về cấu hình reasoning_effort chung; khả năng tương thích thực tế phụ thuộc model được triển khai.'
    }
  },
  anthropic: {
    quickUrl: 'https://api.anthropic.com/v1/',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Mặc định nên ưu tiên alias ổn định chính thức; nếu cần cố định phiên bản hoặc kiểm tra hồi quy, hãy nhập model ID chính xác có kèm ngày.',
    models: [
      createModel('claude-opus-4-6', anthropicAdaptiveThinking),
      createModel('claude-sonnet-4-6', anthropicAdaptiveThinking),
      createModel('claude-haiku-4-5', anthropicManualThinking),
      createModel('claude-3-7-sonnet', anthropicManualThinking),
      createModel('claude-sonnet-4', anthropicManualThinking),
      createModel('claude-opus-4', anthropicManualThinking),
      createModel('claude-opus-4-1', anthropicManualThinking)
    ],
    fallbackThinking: {
      ...anthropicAdaptiveThinking,
      hint: 'Model tùy chỉnh không nằm trong danh sách tài liệu hiện có. Nếu dùng chế độ suy luận thủ công, bạn cần điền rõ budget_tokens; chỉ dùng Adaptive với các model đã được tài liệu xác nhận hỗ trợ.'
    }
  },
  zhipu: {
    quickUrl: 'https://open.bigmodel.cn/api/paas/v4',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Tài liệu Zhipu hỗ trợ dùng thinking.type và clear_thinking để điều khiển chế độ suy luận.',
    models: [
      createModel('glm-5', zhipuThinkingConfig),
      createModel('glm-4.7', zhipuThinkingConfig),
      createModel('glm-4.7-flashx', zhipuThinkingConfig),
      createModel('glm-4.7-flash', zhipuThinkingConfig),
      createModel('glm-4.6', zhipuThinkingConfig),
      createModel('glm-4.6v', zhipuThinkingConfig),
      createModel('glm-4.5', zhipuThinkingConfig),
      createModel('glm-4.5-air', zhipuThinkingConfig),
      createModel('glm-4.5-airx', zhipuThinkingConfig),
      createModel('glm-4.5v', zhipuThinkingConfig)
    ],
    fallbackThinking: {
      ...zhipuThinkingConfig,
      hint: 'Model tùy chỉnh không nằm trong danh sách tài liệu hiện có, nên đã fallback về cấu hình thinking.type / clear_thinking chung.'
    }
  },
  aliyun: {
    quickUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Mặc định nên ưu tiên alias ổn định chính thức; nếu muốn khóa vào phiên bản cụ thể, hãy nhập model ID có kèm ngày hoặc hậu tố tiểu phiên bản.',
    models: [
      createModel('qwen-plus-latest', aliyunThinkingConfig),
      createModel('qwen-turbo-latest', aliyunThinkingConfig),
      createModel('qwen3-max', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b', aliyunThinkingConfig),
      createModel('qwen3-30b-a3b', aliyunThinkingConfig),
      createModel('qwen3-next-80b-a3b-thinking', aliyunThinkingConfig),
      createModel('glm-4.7', aliyunThinkingConfig),
      createModel('glm-4.6', aliyunThinkingConfig),
      createModel('glm-4.5', aliyunThinkingConfig),
      createModel('glm-4.5-air', aliyunThinkingConfig),
      createModel('kimi-k2-thinking', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b-thinking-2507', aliyunThinkingConfig, { label: 'qwen3-235b-a22b-thinking-2507 (bản phiên bản)' }),
      createModel('qwen3-30b-a3b-thinking-2507', aliyunThinkingConfig, { label: 'qwen3-30b-a3b-thinking-2507 (bản phiên bản)' }),
      createModel('kimi/kimi-k2.5', aliyunThinkingConfig, { label: 'kimi/kimi-k2.5 (bản phiên bản)' })
    ],
    fallbackThinking: {
      ...aliyunThinkingConfig,
      hint: 'Model tùy chỉnh không nằm trong danh sách tài liệu hiện có. Nếu model hỗ trợ thinking_budget thì có thể điền theo tài liệu; để trống sẽ không gửi trường này.'
    }
  },
  doubao: {
    quickUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập Model ID, thường có kèm hậu tố phiên bản',
    modelHint: 'Với Doubao, nên ưu tiên dùng đúng Model ID chính thức. Hiện chưa xác nhận có alias ổn định dùng chung, nên hãy lấy Model ID từ console hoặc danh sách model chính thức.',
    models: [
      createModel('doubao-seed-2-0-pro-260215', { label: 'Cường độ suy luận', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Pro (doubao-seed-2-0-pro-260215)' }),
      createModel('doubao-seed-2-0-lite-260215', { label: 'Cường độ suy luận', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Lite (doubao-seed-2-0-lite-260215)' }),
      createModel('doubao-seed-2-0-mini-260215', { label: 'Cường độ suy luận', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Mini (doubao-seed-2-0-mini-260215)' }),
      createModel('doubao-seed-1-6-251015', { label: 'Cường độ suy luận', options: doubaoReasoningOptions }, { label: 'Doubao Seed 1.6 (doubao-seed-1-6-251015)' })
    ],
    fallbackThinking: {
      label: 'Cường độ suy luận',
      options: doubaoReasoningOptions,
      hint: 'Model tùy chỉnh không nằm trong danh sách đã biết, đã fallback về cấu hình reasoning_effort chung; hiệu lực phụ thuộc model thực tế.'
    }
  },
  siliconflow: {
    quickUrl: 'https://api.siliconflow.cn/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Tài liệu SiliconFlow liệt kê trực tiếp các model hỗ trợ enable_thinking; chỉ những model có trong tài liệu mới hiển thị cấu hình ngân sách suy luận.',
    models: [
      createModel('Pro/zai-org/GLM-5', siliconflowThinkingConfig),
      createModel('Pro/zai-org/GLM-4.7', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.6', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-8B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-14B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-32B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-30B-A3B', siliconflowThinkingConfig),
      createModel('tencent/Hunyuan-A13B-Instruct', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.5V', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-397B-A17B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-122B-A10B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-35B-A3B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-27B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-9B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-4B', siliconflowThinkingConfig)
    ],
    fallbackThinking: {
      ...siliconflowThinkingConfig,
      hint: 'Model tùy chỉnh không nằm trong danh sách tài liệu hiện có. Nếu model hỗ trợ enable_thinking hoặc thinking_budget thì có thể điền theo tài liệu; để trống sẽ không gửi thinking_budget.'
    }
  },
  deepseek: {
    quickUrl: 'https://api.deepseek.com/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Với DeepSeek, việc chọn model sẽ quyết định chế độ suy luận: deepseek-chat là không suy luận, còn deepseek-reasoner là suy luận.',
    models: [
      createModel('deepseek-chat', false, {
        hint: 'deepseek-chat là model không suy luận nên không cần tham số thinking bổ sung.'
      }),
      createModel('deepseek-reasoner', false, {
        hint: 'deepseek-reasoner đã có sẵn chế độ suy luận nên không cần tham số thinking bổ sung.'
      })
    ],
    fallbackThinking: {
      label: 'Suy luận sâu',
      options: booleanThinkingOptions,
      hint: 'DeepSeek chính thức khuyến nghị chuyển chế độ suy luận bằng cách đổi tên model. Nếu proxy tùy chỉnh còn hỗ trợ thinking.type, bạn có thể bật công tắc tương thích tại đây.'
    }
  },
  gemini: {
    quickUrl: 'https://generativelanguage.googleapis.com/v1beta/openai/',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model Gemini',
    modelHint: 'Gemini hỗ trợ endpoint tương thích OpenAI; nếu dùng endpoint khác, hãy chỉnh Base URL theo tài liệu Google AI Studio.',
    models: [
      createModel('gemini-3-pro', booleanThinkingOptions),
      createModel('gemini-3-flash', booleanThinkingOptions),
      createModel('gemini-3-flash-lite', booleanThinkingOptions),
      createModel('gemini-2.5-pro', booleanThinkingOptions),
      createModel('gemini-2.5-flash', booleanThinkingOptions),
      createModel('gemini-2.5-flash-lite', booleanThinkingOptions),
      createModel('gemini-2.0-flash', false)
    ],
    fallbackThinking: {
      label: 'Suy luận sâu',
      options: booleanThinkingOptions,
      hint: 'Model Gemini tùy chỉnh có thể không hỗ trợ thinking; hãy bật theo đúng tài liệu model đang dùng.'
    }
  },
  openrouter: {
    quickUrl: 'https://openrouter.ai/api/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập model dạng nhà-cung-cấp/model',
    modelHint: 'OpenRouter gom nhiều nhà cung cấp qua API tương thích OpenAI; tên model thường có dạng openai/gpt-*, anthropic/claude-*, google/gemini-*.',
    models: [
      createModel('openai/gpt-5', false),
      createModel('anthropic/claude-sonnet-4.6', false),
      createModel('google/gemini-2.5-pro', false),
      createModel('deepseek/deepseek-chat', false),
      createModel('meta-llama/llama-3.3-70b-instruct', false)
    ],
    fallbackThinking: null
  },
  '9router': {
    quickUrl: 'https://api.9router.com/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập model 9Router',
    modelHint: '9Router dùng API tương thích OpenAI; nếu endpoint của anh khác thì sửa lại Base URL theo dashboard 9Router.',
    models: [
      createModel('cx/gpt-5.5', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.5-pro', { label: 'Cường độ suy luận', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.5-mini', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.4', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.4-pro', { label: 'Cường độ suy luận', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.4-mini', { label: 'Cường độ suy luận', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.3-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.2-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5.1-codex', { label: 'Cường độ suy luận', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('cx/gpt-5-codex', { label: 'Cường độ suy luận', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'Cường độ suy luận',
      options: openAIReasoningLatest,
      hint: 'Model 9Router tùy chỉnh đang dùng cấu hình reasoning_effort kiểu OpenAI-compatible; hiệu lực phụ thuộc gateway/model thực tế.'
    }
  },
  groq: {
    quickUrl: 'https://api.groq.com/openai/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model Groq',
    modelHint: 'Groq phù hợp phản hồi nhanh với các model mở như Llama, Mixtral, Gemma.',
    models: [
      createModel('llama-3.3-70b-versatile', false),
      createModel('llama-3.1-8b-instant', false),
      createModel('gemma2-9b-it', false),
      createModel('mixtral-8x7b-32768', false)
    ],
    fallbackThinking: null
  },
  together: {
    quickUrl: 'https://api.together.xyz/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model Together AI',
    modelHint: 'Together AI cung cấp nhiều model mở qua API tương thích OpenAI.',
    models: [
      createModel('meta-llama/Llama-3.3-70B-Instruct-Turbo', false),
      createModel('meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo', false),
      createModel('mistralai/Mixtral-8x7B-Instruct-v0.1', false),
      createModel('Qwen/Qwen2.5-72B-Instruct-Turbo', false)
    ],
    fallbackThinking: null
  },
  mistral: {
    quickUrl: 'https://api.mistral.ai/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model Mistral',
    modelHint: 'Mistral API tương thích OpenAI cho chat completions với các model Mistral.',
    models: [
      createModel('mistral-large-latest', false),
      createModel('mistral-medium-latest', false),
      createModel('mistral-small-latest', false),
      createModel('open-mistral-nemo', false)
    ],
    fallbackThinking: null
  },
  xai: {
    quickUrl: 'https://api.x.ai/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model xAI',
    modelHint: 'xAI dùng API tương thích OpenAI cho dòng Grok.',
    models: [
      createModel('grok-4', false),
      createModel('grok-3', false),
      createModel('grok-3-mini', false),
      createModel('grok-2-vision-1212', false)
    ],
    fallbackThinking: null
  },
  perplexity: {
    quickUrl: 'https://api.perplexity.ai',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model Perplexity',
    modelHint: 'Perplexity phù hợp các tác vụ cần tìm kiếm web/câu trả lời có nguồn; API tương thích OpenAI.',
    models: [
      createModel('sonar', false),
      createModel('sonar-pro', false),
      createModel('sonar-reasoning', false),
      createModel('sonar-reasoning-pro', false)
    ],
    fallbackThinking: null
  }
}

function cloneOptions(options = []) {
  return options.map(option => ({ ...option }))
}

function normalizeModelName(modelName) {
  return String(modelName || '').trim().toLowerCase()
}

export function resolveLLMProvider(provider, type) {
  const normalizedProvider = String(provider || '').trim().toLowerCase()
  const normalizedType = String(type || '').trim().toLowerCase()

  if (normalizedProvider === 'openai' && ['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  if (knownProviders.has(normalizedProvider)) {
    return normalizedProvider
  }
  if (['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  return 'openai'
}

export function getProviderFixedType(provider) {
  return providerTypeMap[provider] || 'openai'
}

export function isProviderBaseURLEditable(provider) {
  return editableBaseURLProviders.has(provider)
}

export function getProviderQuickUrl(provider) {
  return catalog[provider]?.quickUrl || ''
}

export function getProviderModelOptions(provider) {
  return (catalog[provider]?.models || []).map(model => ({
    label: model.label,
    value: model.value
  }))
}

export function getProviderModelHint(provider) {
  return catalog[provider]?.modelHint || ''
}

export function getProviderModelFieldLabel(provider) {
  if (provider === 'azure') {
    return 'Tên deployment'
  }
  if (provider === 'doubao') {
    return 'Model ID'
  }
  return 'Tên model'
}

export function getProviderModelPlaceholder(provider) {
  return catalog[provider]?.modelPlaceholder || 'Vui lòng chọn hoặc nhập tên model'
}

export function resolveProviderModel(provider, modelName) {
  const normalized = normalizeModelName(modelName)
  if (!normalized) {
    return null
  }

  const models = catalog[provider]?.models || []
  return models.find(model => normalizeModelName(model.value) === normalized) || null
}

export function getProviderRequestConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  return {
    allowMaxTokens: true,
    allowTemperature: true,
    allowTopP: true,
    temperatureMax: 2,
    ...(model?.request || {})
  }
}

export function getProviderThinkingConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  if (model?.thinking === false) {
    return {
      visible: false,
      hint: model.hint || ''
    }
  }

  const source = model?.thinking || catalog[provider]?.fallbackThinking
  if (!source) {
    return {
      visible: false,
      hint: model?.hint || ''
    }
  }

  return {
    visible: true,
    label: source.label || 'Suy luận sâu',
    options: cloneOptions(source.options),
    showBudgetFor: [...(source.showBudgetFor || [])],
    budgetMin: source.budgetMin || 1,
    budgetMax: source.budgetMax || 100000,
    budgetStep: source.budgetStep || 1,
    budgetRequiredFor: [...(source.budgetRequiredFor || [])],
    showEffortFor: [...(source.showEffortFor || [])],
    effortOptions: cloneOptions(source.effortOptions || []),
    showClearThinkingFor: [...(source.showClearThinkingFor || [])],
    clearThinkingOptions: cloneOptions(source.clearThinkingOptions || clearHistoryOptions),
    hint: model?.hint || source.hint || ''
  }
}
