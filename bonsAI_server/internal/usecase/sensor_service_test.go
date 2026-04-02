package usecase

import (
	"context"
	"errors"
	"testing"

	"bonsai_server/internal/domain"
)

type failingSensorSource struct{}

func (f failingSensorSource) Current(context.Context) (domain.SensorSnapshot, error) {
	return domain.SensorSnapshot{}, errors.New("boom")
}

func TestSensorServiceResolveForChatFallsBackToClientPayload(t *testing.T) {
	t.Parallel()

	service := NewSensorService(failingSensorSource{})
	clientSnapshot := &domain.SensorSnapshot{
		Temperature:  21.2,
		Humidity:     48,
		SoilMoisture: 31,
		Illuminance:  7200,
		LastUpdated:  "client",
	}

	got := service.ResolveForChat(context.Background(), clientSnapshot)

	if got.Source != "client" {
		t.Fatalf("source = %q", got.Source)
	}
	if got.Temperature != 21.2 {
		t.Fatalf("temperature = %v", got.Temperature)
	}
}
