package aliyun_funasr

// Header là header sự kiện WebSocket.
type Header struct {
	Action       string                 `json:"action,omitempty"`
	TaskID       string                 `json:"task_id,omitempty"`
	Streaming    string                 `json:"streaming,omitempty"`
	Event        string                 `json:"event,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// Params là tham số nhận diện.
type Params struct {
	Format                     string `json:"format,omitempty"`
	SampleRate                 int    `json:"sample_rate,omitempty"`
	VocabularyID               string `json:"vocabulary_id,omitempty"`
	DisfluencyRemovalEnabled   bool   `json:"disfluency_removal_enabled,omitempty"`
	SemanticPunctuationEnabled bool   `json:"semantic_punctuation_enabled,omitempty"`
}

// Output là output nhận diện.
type Output struct {
	Sentence struct {
		BeginTime   int64  `json:"begin_time"`
		EndTime     *int64 `json:"end_time"`
		Text        string `json:"text"`
		Heartbeat   bool   `json:"heartbeat"`
		SentenceEnd bool   `json:"sentence_end"`
		Words       []struct {
			BeginTime   int64  `json:"begin_time"`
			EndTime     *int64 `json:"end_time"`
			Text        string `json:"text"`
			Punctuation string `json:"punctuation"`
		} `json:"words"`
	} `json:"sentence"`
}

// Payload là payload sự kiện.
type Payload struct {
	TaskGroup  string `json:"task_group,omitempty"`
	Task       string `json:"task,omitempty"`
	Function   string `json:"function,omitempty"`
	Model      string `json:"model,omitempty"`
	Parameters Params `json:"parameters,omitempty"`
	Input      Input  `json:"input,omitempty"`
	Output     Output `json:"output,omitempty"`
	Usage      *struct {
		Duration int `json:"duration"`
	} `json:"usage,omitempty"`
}

// Input là input sự kiện, hiện dùng làm placeholder.
type Input struct{}

// Event là cấu trúc sự kiện.
type Event struct {
	Header  Header  `json:"header"`
	Payload Payload `json:"payload"`
}
