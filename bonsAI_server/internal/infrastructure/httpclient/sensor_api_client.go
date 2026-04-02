package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"bonsai_server/internal/domain"
)

type SensorAPIClient struct {
	endpoint string
	client   *http.Client
}

func NewSensorAPIClient(endpoint string, client *http.Client) *SensorAPIClient {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &SensorAPIClient{
		endpoint: endpoint,
		client:   client,
	}
}

func (c *SensorAPIClient) Current(ctx context.Context) (domain.SensorSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return domain.SensorSnapshot{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return domain.SensorSnapshot{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.SensorSnapshot{}, fmt.Errorf("sensor api returned %d", resp.StatusCode)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return domain.SensorSnapshot{}, err
	}

	return normalizeSensorSnapshot(raw), nil
}

func normalizeSensorSnapshot(raw map[string]any) domain.SensorSnapshot {
	return domain.SensorSnapshot{
		Temperature:  toFloat(first(raw, "temperature", "temp_c", "temp"), 24.6),
		Humidity:     toFloat(first(raw, "humidity", "humidity_percent"), 58),
		SoilMoisture: toFloat(first(raw, "soilMoisture", "soil_moisture", "moisture"), 43),
		Illuminance:  toFloat(first(raw, "illuminance", "light_lux", "lux"), 12800),
		LastUpdated:  toString(first(raw, "lastUpdated", "timestamp"), "just now"),
		Source:       "live",
	}
}

func first(raw map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			return value
		}
	}
	return nil
}

func toFloat(value any, fallback float64) float64 {
	switch v := value.(type) {
	case nil:
		return fallback
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if parsed, err := v.Float64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := json.Number(v).Float64(); err == nil {
			return parsed
		}
	}
	return fallback
}

func toString(value any, fallback string) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if value == nil || text == "" || text == "<nil>" {
		return fallback
	}
	return text
}
