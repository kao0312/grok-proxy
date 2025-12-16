package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
)

var (
	toolUsageCardPattern = regexp.MustCompile(`<xai:tool_usage_card>`)
	grokRenderPattern    = regexp.MustCompile(`(?s)<grok:render[^>]*>.*?</grok:render>`)
)

func escapeMarkdownText(text string) string {
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	return text
}

func HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var models []ModelInfo
	for id := range ModelMapping {
		models = append(models, ModelInfo{
			ID:      id,
			Object:  "model",
			Created: 1700000000,
			OwnedBy: "grok",
		})
	}

	resp := ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	cookie := fmt.Sprintf("sso-rw=%s;sso=%s", token, token)

	modelConfig, exists := ModelMapping[req.Model]
	if !exists {
		modelConfig = ModelConfig{
			ModelName: req.Model,
			ModelMode: "MODEL_MODE_AUTO",
		}
	}

	fileAttachments, err := ExtractAndUploadImages(req.Messages, cookie)
	if err != nil {
		LogError("Failed to upload images: %v", err)
	}

	grokReq := prepareGrokRequest(req.Messages, modelConfig, fileAttachments)

	body, _ := json.Marshal(grokReq)
	LogDebug("Grok request: %s", string(body))

	upstreamReq, err := fhttp.NewRequest("POST", BaseURL+"/rest/app-chat/conversations/new", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	SetChatHeaders(upstreamReq, cookie)

	client := GetHTTPClient()
	resp, err := client.Do(upstreamReq)
	if err != nil {
		LogError("Failed to connect to upstream: %v", err)
		http.Error(w, "Failed to connect to upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText := string(bodyBytes)
		LogError("Upstream error - Status: %d, Response: %s", resp.StatusCode, bodyText)
		http.Error(w, fmt.Sprintf("Upstream error: %d", resp.StatusCode), resp.StatusCode)
		return
	}

	if req.Stream {
		handleStreamResponse(w, resp, req.Model, cookie)
	} else {
		handleNonStreamResponse(w, resp, req.Model, cookie)
	}
}

func prepareGrokRequest(messages []Message, modelConfig ModelConfig, fileAttachments []string) GrokRequest {
	var processed []string
	var lastRole string
	var customPersonality string

	var nonSystemMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			text, _ := msg.ParseContent()
			customPersonality = text
			continue
		}
		nonSystemMessages = append(nonSystemMessages, msg)
	}

	// 单条 user 消息直接发送，多条消息使用 "USER: xxx" 格式
	if len(nonSystemMessages) == 1 && nonSystemMessages[0].Role == "user" {
		text, _ := nonSystemMessages[0].ParseContent()
		if text != "" {
			processed = append(processed, text)
		}
	} else {
		for _, msg := range nonSystemMessages {
			role := "user"
			if msg.Role == "assistant" {
				role = "assistant"
			}

			text, _ := msg.ParseContent()
			if text == "" {
				continue
			}

			if role == lastRole && len(processed) > 0 {
				processed[len(processed)-1] += "\n" + text
			} else {
				processed = append(processed, fmt.Sprintf("%s: %s", strings.ToUpper(role), text))
				lastRole = role
			}
		}
	}

	grokReq := GrokRequest{
		Temporary:                 true,
		ModelName:                 modelConfig.ModelName,
		Message:                   strings.Join(processed, "\n"),
		FileAttachments:           []string{},
		ImageAttachments:          []interface{}{},
		DisableSearch:             false,
		EnableImageGeneration:     true,
		ReturnImageBytes:          false,
		ReturnRawGrokInXaiRequest: false,
		EnableImageStreaming:      true,
		ImageGenerationCount:      2,
		ForceConcise:              false,
		ToolOverrides:             map[string]interface{}{},
		EnableSideBySide:          false,
		SendFinalMetadata:         true,
		IsReasoning:               false,
		DisableTextFollowUps:      false,
		ResponseMetadata: ResponseMetadata{
			ModelConfigOverride: ModelConfigOverride{
				ModelMap: map[string]interface{}{},
			},
			RequestModelDetails: RequestModelDetails{
				ModelID: modelConfig.ModelName,
			},
		},
		DisableMemory:               true,
		ForceSideBySide:             false,
		ModelMode:                   modelConfig.ModelMode,
		IsAsyncChat:                 false,
		DisableSelfHarmShortCircuit: false,
		CollectionIds:               []interface{}{},
	}

	if customPersonality != "" {
		grokReq.CustomPersonality = customPersonality
	}

	if len(fileAttachments) > 0 {
		grokReq.FileAttachments = fileAttachments
	}

	return grokReq
}

func processToolResponse(data *GrokResponse) string {
	if data == nil || data.MessageTag == "tool_usage_card" {
		return ""
	}

	if data.CardAttachment != nil && data.CardAttachment.JSONData != "" {
		var cardData ImageCard
		if err := json.Unmarshal([]byte(data.CardAttachment.JSONData), &cardData); err == nil {
			if cardData.CardType == "image_card" && cardData.Image != nil {
				var result string
				if cardData.Image.Original != "" {
					caption := cardData.Caption
					if caption == "" && cardData.Image.Title != "" {
						caption = cardData.Image.Title
					}
					if caption == "" {
						caption = "image"
					}
					result += fmt.Sprintf("\n![%s](%s)\n", escapeMarkdownText(caption), cardData.Image.Original)
				}
				if cardData.Image.Title != "" && cardData.Image.Link != "" {
					result += fmt.Sprintf("\n[%s](%s)\n", escapeMarkdownText(cardData.Image.Title), cardData.Image.Link)
				}
				if result != "" {
					return result
				}
			}
		}
	}

	if data.WebSearchResults != nil && len(data.WebSearchResults.Results) > 0 {
		var results []string
		for _, r := range data.WebSearchResults.Results {
			if r.Title != "" && r.URL != "" {
				results = append(results, fmt.Sprintf("[%s](%s)", escapeMarkdownText(r.Title), r.URL))
			}
		}
		if len(results) > 0 {
			return "\n" + strings.Join(results, "\n") + "\n"
		}
	}

	text := data.Token
	if text == "" || toolUsageCardPattern.MatchString(text) {
		return ""
	}

	text = grokRenderPattern.ReplaceAllString(text, "")
	return text
}

func createChunk(model, content, reasoning string, finished, isFirstChunk bool) ChatCompletionChunk {
	chunk := ChatCompletionChunk{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index:        0,
				Delta:        &Delta{},
				FinishReason: nil,
			},
		},
	}

	if finished {
		chunk.Choices[0].FinishReason = stringPtr("stop")
		chunk.Choices[0].Delta = nil
	} else {
		if isFirstChunk {
			chunk.Choices[0].Delta.Role = "assistant"
		}
		if content != "" {
			chunk.Choices[0].Delta.Content = content
		}
		if reasoning != "" {
			chunk.Choices[0].Delta.ReasoningContent = reasoning
		}
	}

	return chunk
}

func writeSSE(w http.ResponseWriter, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
}

func stringPtr(s string) *string {
	return &s
}

func handleStreamResponse(w http.ResponseWriter, resp *fhttp.Response, model, cookie string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	var imageURLs []string
	var conversationID, responseID string

	writeSSE(w, createChunk(model, "", "", false, true))
	flusher.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var streamResp GrokStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			continue
		}

		LogDebug("Upstream response: %s", line)

		if streamResp.Error != nil {
			writeSSE(w, map[string]interface{}{
				"error": map[string]string{
					"message": "RateLimitError",
					"type":    "rate_limit_error",
				},
			})
			flusher.Flush()
			return
		}

		data := streamResp.Result
		if data == nil {
			continue
		}

		if data.Conversation != nil && data.Conversation.ConversationID != "" {
			conversationID = data.Conversation.ConversationID
		}

		if data.Response == nil {
			continue
		}

		grokResp := data.Response

		if grokResp.ResponseID != "" {
			responseID = grokResp.ResponseID
		}

		// 只收集最终图片（progress=100），过滤掉中间的 part 图片
		if grokResp.StreamingImageGenerationResponse != nil && grokResp.StreamingImageGenerationResponse.ImageURL != "" {
			if grokResp.StreamingImageGenerationResponse.Progress == 100 {
				imageURLs = append(imageURLs, grokResp.StreamingImageGenerationResponse.ImageURL)
			}
		}
		if grokResp.CachedImageGenerationResponse != nil && grokResp.CachedImageGenerationResponse.ImageURL != "" {
			imageURLs = append(imageURLs, grokResp.CachedImageGenerationResponse.ImageURL)
		}

		isThinkingContent := grokResp.IsThinking && grokResp.MessageTag != "header"
		isSearchResult := grokResp.MessageTag == "raw_function_result" && grokResp.WebSearchResults != nil

		if isThinkingContent || isSearchResult {
			content := processToolResponse(grokResp)
			if content != "" {
				writeSSE(w, createChunk(model, "", content, false, false))
				flusher.Flush()
			}
		}

		if !grokResp.IsThinking && grokResp.MessageTag != "tool_usage_card" && grokResp.MessageTag != "header" && grokResp.MessageTag != "raw_function_result" {
			content := processToolResponse(grokResp)
			if content != "" {
				writeSSE(w, createChunk(model, content, "", false, false))
				flusher.Flush()
			}
		}
	}

	// 分享会话以公开图片访问权限
	if len(imageURLs) > 0 && conversationID != "" && responseID != "" {
		if err := shareConversation(conversationID, responseID, cookie); err != nil {
			LogError("Failed to share conversation: %v", err)
		} else {
			LogInfo("Conversation shared successfully: %s", conversationID)
		}

		for i, imageURL := range imageURLs {
			fullURL := AssetsURL + "/" + imageURL
			prefix := "\n"
			if i == 0 {
				prefix = "\n\n"
			}
			content := fmt.Sprintf("%s![image](%s)", prefix, fullURL)
			writeSSE(w, createChunk(model, content, "", false, false))
			flusher.Flush()
		}
	}

	writeSSE(w, createChunk(model, "", "", true, false))
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleNonStreamResponse(w http.ResponseWriter, resp *fhttp.Response, model, cookie string) {
	scanner := bufio.NewScanner(resp.Body)
	var finalContent, reasoningContent string
	var imageURLs []string
	var conversationID, responseID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var streamResp GrokStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			continue
		}

		LogDebug("Upstream response: %s", line)

		if streamResp.Error != nil {
			http.Error(w, "RateLimitError", http.StatusTooManyRequests)
			return
		}

		data := streamResp.Result
		if data == nil {
			continue
		}

		if data.Conversation != nil && data.Conversation.ConversationID != "" {
			conversationID = data.Conversation.ConversationID
		}

		if data.Response == nil {
			continue
		}

		grokResp := data.Response

		if grokResp.ResponseID != "" {
			responseID = grokResp.ResponseID
		}

		// 只收集最终图片（progress=100），过滤掉中间的 part 图片
		if grokResp.StreamingImageGenerationResponse != nil && grokResp.StreamingImageGenerationResponse.ImageURL != "" {
			if grokResp.StreamingImageGenerationResponse.Progress == 100 {
				imageURLs = append(imageURLs, grokResp.StreamingImageGenerationResponse.ImageURL)
			}
		}
		if grokResp.CachedImageGenerationResponse != nil && grokResp.CachedImageGenerationResponse.ImageURL != "" {
			imageURLs = append(imageURLs, grokResp.CachedImageGenerationResponse.ImageURL)
		}

		isThinkingContent := grokResp.IsThinking && grokResp.MessageTag != "header"
		isSearchResult := grokResp.MessageTag == "raw_function_result" && grokResp.WebSearchResults != nil

		if isThinkingContent || isSearchResult {
			content := processToolResponse(grokResp)
			if content != "" {
				reasoningContent += content
			}
		}

		if grokResp.Token != "" && !grokResp.IsThinking {
			finalContent += grokResp.Token
		}
	}

	if len(imageURLs) > 0 && conversationID != "" && responseID != "" {
		if err := shareConversation(conversationID, responseID, cookie); err != nil {
			LogError("Failed to share conversation: %v", err)
		} else {
			LogInfo("Conversation shared successfully: %s", conversationID)
		}

		for i, imageURL := range imageURLs {
			fullURL := AssetsURL + "/" + imageURL
			prefix := "\n"
			if i == 0 {
				prefix = "\n\n"
			}
			finalContent += fmt.Sprintf("%s![image](%s)", prefix, fullURL)
		}
	}

	chatResp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index: 0,
				Message: &MessageResp{
					Role:             "assistant",
					Content:          finalContent,
					ReasoningContent: reasoningContent,
				},
				FinishReason: stringPtr("stop"),
			},
		},
		Usage: Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(chatResp)
}

func shareConversation(conversationID, responseID, cookie string) error {
	body, err := json.Marshal(ShareRequest{
		ResponseID:    responseID,
		AllowIndexing: true,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/rest/app-chat/conversations/%s/share", BaseURL, conversationID)
	req, err := fhttp.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	SetChatHeaders(req, cookie)

	client := GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText := string(bodyBytes)
		LogError("Share conversation failed - Status: %d, Response: %s", resp.StatusCode, bodyText)
		return fmt.Errorf("share conversation failed: status %d", resp.StatusCode)
	}

	var shareResp ShareResponse
	if err := json.NewDecoder(resp.Body).Decode(&shareResp); err != nil {
		return err
	}

	LogDebug("Share response: %+v", shareResp)
	return nil
}
