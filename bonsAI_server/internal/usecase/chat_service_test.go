package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"bonsai_server/internal/domain"
)

type stubTranslator struct {
	results []domain.ChatTranslationResult
	err     error
}

type stubStreamer struct {
	streamedRequests []domain.ChatRequest
	writeDelta       string
	err              error
}

func (s *stubStreamer) Stream(
	_ context.Context,
	request domain.ChatRequest,
	_ domain.SensorSnapshot,
	writer domain.StreamWriter,
) error {
	s.streamedRequests = append(s.streamedRequests, request)
	if s.err != nil {
		return s.err
	}
	if s.writeDelta != "" {
		if err := writer.WriteEvent("message", map[string]any{"delta": s.writeDelta}); err != nil {
			return err
		}
	}
	return writer.WriteEvent("done", map[string]any{"done": true})
}

type stubMemoryStore struct {
	memories   []domain.ChatMemory
	lastRecall struct {
		sessionID string
		message   string
	}
	saved []domain.ChatMemoryEntry
}

func (s *stubMemoryStore) EnsureReady(context.Context) error {
	return nil
}

func (s *stubMemoryStore) Recall(_ context.Context, sessionID, message string) ([]domain.ChatMemory, error) {
	s.lastRecall.sessionID = sessionID
	s.lastRecall.message = message
	return s.memories, nil
}

func (s *stubMemoryStore) ListRecent(context.Context, int) ([]domain.ChatMemory, error) {
	return s.memories, nil
}

func (s *stubMemoryStore) SaveConversation(_ context.Context, entry domain.ChatMemoryEntry) error {
	s.saved = append(s.saved, entry)
	return nil
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
		nil,
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

func TestChatServiceStreamUsesRecalledMemoriesAndSavesConversation(t *testing.T) {
	t.Parallel()

	memoryStore := &stubMemoryStore{
		memories: []domain.ChatMemory{
			{
				UserMessage:      "先週は乾きやすいと話していた",
				AssistantMessage: "朝の土の乾き方を見て少量ずつ水やりする方針でした。",
			},
		},
	}
	streamer := &stubStreamer{writeDelta: "今日は少し乾き気味です。"}
	writer := &capturingStreamWriterForTest{}

	service := NewChatService(NewSensorService(nil), streamer, nil, nil, nil, memoryStore)
	err := service.Stream(t.Context(), domain.ChatRequest{
		SessionID: "session-1",
		Message:   "今日は水やりした方がいい？",
	}, writer)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if memoryStore.lastRecall.sessionID != "session-1" {
		t.Fatalf("recall sessionID = %q", memoryStore.lastRecall.sessionID)
	}
	if len(streamer.streamedRequests) != 1 {
		t.Fatalf("len(streamedRequests) = %d", len(streamer.streamedRequests))
	}
	if len(streamer.streamedRequests[0].Memories) != 1 {
		t.Fatalf("len(request.Memories) = %d", len(streamer.streamedRequests[0].Memories))
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(memoryStore.saved) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if len(memoryStore.saved) != 1 {
		t.Fatalf("len(saved) = %d", len(memoryStore.saved))
	}
	if memoryStore.saved[0].AssistantMessage != "今日は少し乾き気味です。" {
		t.Fatalf("saved assistant message = %q", memoryStore.saved[0].AssistantMessage)
	}
}

type capturingStreamWriterForTest struct{}

func (w *capturingStreamWriterForTest) SetHeader(string, string) {}
func (w *capturingStreamWriterForTest) WriteHeader(int)          {}
func (w *capturingStreamWriterForTest) WriteChunk([]byte) (int, error) {
	return 0, nil
}
func (w *capturingStreamWriterForTest) WriteEvent(string, any) error { return nil }
func (w *capturingStreamWriterForTest) Flush()                       {}
