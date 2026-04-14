package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"bonsai_server/internal/domain"
)

type ChatService struct {
	sensors            *SensorService
	primary            domain.ChatStreamer
	fallback           domain.ChatStreamer
	primaryTranslator  domain.ChatTranslator
	fallbackTranslator domain.ChatTranslator
	memory             domain.ChatMemoryStore
}

func NewChatService(
	sensors *SensorService,
	primary, fallback domain.ChatStreamer,
	primaryTranslator, fallbackTranslator domain.ChatTranslator,
	memory domain.ChatMemoryStore,
) *ChatService {
	return &ChatService{
		sensors:            sensors,
		primary:            primary,
		fallback:           fallback,
		primaryTranslator:  primaryTranslator,
		fallbackTranslator: fallbackTranslator,
		memory:             memory,
	}
}

func (s *ChatService) Stream(ctx context.Context, request domain.ChatRequest, writer domain.StreamWriter) error {
	sensors := s.sensors.ResolveForChat(ctx, request.Sensors)
	request = s.enrichRequestWithMemory(ctx, request)
	var primaryErr error

	if s.primary != nil {
		if err := s.streamAndRemember(ctx, request, sensors, s.primary, writer); err == nil {
			return nil
		} else {
			primaryErr = err
			log.Printf("primary stream failed, falling back: %v", err)
		}
	}

	if s.fallback == nil {
		if primaryErr != nil {
			return fmt.Errorf("llm streamer is unavailable after primary failure: %w", primaryErr)
		}
		return errors.New("llm streamer is unavailable")
	}

	return s.streamAndRemember(ctx, request, sensors, s.fallback, writer)
}

func (s *ChatService) Translate(
	ctx context.Context,
	request domain.ChatTranslationRequest,
) ([]domain.ChatTranslationResult, error) {
	var primaryErr error

	if s.primaryTranslator != nil {
		if translations, err := s.primaryTranslator.Translate(ctx, request); err == nil {
			return translations, nil
		} else {
			primaryErr = err
			log.Printf("primary translation failed, falling back: %v", err)
		}
	}

	if s.fallbackTranslator == nil {
		if primaryErr != nil {
			return nil, fmt.Errorf("chat translator is unavailable after primary failure: %w", primaryErr)
		}
		return nil, errors.New("chat translator is unavailable")
	}

	return s.fallbackTranslator.Translate(ctx, request)
}

func (s *ChatService) enrichRequestWithMemory(ctx context.Context, request domain.ChatRequest) domain.ChatRequest {
	if s.memory == nil || strings.TrimSpace(request.Message) == "" {
		return request
	}

	memories, err := s.memory.Recall(ctx, request.SessionID, request.Message)
	if err != nil {
		log.Printf("memory recall failed: %v", err)
		return request
	}
	if len(memories) == 0 {
		return request
	}

	request.Memories = memories
	return request
}

func (s *ChatService) streamAndRemember(
	ctx context.Context,
	request domain.ChatRequest,
	sensors domain.SensorSnapshot,
	streamer domain.ChatStreamer,
	writer domain.StreamWriter,
) error {
	capturingWriter := newCapturingStreamWriter(writer)
	if err := streamer.Stream(ctx, request, sensors, capturingWriter); err != nil {
		return err
	}

	s.saveConversationAsync(request, capturingWriter.Content())
	return nil
}

func (s *ChatService) saveConversationAsync(request domain.ChatRequest, assistantMessage string) {
	if s.memory == nil {
		return
	}

	userMessage := strings.TrimSpace(request.Message)
	assistantMessage = strings.TrimSpace(assistantMessage)
	if userMessage == "" || assistantMessage == "" {
		return
	}

	entry := domain.ChatMemoryEntry{
		SessionID:        request.SessionID,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		CreatedAt:        time.Now().Format(time.RFC3339),
	}

	go func() {
		saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := s.memory.SaveConversation(saveCtx, entry); err != nil {
			log.Printf("memory save failed: %v", err)
		}
	}()
}
