package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/config"
)

type SystemHandler struct {
	config config.Config
	client *http.Client
}

func NewSystemHandler(cfg config.Config, client *http.Client) *SystemHandler {
	if client == nil {
		client = &http.Client{Timeout: 1500 * time.Millisecond}
	}

	return &SystemHandler{
		config: cfg,
		client: client,
	}
}

func (h *SystemHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *SystemHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"llmConnected": h.llmConnected(c.Request.Context()),
	})
}

func (h *SystemHandler) RuntimeConfig(c *gin.Context) {
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf("window.__BONSAI_CONFIG__ = {\n  apiBase: %q\n};\n", h.config.PublicAPIBase))
}

func (h *SystemHandler) llmConnected(ctx context.Context) bool {
	healthURL, ok := llmHealthURL(h.config.LLMChatStreamURL)
	if !ok {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func llmHealthURL(chatStreamURL string) (string, bool) {
	if chatStreamURL == "" {
		return "", false
	}

	parsed, err := url.Parse(chatStreamURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	parsed.Path = "/health"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), true
}
