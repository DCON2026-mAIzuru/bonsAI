package main

import (
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
	demoChat := demo.NewChatStreamer(0)
	if cfg.LLMChatStreamURL != "" {
		liveLLMClient := httpclient.NewLLMStreamClient(cfg.LLMChatStreamURL, cfg.LLMModel, http.DefaultClient)
		liveChatStreamer = liveLLMClient
		liveChatTranslator = liveLLMClient
	}
	chatService := usecase.NewChatService(sensorService, liveChatStreamer, demoChat, liveChatTranslator, demoChat)

	systemHandler := httphandler.NewSystemHandler(cfg, nil)
	sensorHandler := httphandler.NewSensorHandler(sensorService)
	chatHandler := httphandler.NewChatHandler(chatService)

	router := httphandler.NewRouter(cfg, systemHandler, sensorHandler, chatHandler)

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

	if err := router.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
