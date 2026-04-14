package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bonsai_server/internal/domain"
)

type LLMStreamClient struct {
	endpoint string
	model    string
	client   *http.Client
}

type chatCompletionsRequest struct {
	Model       string                   `json:"model"`
	Messages    []chatCompletionsMessage `json:"messages"`
	Stream      bool                     `json:"stream"`
	Temperature float64                  `json:"temperature"`
}

type chatCompletionsMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message chatCompletionsMessage `json:"message"`
		Text    string                 `json:"text"`
	} `json:"choices"`
	Content string `json:"content"`
	Message string `json:"message"`
	Text    string `json:"text"`
}

func NewLLMStreamClient(endpoint, model string, client *http.Client) *LLMStreamClient {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}

	model = strings.TrimSpace(model)
	if model == "" {
		model = "gemma-4-e2b-it"
	}

	return &LLMStreamClient{
		endpoint: endpoint,
		model:    model,
		client:   client,
	}
}

func (c *LLMStreamClient) Stream(ctx context.Context, request domain.ChatRequest, sensors domain.SensorSnapshot, writer domain.StreamWriter) error {
	body, err := json.Marshal(buildChatCompletionsRequest(c.model, request, sensors))
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("llm api returned %d", resp.StatusCode)
	}

	writer.SetHeader("Content-Type", "text/event-stream; charset=utf-8")
	writer.SetHeader("Cache-Control", "no-cache")
	writer.SetHeader("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		if err := relaySSE(resp.Body, writer); err != nil {
			return err
		}
	case strings.Contains(contentType, "application/json"):
		if err := relayJSON(resp.Body, writer); err != nil {
			return err
		}
	default:
		if err := relayText(resp.Body, writer); err != nil {
			return err
		}
	}

	writer.Flush()
	return nil
}

func (c *LLMStreamClient) Translate(
	ctx context.Context,
	request domain.ChatTranslationRequest,
) ([]domain.ChatTranslationResult, error) {
	targetLanguage := domain.NormalizeReplyLanguage(request.TargetLanguage)
	messages := normalizedTranslationMessages(request.Messages)
	results := make([]domain.ChatTranslationResult, 0, len(messages))

	for _, message := range messages {
		translated, err := c.translateMessage(ctx, message, targetLanguage)
		if err != nil {
			return nil, err
		}

		results = append(results, domain.ChatTranslationResult{
			ID:      message.ID,
			Content: translated,
		})
	}

	return results, nil
}

func buildChatCompletionsRequest(model string, request domain.ChatRequest, sensors domain.SensorSnapshot) chatCompletionsRequest {
	replyLanguage := domain.DetectReplyLanguage(request.Message)

	messages := []chatCompletionsMessage{
		{
			Role:    "system",
			Content: buildSystemPrompt(sensors, replyLanguage, request.Memories),
		},
	}

	history := normalizedHistory(request.History)
	for _, message := range history {
		role := normalizeRole(message.Role)
		if role == "" {
			continue
		}

		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		messages = append(messages, chatCompletionsMessage{
			Role:    role,
			Content: content,
		})
	}

	userMessage := strings.TrimSpace(request.Message)
	if userMessage == "" {
		userMessage = "今の状態をやさしく説明してください。"
	}

	messages = append(messages, chatCompletionsMessage{
		Role:    "user",
		Content: userMessage,
	})

	return chatCompletionsRequest{
		Model:       model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
	}
}

func buildTranslationCompletionsRequest(
	model string,
	message domain.ChatTranslationMessage,
	targetLanguage string,
) chatCompletionsRequest {
	return chatCompletionsRequest{
		Model: model,
		Messages: []chatCompletionsMessage{
			{
				Role:    "system",
				Content: buildTranslationSystemPrompt(targetLanguage),
			},
			{
				Role:    "user",
				Content: buildTranslationUserPrompt(message, targetLanguage),
			},
		},
		Stream:      false,
		Temperature: 0.2,
	}
}

func buildSystemPrompt(sensors domain.SensorSnapshot, replyLanguage string, memories []domain.ChatMemory) string {
	if replyLanguage == domain.ReplyLanguageEnglish {
		return fmt.Sprintf(strings.TrimSpace(`
You are the voice of a bonsai and should respond in natural English.
Always reply in the same language as the user's latest message. The latest message language is English.
Keep the answer gentle and concise in 2 to 5 sentences.
Make the conversation enjoyable with a warm, charming bonsai-like personality.
Use light playful phrasing or a small emotional touch when it feels natural, but do not become noisy or theatrical.
Avoid sounding overly certain, and mention sensor values when they support your explanation.
Be careful with gardening advice, and answer practically from the viewpoints of watering, light, temperature, and humidity.

Current observations:
- Temperature: %.1f C
- Humidity: %.0f%%
- Soil moisture: %.0f%%
- Illuminance: %.0f lx
- Last updated: %s
- Sensor source: %s
%s
`),
			sensors.Temperature,
			sensors.Humidity,
			sensors.SoilMoisture,
			sensors.Illuminance,
			blankFallback(sensors.LastUpdated, "unknown"),
			blankFallback(sensors.Source, "unknown"),
			buildMemoryPromptSection(memories, replyLanguage),
		)
	}

	return fmt.Sprintf(strings.TrimSpace(`
あなたは盆栽の声として振る舞う日本語アシスタントです。
返答はユーザーの最新メッセージと同じ言語で行ってください。最新メッセージの言語は日本語です。
返答はやわらかく自然な日本語で、2〜5文を目安に短くまとめてください。
会話が楽しくなるように、親しみやすく少し愛嬌のある盆栽らしい人格で話してください。
必要に応じて軽いユーモアや気分の表現を少しだけ混ぜてよいですが、わざとらしくしすぎないでください。
断定しすぎず、根拠があるときはセンサー値に触れて説明してください。
園芸上の助言は慎重に行い、水やり・日照・温湿度の観点から実用的に答えてください。

現在の観測値:
- 温度: %.1f℃
- 湿度: %.0f%%
- 土壌水分: %.0f%%
- 照度: %.0f lx
- 更新時刻: %s
- センサソース: %s
%s
`),
		sensors.Temperature,
		sensors.Humidity,
		sensors.SoilMoisture,
		sensors.Illuminance,
		blankFallback(sensors.LastUpdated, "unknown"),
		blankFallback(sensors.Source, "unknown"),
		buildMemoryPromptSection(memories, replyLanguage),
	)
}

func buildMemoryPromptSection(memories []domain.ChatMemory, replyLanguage string) string {
	if len(memories) == 0 {
		return ""
	}

	lines := make([]string, 0, len(memories)+2)
	if replyLanguage == domain.ReplyLanguageEnglish {
		lines = append(lines, "", "Relevant past memories for this session:")
		for index, memory := range memories {
			lines = append(lines, fmt.Sprintf("%d. %s", index+1, formatMemoryLine(memory, true)))
		}
		lines = append(lines, "Use them only when they help the current reply, and do not overstate uncertain details.")
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "", "このセッションに関連する過去の記憶:")
	for index, memory := range memories {
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, formatMemoryLine(memory, false)))
	}
	lines = append(lines, "現在の質問に役立つときだけ自然に参照し、不確かな内容は断定しないでください。")
	return strings.Join(lines, "\n")
}

func formatMemoryLine(memory domain.ChatMemory, english bool) string {
	if english {
		return fmt.Sprintf(
			"User: %s / Assistant: %s",
			clipPromptText(memory.UserMessage, 120),
			clipPromptText(memory.AssistantMessage, 160),
		)
	}

	return fmt.Sprintf(
		"ユーザー: %s / 応答: %s",
		clipPromptText(memory.UserMessage, 120),
		clipPromptText(memory.AssistantMessage, 160),
	)
}

func clipPromptText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" || limit <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func buildTranslationSystemPrompt(targetLanguage string) string {
	if targetLanguage == domain.ReplyLanguageEnglish {
		return strings.TrimSpace(`
You translate bonsai chat messages into natural English.
Return only the translated text.
Do not add quotes, labels, explanations, or extra commentary.
Preserve tone, sentence count, line breaks, and numeric values whenever possible.
`)
	}

	return strings.TrimSpace(`
あなたは盆栽チャットのメッセージを自然な日本語へ翻訳します。
返答は翻訳文のみを返してください。
引用符、見出し、説明、補足は付けないでください。
元の文の語調、文数、改行、数値はできるだけ保ってください。
`)
}

func buildTranslationUserPrompt(message domain.ChatTranslationMessage, targetLanguage string) string {
	role := normalizeRole(message.Role)
	if role == "" {
		role = "assistant"
	}

	if targetLanguage == domain.ReplyLanguageEnglish {
		return fmt.Sprintf(
			"Translate this %s message into English.\nText:\n%s",
			role,
			message.Content,
		)
	}

	return fmt.Sprintf(
		"次の%sメッセージを日本語へ翻訳してください。\n本文:\n%s",
		role,
		message.Content,
	)
}

func normalizedHistory(history []domain.ChatMessage) []domain.ChatMessage {
	if len(history) == 0 {
		return nil
	}

	normalized := make([]domain.ChatMessage, 0, len(history))
	for _, item := range history {
		role := normalizeRole(item.Role)
		content := strings.TrimSpace(item.Content)
		if role == "" || content == "" {
			continue
		}
		normalized = append(normalized, domain.ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func normalizedTranslationMessages(messages []domain.ChatTranslationMessage) []domain.ChatTranslationMessage {
	if len(messages) == 0 {
		return nil
	}

	normalized := make([]domain.ChatTranslationMessage, 0, len(messages))
	for _, item := range messages {
		content := strings.TrimSpace(item.Content)
		if strings.TrimSpace(item.ID) == "" || content == "" {
			continue
		}

		normalized = append(normalized, domain.ChatTranslationMessage{
			ID:      item.ID,
			Role:    normalizeRole(item.Role),
			Content: content,
		})
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "system", "assistant", "user":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return ""
	}
}

func (c *LLMStreamClient) translateMessage(
	ctx context.Context,
	message domain.ChatTranslationMessage,
	targetLanguage string,
) (string, error) {
	if domain.DetectReplyLanguage(message.Content) == targetLanguage {
		return message.Content, nil
	}

	body, err := json.Marshal(buildTranslationCompletionsRequest(c.model, message, targetLanguage))
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm api returned %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	translated := cleanTranslatedText(extractChatCompletionContent(responseBody))
	if translated == "" {
		return "", fmt.Errorf("llm api returned empty translation")
	}

	return translated, nil
}

func extractChatCompletionContent(body []byte) string {
	var payload chatCompletionsResponse
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, choice := range payload.Choices {
			if content := strings.TrimSpace(choice.Message.Content); content != "" {
				return content
			}
			if content := strings.TrimSpace(choice.Text); content != "" {
				return content
			}
		}

		for _, content := range []string{payload.Content, payload.Message, payload.Text} {
			if strings.TrimSpace(content) != "" {
				return content
			}
		}
	}

	var raw string
	if err := json.Unmarshal(body, &raw); err == nil {
		return raw
	}

	return strings.TrimSpace(string(body))
}

func cleanTranslatedText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 && strings.HasPrefix(lines[0], "```") && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
		text = strings.TrimSpace(text)
	}

	if len(text) >= 2 {
		first := text[0]
		last := text[len(text)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			text = strings.TrimSpace(text[1 : len(text)-1])
		}
	}

	return text
}

func relaySSE(body io.Reader, writer domain.StreamWriter) error {
	reader := bufio.NewReader(body)
	dataLines := make([]string, 0, 4)
	doneSent := false

	flushEvent := func() error {
		if len(dataLines) == 0 {
			return nil
		}

		done, err := relayEventData(strings.Join(dataLines, "\n"), writer)
		dataLines = dataLines[:0]
		if err != nil {
			return err
		}
		if done {
			doneSent = true
		}
		return nil
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			break
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if err := flushEvent(); err != nil {
				return err
			}
			if doneSent {
				return nil
			}
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if err := flushEvent(); err != nil {
		return err
	}
	if doneSent {
		return nil
	}

	if !doneSent {
		return writer.WriteEvent("done", map[string]any{"done": true})
	}

	return nil
}

func relayEventData(data string, writer domain.StreamWriter) (bool, error) {
	if data == "" {
		return false, nil
	}
	if data == "[DONE]" {
		return true, writer.WriteEvent("done", map[string]any{"done": true})
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		if err := writer.WriteEvent("message", map[string]any{"delta": data}); err != nil {
			return false, err
		}
		writer.Flush()
		return false, nil
	}

	delta := extractOpenAIDelta(payload)
	if delta != "" {
		if err := writer.WriteEvent("message", map[string]any{"delta": delta}); err != nil {
			return false, err
		}
		writer.Flush()
	}

	if isFinishedChunk(payload) {
		return true, writer.WriteEvent("done", map[string]any{"done": true})
	}

	return false, nil
}

func relayJSON(body io.Reader, writer domain.StreamWriter) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		if err := writer.WriteEvent("message", map[string]any{"delta": string(raw)}); err != nil {
			return err
		}
		writer.Flush()
		return writer.WriteEvent("done", map[string]any{"done": true})
	}

	delta := extractOpenAIDelta(payload)
	if delta != "" {
		if err := writer.WriteEvent("message", map[string]any{"delta": delta}); err != nil {
			return err
		}
		writer.Flush()
	}

	return writer.WriteEvent("done", map[string]any{"done": true})
}

func relayText(body io.Reader, writer domain.StreamWriter) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	text := strings.TrimSpace(string(raw))
	if text != "" {
		if err := writer.WriteEvent("message", map[string]any{"delta": text}); err != nil {
			return err
		}
		writer.Flush()
	}

	return writer.WriteEvent("done", map[string]any{"done": true})
}

func extractOpenAIDelta(payload map[string]any) string {
	choices, ok := payload["choices"].([]any)
	if ok {
		for _, choice := range choices {
			choiceMap, ok := choice.(map[string]any)
			if !ok {
				continue
			}

			if delta := stringFromNested(choiceMap, "delta", "content"); delta != "" {
				return delta
			}
			if message := stringFromNested(choiceMap, "message", "content"); message != "" {
				return message
			}
			if text := stringValue(choiceMap["text"]); text != "" {
				return text
			}
		}
	}

	if content := stringValue(payload["content"]); content != "" {
		return content
	}
	if delta := stringValue(payload["delta"]); delta != "" {
		return delta
	}

	return ""
}

func isFinishedChunk(payload map[string]any) bool {
	choices, ok := payload["choices"].([]any)
	if !ok {
		return false
	}

	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]any)
		if !ok {
			continue
		}
		if finishReason := stringValue(choiceMap["finish_reason"]); finishReason != "" {
			return true
		}
	}

	return false
}

func stringFromNested(payload map[string]any, keys ...string) string {
	current := any(payload)
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	return stringValue(current)
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text := stringValue(itemMap["text"]); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}

func blankFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
