package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/domain"
	"bonsai_server/internal/usecase"
)

type ChatHandler struct {
	chatService *usecase.ChatService
}

func NewChatHandler(chatService *usecase.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) Stream(c *gin.Context) {
	var request domain.ChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	if err := h.chatService.Stream(c.Request.Context(), request, newGinStreamWriter(c)); err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "failed to stream chat"})
	}
}

func (h *ChatHandler) Translate(c *gin.Context) {
	var request domain.ChatTranslationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	translations, err := h.chatService.Translate(c.Request.Context(), request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "failed to translate chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"translations": translations})
}
