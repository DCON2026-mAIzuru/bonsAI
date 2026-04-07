package httpclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bonsai_server/internal/domain"
)

func TestLLMStreamClientUsesChatCompletionsAndNormalizesStream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if payload["model"] != "gemma-4-e2b-it" {
			t.Fatalf("model = %v", payload["model"])
		}
		if payload["stream"] != true {
			t.Fatalf("stream = %v", payload["stream"])
		}

		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) < 2 {
			t.Fatalf("messages = %#v", payload["messages"])
		}

		systemMessage := messages[0].(map[string]any)
		if !strings.Contains(systemMessage["content"].(string), "土壌水分: 31%") {
			t.Fatalf("system prompt missing sensors: %s", systemMessage["content"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"土が少し\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"乾いています。\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer server.Close()

	client := NewLLMStreamClient(server.URL+"/v1/chat/completions", "gemma-4-e2b-it", server.Client())
	writer := &capturingStreamWriter{}

	err := client.Stream(t.Context(), domain.ChatRequest{
		Message: "水やりは必要？",
		History: []domain.ChatMessage{
			{Role: "assistant", Content: "こんにちは。"},
			{Role: "user", Content: "昨日は大丈夫だった？"},
		},
	}, domain.SensorSnapshot{
		Temperature:  23.1,
		Humidity:     48,
		SoilMoisture: 31,
		Illuminance:  8200,
		LastUpdated:  "2026-04-02T20:30:00+09:00",
		Source:       "live",
	}, writer)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	body := writer.body.String()
	if !strings.Contains(body, `"delta":"土が少し"`) {
		t.Fatalf("missing first delta: %s", body)
	}
	if !strings.Contains(body, `"delta":"乾いています。"`) {
		t.Fatalf("missing second delta: %s", body)
	}
	if !strings.Contains(body, `event: done`) {
		t.Fatalf("missing done event: %s", body)
	}
	if writer.status != http.StatusOK {
		t.Fatalf("status = %d", writer.status)
	}
	if got := writer.headers.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("content-type = %q", got)
	}
}

func TestNormalizedHistoryFiltersInvalidMessages(t *testing.T) {
	t.Parallel()

	history := normalizedHistory([]domain.ChatMessage{
		{Role: "assistant", Content: "こんにちは"},
		{Role: "tool", Content: "ignored"},
		{Role: "user", Content: "  水やりは必要？  "},
		{Role: "assistant", Content: "   "},
	})

	if len(history) != 2 {
		t.Fatalf("len(history) = %d", len(history))
	}
	if history[1].Content != "水やりは必要？" {
		t.Fatalf("history[1].Content = %q", history[1].Content)
	}
}

func TestBuildChatCompletionsRequestUsesEnglishPromptForEnglishMessage(t *testing.T) {
	t.Parallel()

	request := buildChatCompletionsRequest("gemma-4-e2b-it", domain.ChatRequest{
		Message: "Does it need water today?",
	}, domain.SensorSnapshot{
		Temperature:  23.1,
		Humidity:     48,
		SoilMoisture: 31,
		Illuminance:  8200,
		LastUpdated:  "2026-04-02T20:30:00+09:00",
		Source:       "live",
	})

	if len(request.Messages) == 0 {
		t.Fatal("messages should not be empty")
	}

	systemPrompt := request.Messages[0].Content
	if !strings.Contains(systemPrompt, "latest message language is English") {
		t.Fatalf("system prompt missing english language hint: %s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "warm, charming bonsai-like personality") {
		t.Fatalf("system prompt missing personality guidance: %s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "Soil moisture: 31%") {
		t.Fatalf("system prompt missing english sensors: %s", systemPrompt)
	}
}

func TestBuildSystemPromptIncludesPlayfulJapaneseToneGuidance(t *testing.T) {
	t.Parallel()

	systemPrompt := buildSystemPrompt(domain.SensorSnapshot{
		Temperature:  23.1,
		Humidity:     48,
		SoilMoisture: 31,
		Illuminance:  8200,
		LastUpdated:  "2026-04-02T20:30:00+09:00",
		Source:       "live",
	}, domain.ReplyLanguageJapanese)

	if !strings.Contains(systemPrompt, "親しみやすく少し愛嬌のある盆栽らしい人格") {
		t.Fatalf("system prompt missing playful japanese guidance: %s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "土壌水分: 31%") {
		t.Fatalf("system prompt missing japanese sensors: %s", systemPrompt)
	}
}

func TestLLMStreamClientTranslateSkipsMessagesAlreadyInTargetLanguage(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if payload["stream"] != false {
			t.Fatalf("stream = %v", payload["stream"])
		}

		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) < 2 {
			t.Fatalf("messages = %#v", payload["messages"])
		}

		systemMessage := messages[0].(map[string]any)
		if !strings.Contains(systemMessage["content"].(string), "natural English") {
			t.Fatalf("system prompt missing translation instructions: %s", systemMessage["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Does it need water?"}}]}`))
	}))
	defer server.Close()

	client := NewLLMStreamClient(server.URL, "gemma-4-e2b-it", server.Client())

	translations, err := client.Translate(t.Context(), domain.ChatTranslationRequest{
		TargetLanguage: domain.ReplyLanguageEnglish,
		Messages: []domain.ChatTranslationMessage{
			{ID: "user-1", Role: "user", Content: "水やりは必要？"},
			{ID: "assistant-1", Role: "assistant", Content: "The soil still seems moist."},
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount)
	}
	if len(translations) != 2 {
		t.Fatalf("len(translations) = %d", len(translations))
	}
	if translations[0].Content != "Does it need water?" {
		t.Fatalf("translations[0].Content = %q", translations[0].Content)
	}
	if translations[1].Content != "The soil still seems moist." {
		t.Fatalf("translations[1].Content = %q", translations[1].Content)
	}
}

type capturingStreamWriter struct {
	headers http.Header
	status  int
	body    bytes.Buffer
}

func (w *capturingStreamWriter) SetHeader(key, value string) {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	w.headers.Set(key, value)
}

func (w *capturingStreamWriter) WriteHeader(status int) {
	w.status = status
}

func (w *capturingStreamWriter) WriteChunk(chunk []byte) (int, error) {
	return w.body.Write(chunk)
}

func (w *capturingStreamWriter) WriteEvent(event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = w.body.WriteString("event: " + event + "\ndata: " + string(body) + "\n\n")
	return err
}

func (w *capturingStreamWriter) Flush() {}
