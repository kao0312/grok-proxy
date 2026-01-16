package internal

const (
	BaseURL   = "https://grok.com"
	AssetsURL = "https://assets.grok.com"
)

type ModelConfig struct {
	ModelName string
	ModelMode string
}

var ModelMapping = map[string]ModelConfig{
	"grok-3": {
		ModelName: "grok-3",
		ModelMode: "MODEL_MODE_FAST",
	},
	"grok-4": {
		ModelName: "grok-4",
		ModelMode: "MODEL_MODE_EXPERT",
	},
	"grok-4-auto": {
		ModelName: "grok-4-auto",
		ModelMode: "MODEL_MODE_AUTO",
	},
	"grok-4-fast": {
		ModelName: "grok-4-mini-thinking-tahoe",
		ModelMode: "MODEL_MODE_GROK_4_MINI_THINKING",
	},
	// "grok-4.1": {
	// 	ModelName: "grok-4-1-non-thinking-w-tool",
	// 	ModelMode: "MODEL_MODE_GROK_4_1_NON_THINKING",
	// },
	"grok-4.1-thinking": {
		ModelName: "grok-4-1-thinking-1129", // grok-4-1-thinking-1108b
		ModelMode: "MODEL_MODE_GROK_4_1_THINKING",
	},
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ParseContent 解析消息内容，返回文本和图片URL列表
func (m *Message) ParseContent() (text string, imageURLs []string) {
	switch content := m.Content.(type) {
	case string:
		return content, nil
	case []interface{}:
		for _, item := range content {
			if part, ok := item.(map[string]interface{}); ok {
				partType, _ := part["type"].(string)
				if partType == "text" {
					if t, ok := part["text"].(string); ok {
						text += t
					}
				} else if partType == "image_url" {
					if imgURL, ok := part["image_url"].(map[string]interface{}); ok {
						if url, ok := imgURL["url"].(string); ok {
							imageURLs = append(imageURLs, url)
						}
					}
				}
			}
		}
	}
	return text, imageURLs
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int          `json:"index"`
	Delta        *Delta       `json:"delta,omitempty"`
	Message      *MessageResp `json:"message,omitempty"`
	FinishReason *string      `json:"finish_reason"`
}

type Delta struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type MessageResp struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Grok 上游请求格式
type GrokRequest struct {
	Temporary                   bool                   `json:"temporary"`
	ModelName                   string                 `json:"modelName"`
	Message                     string                 `json:"message"`
	FileAttachments             []string               `json:"fileAttachments"`
	ImageAttachments            []interface{}          `json:"imageAttachments"`
	DisableSearch               bool                   `json:"disableSearch"`
	EnableImageGeneration       bool                   `json:"enableImageGeneration"`
	ReturnImageBytes            bool                   `json:"returnImageBytes"`
	ReturnRawGrokInXaiRequest   bool                   `json:"returnRawGrokInXaiRequest"`
	EnableImageStreaming        bool                   `json:"enableImageStreaming"`
	ImageGenerationCount        int                    `json:"imageGenerationCount"`
	ForceConcise                bool                   `json:"forceConcise"`
	ToolOverrides               map[string]interface{} `json:"toolOverrides"`
	EnableSideBySide            bool                   `json:"enableSideBySide"`
	SendFinalMetadata           bool                   `json:"sendFinalMetadata"`
	CustomPersonality           string                 `json:"customPersonality,omitempty"`
	IsReasoning                 bool                   `json:"isReasoning"`
	DisableTextFollowUps        bool                   `json:"disableTextFollowUps"`
	ResponseMetadata            ResponseMetadata       `json:"responseMetadata"`
	DisableMemory               bool                   `json:"disableMemory"`
	ForceSideBySide             bool                   `json:"forceSideBySide"`
	ModelMode                   string                 `json:"modelMode"`
	IsAsyncChat                 bool                   `json:"isAsyncChat"`
	DisableSelfHarmShortCircuit bool                   `json:"disableSelfHarmShortCircuit"`
	CollectionIds               []interface{}          `json:"collectionIds"`
}

type ResponseMetadata struct {
	ModelConfigOverride ModelConfigOverride `json:"modelConfigOverride"`
	RequestModelDetails RequestModelDetails `json:"requestModelDetails"`
}

type ModelConfigOverride struct {
	ModelMap map[string]interface{} `json:"modelMap"`
}

type RequestModelDetails struct {
	ModelID string `json:"modelId"`
}

// Grok 上游响应格式
type GrokStreamResponse struct {
	Result *GrokResult `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type GrokResult struct {
	Response     *GrokResponse     `json:"response,omitempty"`
	Conversation *ConversationInfo `json:"conversation,omitempty"`
}

type ConversationInfo struct {
	ConversationID string `json:"conversationId,omitempty"`
}

type GrokResponse struct {
	Token                            string             `json:"token,omitempty"`
	IsThinking                       bool               `json:"isThinking,omitempty"`
	MessageTag                       string             `json:"messageTag,omitempty"`
	ToolUsageCardID                  string             `json:"toolUsageCardId,omitempty"`
	WebSearchResults                 *WebSearchResults  `json:"webSearchResults,omitempty"`
	CardAttachment                   *CardAttachment    `json:"cardAttachment,omitempty"`
	CachedImageGenerationResponse    *CachedImageGen    `json:"cachedImageGenerationResponse,omitempty"`
	StreamingImageGenerationResponse *StreamingImageGen `json:"streamingImageGenerationResponse,omitempty"`
	ResponseID                       string             `json:"responseId,omitempty"`
	ConversationID                   string             `json:"conversationId,omitempty"`
}

type WebSearchResults struct {
	Results []WebSearchResult `json:"results"`
}

type WebSearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type CardAttachment struct {
	JSONData string `json:"jsonData,omitempty"`
}

type ImageCard struct {
	CardType string     `json:"cardType"`
	Caption  string     `json:"caption,omitempty"`
	Image    *ImageInfo `json:"image,omitempty"`
}

type ImageInfo struct {
	Title    string `json:"title,omitempty"`
	Original string `json:"original,omitempty"`
	Link     string `json:"link,omitempty"`
}

type CachedImageGen struct {
	ImageURL string `json:"imageUrl,omitempty"`
}

type StreamingImageGen struct {
	ImageID    string `json:"imageId,omitempty"`
	ImageURL   string `json:"imageUrl,omitempty"`
	Seq        int    `json:"seq,omitempty"`
	Progress   int    `json:"progress,omitempty"`
	Moderated  bool   `json:"moderated,omitempty"`
	ImageModel string `json:"imageModel,omitempty"`
}

// ShareRequest 会话分享请求
type ShareRequest struct {
	ResponseID    string `json:"responseId"`
	AllowIndexing bool   `json:"allowIndexing"`
}

// ShareResponse 会话分享响应
type ShareResponse struct {
	ShareLinkID string `json:"shareLinkId"`
}
