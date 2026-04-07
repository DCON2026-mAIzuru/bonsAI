package demo

import (
	"context"
	"time"

	"bonsai_server/internal/domain"
)

var sensorFrames = []domain.SensorSnapshot{
	{Temperature: 24.6, Humidity: 58, SoilMoisture: 43, Illuminance: 12800},
	{Temperature: 24.9, Humidity: 57, SoilMoisture: 41, Illuminance: 13200},
	{Temperature: 25.1, Humidity: 55, SoilMoisture: 38, Illuminance: 14000},
	{Temperature: 24.7, Humidity: 60, SoilMoisture: 36, Illuminance: 11800},
	{Temperature: 24.3, Humidity: 62, SoilMoisture: 44, Illuminance: 9600},
	{Temperature: 24.1, Humidity: 59, SoilMoisture: 47, Illuminance: 8800},
}

type SensorSource struct {
	now func() time.Time
}

func NewSensorSource(now func() time.Time) *SensorSource {
	if now == nil {
		now = time.Now
	}
	return &SensorSource{now: now}
}

func (s *SensorSource) Current(_ context.Context) (domain.SensorSnapshot, error) {
	current := s.now().UTC()
	slot := current.Unix() / 10
	frame := sensorFrames[int(slot%int64(len(sensorFrames)))]

	frame.LastUpdated = time.Unix(slot*10, 0).UTC().Format(time.RFC3339)
	frame.Source = "demo"
	return frame, nil
}
