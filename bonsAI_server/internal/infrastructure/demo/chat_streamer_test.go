package demo

import (
	"strings"
	"testing"

	"bonsai_server/internal/domain"
)

func TestReplyChunksReturnsEnglishForEnglishMessage(t *testing.T) {
	t.Parallel()

	chunks := replyChunks("Does it need water today?", domain.SensorSnapshot{
		Temperature:  24.6,
		Humidity:     58,
		SoilMoisture: 30,
		Illuminance:  12800,
	})

	joined := strings.Join(chunks, "")
	if !strings.Contains(joined, "soil moisture is a little low") {
		t.Fatalf("expected english reply, got %q", joined)
	}
	if strings.Contains(joined, "土壌水分") {
		t.Fatalf("expected english-only reply, got %q", joined)
	}
}

func TestTranslateReturnsEnglishForUserAndAssistantMessages(t *testing.T) {
	t.Parallel()

	streamer := NewChatStreamer(0)

	translations, err := streamer.Translate(t.Context(), domain.ChatTranslationRequest{
		TargetLanguage: domain.ReplyLanguageEnglish,
		Messages: []domain.ChatTranslationMessage{
			{ID: "user-1", Role: "user", Content: "水やりは必要？"},
			{
				ID:      "assistant-1",
				Role:    "assistant",
				Content: "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。",
			},
		},
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	if len(translations) != 2 {
		t.Fatalf("len(translations) = %d", len(translations))
	}
	if translations[0].Content != "Does it need water?" {
		t.Fatalf("translations[0].Content = %q", translations[0].Content)
	}
	if !strings.Contains(translations[1].Content, "soil moisture is a little low") {
		t.Fatalf("translations[1].Content = %q", translations[1].Content)
	}
}
