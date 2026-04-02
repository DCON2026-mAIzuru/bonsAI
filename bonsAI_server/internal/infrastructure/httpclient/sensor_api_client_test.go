package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSensorAPIClientSupportsAlternateKeys(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"temp_c":"22.4","humidity_percent":61,"soil_moisture":39,"light_lux":"8700","timestamp":"2026-04-02T10:00:00Z"}`))
	}))
	defer server.Close()

	client := NewSensorAPIClient(server.URL, server.Client())
	got, err := client.Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if got.Temperature != 22.4 {
		t.Fatalf("temperature = %v", got.Temperature)
	}
	if got.Humidity != 61 {
		t.Fatalf("humidity = %v", got.Humidity)
	}
	if got.SoilMoisture != 39 {
		t.Fatalf("soil moisture = %v", got.SoilMoisture)
	}
	if got.Illuminance != 8700 {
		t.Fatalf("illuminance = %v", got.Illuminance)
	}
	if got.LastUpdated != "2026-04-02T10:00:00Z" {
		t.Fatalf("lastUpdated = %q", got.LastUpdated)
	}
	if got.Source != "live" {
		t.Fatalf("source = %q", got.Source)
	}
}
