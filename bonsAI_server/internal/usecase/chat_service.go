package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bonsai_server/internal/domain"
)

type ChatService struct {
	sensors            *SensorService
	primary            domain.ChatStreamer
	fallback           domain.ChatStreamer
	primaryTranslator  domain.ChatTranslator
	fallbackTranslator domain.ChatTranslator
}

func NewChatService(
	sensors *SensorService,
	primary, fallback domain.ChatStreamer,
	primaryTranslator, fallbackTranslator domain.ChatTranslator,
) *ChatService {
	return &ChatService{
		sensors:            sensors,
		primary:            primary,
		fallback:           fallback,
		primaryTranslator:  primaryTranslator,
		fallbackTranslator: fallbackTranslator,
	}
}

func (s *ChatService) Stream(ctx context.Context, request domain.ChatRequest, writer domain.StreamWriter) error {
	sensors := s.sensors.ResolveForChat(ctx, request.Sensors)
	var primaryErr error

	if s.primary != nil {
		if err := s.primary.Stream(ctx, request, sensors, writer); err == nil {
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

	return s.fallback.Stream(ctx, request, sensors, writer)
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
