package usecase

import (
	"context"

	"bonsai_server/internal/domain"
)

type SensorService struct {
	source   domain.SensorSource
	fallback domain.SensorSnapshot
}

func NewSensorService(source domain.SensorSource) *SensorService {
	return &SensorService{
		source:   source,
		fallback: domain.DemoSensorSnapshot(),
	}
}

func (s *SensorService) Current(ctx context.Context) (domain.SensorSnapshot, error) {
	if s.source == nil {
		return s.fallback, nil
	}
	return s.source.Current(ctx)
}

func (s *SensorService) ResolveForChat(ctx context.Context, clientFallback *domain.SensorSnapshot) domain.SensorSnapshot {
	sensors, err := s.Current(ctx)
	if err == nil {
		return sensors
	}
	if clientFallback != nil {
		resolved := *clientFallback
		if resolved.Source == "" {
			resolved.Source = "client"
		}
		return resolved
	}
	return s.fallback
}
