package usecase

import (
	"context"
	"errors"

	"bonsai_server/internal/domain"
)

type ChatService struct {
	sensors  *SensorService
	primary  domain.ChatStreamer
	fallback domain.ChatStreamer
}

func NewChatService(sensors *SensorService, primary, fallback domain.ChatStreamer) *ChatService {
	return &ChatService{
		sensors:  sensors,
		primary:  primary,
		fallback: fallback,
	}
}

func (s *ChatService) Stream(ctx context.Context, request domain.ChatRequest, writer domain.StreamWriter) error {
	sensors := s.sensors.ResolveForChat(ctx, request.Sensors)

	if s.primary != nil {
		if err := s.primary.Stream(ctx, request, sensors, writer); err == nil {
			return nil
		}
	}

	if s.fallback == nil {
		return errors.New("llm streamer is unavailable")
	}

	return s.fallback.Stream(ctx, request, sensors, writer)
}
