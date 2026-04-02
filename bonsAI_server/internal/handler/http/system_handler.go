package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/config"
)

type SystemHandler struct {
	config config.Config
}

func NewSystemHandler(cfg config.Config) *SystemHandler {
	return &SystemHandler{config: cfg}
}

func (h *SystemHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *SystemHandler) RuntimeConfig(c *gin.Context) {
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.String(http.StatusOK, fmt.Sprintf("window.__BONSAI_CONFIG__ = {\n  apiBase: %q\n};\n", h.config.PublicAPIBase))
}
