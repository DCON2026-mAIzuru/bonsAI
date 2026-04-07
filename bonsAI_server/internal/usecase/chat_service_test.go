package usecase

import (
	"context"
	"errors"
	"testing"

	"bonsai_server/internal/domain"
)

type stubTranslator struct {
	results []domain.ChatTranslationResult
	err     error
}

func (s stubTranslator) Translate(
	_ context.Context,
	_ domain.ChatTranslationRequest,
) ([]domain.ChatTranslationResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func TestChatServiceTranslateFallsBackWhenPrimaryFails(t *testing.T) {
	t.Parallel()

	service := NewChatService(
		nil,
		nil,
		nil,
		stubTranslator{err: errors.New("primary failed")},
		stubTranslator{
			results: []domain.ChatTranslationResult{
				{ID: "user-1", Content: "Does it need water?"},
				{ID: "assistant-1", Content: "The soil moisture is a little low."},
			},
		},
	)

	translations, err := service.Translate(t.Context(), domain.ChatTranslationRequest{
		TargetLanguage: domain.ReplyLanguageEnglish,
		Messages: []domain.ChatTranslationMessage{
			{ID: "user-1", Role: "user", Content: "水やりは必要？"},
			{ID: "assistant-1", Role: "assistant", Content: "土壌水分が少し低めです。"},
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
	if translations[1].Content != "The soil moisture is a little low." {
		t.Fatalf("translations[1].Content = %q", translations[1].Content)
	}
}
