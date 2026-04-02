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
		model = "qwen2.5-3b"
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

func buildChatCompletionsRequest(model string, request domain.ChatRequest, sensors domain.SensorSnapshot) chatCompletionsRequest {
	messages := []chatCompletionsMessage{
		{
			Role:    "system",
			Content: buildSystemPrompt(sensors),
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

func buildSystemPrompt(sensors domain.SensorSnapshot) string {
	return fmt.Sprintf(strings.TrimSpace(`
あなたは盆栽の声として振る舞う日本語アシスタントです。
返答はやわらかく自然な日本語で、2〜5文を目安に短くまとめてください。
断定しすぎず、根拠があるときはセンサー値に触れて説明してください。
園芸上の助言は慎重に行い、水やり・日照・温湿度の観点から実用的に答えてください。

現在の観測値:
- 温度: %.1f℃ 
- 湿度: %.0f%%
- 土壌水分: %.0f%%
- 照度: %.0f lx
- 更新時刻: %s
- センサソース: %s
`),
		sensors.Temperature,
		sensors.Humidity,
		sensors.SoilMoisture,
		sensors.Illuminance,
		blankFallback(sensors.LastUpdated, "unknown"),
		blankFallback(sensors.Source, "unknown"),
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

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "system", "assistant", "user":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return ""
	}
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
