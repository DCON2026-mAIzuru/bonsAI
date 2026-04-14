package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/domain"
)

type MemoryHandler struct {
	store domain.ChatMemoryStore
}

func NewMemoryHandler(store domain.ChatMemoryStore) *MemoryHandler {
	return &MemoryHandler{store: store}
}

func (h *MemoryHandler) List(c *gin.Context) {
	if h.store == nil {
		c.JSON(http.StatusOK, gin.H{"memories": []domain.ChatMemory{}})
		return
	}

	limit := 20
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}

	memories, err := h.store.ListRecent(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to load memories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"memories": memories})
}
