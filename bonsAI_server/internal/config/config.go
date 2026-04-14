package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ServerAddr        string
	StaticDir         string
	SensorAPIURL      string
	LLMChatStreamURL  string
	LLMModel          string
	PublicAPIBase     string
	MemoryQdrantURL   string
	MemoryCollection  string
	MemorySearchLimit int
	MemoryVectorSize  int
}

func Load() Config {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	defaultStaticDir := filepath.Clean(filepath.Join(wd, "..", "bonsAI_front", "dist"))

	return Config{
		ServerAddr:        envOrDefault("BONSAI_SERVER_ADDR", ":8080"),
		StaticDir:         envOrDefault("BONSAI_STATIC_DIR", defaultStaticDir),
		SensorAPIURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("BONSAI_SENSOR_API_URL")), "/"),
		LLMChatStreamURL:  strings.TrimSpace(os.Getenv("BONSAI_LLM_CHAT_STREAM_URL")),
		LLMModel:          envOrDefault("BONSAI_LLM_MODEL", "gemma-4-e2b-it"),
		PublicAPIBase:     strings.TrimSpace(os.Getenv("BONSAI_PUBLIC_API_BASE")),
		MemoryQdrantURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("BONSAI_MEMORY_QDRANT_URL")), "/"),
		MemoryCollection:  envOrDefault("BONSAI_MEMORY_COLLECTION", "bonsai-memory"),
		MemorySearchLimit: envOrDefaultInt("BONSAI_MEMORY_SEARCH_LIMIT", 3),
		MemoryVectorSize:  envOrDefaultInt("BONSAI_MEMORY_VECTOR_SIZE", 192),
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envOrDefaultInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
