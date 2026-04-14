package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/config"
	"bonsai_server/internal/domain"
	httphandler "bonsai_server/internal/handler/http"
	"bonsai_server/internal/infrastructure/demo"
	"bonsai_server/internal/infrastructure/httpclient"
	"bonsai_server/internal/usecase"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	cfg := config.Load()

	var sensorSource domain.SensorSource
	if cfg.SensorAPIURL != "" {
		sensorSource = httpclient.NewSensorAPIClient(cfg.SensorAPIURL, &http.Client{Timeout: 4 * time.Second})
	} else {
		sensorSource = demo.NewSensorSource(nil)
	}
	sensorService := usecase.NewSensorService(sensorSource)

	var liveChatStreamer domain.ChatStreamer
	var liveChatTranslator domain.ChatTranslator
	var memoryStore domain.ChatMemoryStore
	demoChat := demo.NewChatStreamer(0)
	if cfg.LLMChatStreamURL != "" {
		liveLLMClient := httpclient.NewLLMStreamClient(cfg.LLMChatStreamURL, cfg.LLMModel, http.DefaultClient)
		liveChatStreamer = liveLLMClient
		liveChatTranslator = liveLLMClient
	}
	if cfg.MemoryQdrantURL != "" {
		memoryStore = httpclient.NewQdrantMemoryStore(httpclient.QdrantMemoryConfig{
			Endpoint:    cfg.MemoryQdrantURL,
			Collection:  cfg.MemoryCollection,
			SearchLimit: cfg.MemorySearchLimit,
			VectorSize:  cfg.MemoryVectorSize,
			Client: &http.Client{
				Timeout: 1500 * time.Millisecond,
			},
		})
		if err := memoryStore.EnsureReady(context.Background()); err != nil {
			log.Printf("memory store warm-up failed, continuing in fail-soft mode: %v", err)
		}
	}
	chatService := usecase.NewChatService(sensorService, liveChatStreamer, demoChat, liveChatTranslator, demoChat, memoryStore)

	systemHandler := httphandler.NewSystemHandler(cfg, nil)
	sensorHandler := httphandler.NewSensorHandler(sensorService)
	chatHandler := httphandler.NewChatHandler(chatService)
	memoryHandler := httphandler.NewMemoryHandler(memoryStore)

	router := httphandler.NewRouter(cfg, systemHandler, sensorHandler, chatHandler, memoryHandler)

	log.Printf("bonsAI_server listening on %s", cfg.ServerAddr)
	log.Printf("serving static files from %s", cfg.StaticDir)
	if cfg.SensorAPIURL == "" {
		log.Printf("sensor api: demo mode")
	} else {
		log.Printf("sensor api: %s", cfg.SensorAPIURL)
	}
	if cfg.LLMChatStreamURL == "" {
		log.Printf("llm stream: not configured")
	} else {
		log.Printf("llm stream: %s (model=%s)", cfg.LLMChatStreamURL, cfg.LLMModel)
	}
	if cfg.MemoryQdrantURL == "" {
		log.Printf("memory store: disabled")
	} else {
		log.Printf(
			"memory store: %s (collection=%s limit=%d dim=%d)",
			cfg.MemoryQdrantURL,
			cfg.MemoryCollection,
			cfg.MemorySearchLimit,
			cfg.MemoryVectorSize,
		)
	}

	if err := router.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
