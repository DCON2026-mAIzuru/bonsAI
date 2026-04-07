package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ServerAddr       string
	StaticDir        string
	SensorAPIURL     string
	LLMChatStreamURL string
	LLMModel         string
	PublicAPIBase    string
}

func Load() Config {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	defaultStaticDir := filepath.Clean(filepath.Join(wd, "..", "bonsAI_front", "dist"))

	return Config{
		ServerAddr:       envOrDefault("BONSAI_SERVER_ADDR", ":8080"),
		StaticDir:        envOrDefault("BONSAI_STATIC_DIR", defaultStaticDir),
		SensorAPIURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("BONSAI_SENSOR_API_URL")), "/"),
		LLMChatStreamURL: strings.TrimSpace(os.Getenv("BONSAI_LLM_CHAT_STREAM_URL")),
		LLMModel:         envOrDefault("BONSAI_LLM_MODEL", "gemma-4-e2b-it"),
		PublicAPIBase:    strings.TrimSpace(os.Getenv("BONSAI_PUBLIC_API_BASE")),
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
