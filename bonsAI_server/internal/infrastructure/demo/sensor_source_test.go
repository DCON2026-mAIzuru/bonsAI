package demo

import (
	"testing"
	"time"
)

func TestSensorSourceCurrentChangesEveryTenSeconds(t *testing.T) {
	t.Parallel()

	source := NewSensorSource(func() time.Time {
		return time.Unix(20, 0)
	})

	got, err := source.Current(t.Context())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if got.Temperature != 25.1 {
		t.Fatalf("Temperature = %v", got.Temperature)
	}
	if got.SoilMoisture != 38 {
		t.Fatalf("SoilMoisture = %v", got.SoilMoisture)
	}
	if got.LastUpdated != "1970-01-01T00:00:20Z" {
		t.Fatalf("LastUpdated = %q", got.LastUpdated)
	}
	if got.Source != "demo" {
		t.Fatalf("Source = %q", got.Source)
	}
}

func TestSensorSourceCurrentKeepsSameFrameWithinTenSecondWindow(t *testing.T) {
	t.Parallel()

	firstSource := NewSensorSource(func() time.Time {
		return time.Unix(29, 0)
	})
	secondSource := NewSensorSource(func() time.Time {
		return time.Unix(21, 0)
	})

	first, err := firstSource.Current(t.Context())
	if err != nil {
		t.Fatalf("first Current() error = %v", err)
	}
	second, err := secondSource.Current(t.Context())
	if err != nil {
		t.Fatalf("second Current() error = %v", err)
	}

	if first.Temperature != second.Temperature {
		t.Fatalf("Temperature differs within same slot: %v vs %v", first.Temperature, second.Temperature)
	}
	if first.LastUpdated != second.LastUpdated {
		t.Fatalf("LastUpdated differs within same slot: %q vs %q", first.LastUpdated, second.LastUpdated)
	}
}
