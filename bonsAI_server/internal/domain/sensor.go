package domain

type SensorSnapshot struct {
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	SoilMoisture float64 `json:"soilMoisture"`
	Illuminance  float64 `json:"illuminance"`
	LastUpdated  string  `json:"lastUpdated"`
	Source       string  `json:"source"`
}

func DemoSensorSnapshot() SensorSnapshot {
	return SensorSnapshot{
		Temperature:  24.6,
		Humidity:     58,
		SoilMoisture: 43,
		Illuminance:  12800,
		LastUpdated:  "just now",
		Source:       "demo",
	}
}
