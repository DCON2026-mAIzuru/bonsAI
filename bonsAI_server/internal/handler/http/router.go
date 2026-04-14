package http

import (
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/config"
)

func NewRouter(
	cfg config.Config,
	systemHandler *SystemHandler,
	sensorHandler *SensorHandler,
	chatHandler *ChatHandler,
	memoryHandler *MemoryHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), requestLogger())

	router.GET("/healthz", systemHandler.Healthz)
	router.GET("/api/system/status", systemHandler.Status)
	router.GET("/runtime-config.js", systemHandler.RuntimeConfig)
	router.GET("/api/sensors", sensorHandler.GetCurrent)
	router.POST("/api/chat/stream", chatHandler.Stream)
	router.POST("/api/chat/translate", chatHandler.Translate)
	router.GET("/api/memories", memoryHandler.List)
	router.NoRoute(spaFallback(cfg.StaticDir))

	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		gin.DefaultWriter.Write([]byte(
			c.Request.Method + " " + c.Request.URL.Path + " " + time.Since(start).Round(time.Millisecond).String() + "\n",
		))
	}
}

func spaFallback(staticDir string) gin.HandlerFunc {
	indexPath := filepath.Join(staticDir, "index.html")

	return func(c *gin.Context) {
		if c.Request.Method != stdhttp.MethodGet && c.Request.Method != stdhttp.MethodHead {
			c.Status(stdhttp.StatusNotFound)
			return
		}

		requestPath := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), "/")
		if requestPath != "" && requestPath != "." {
			fullPath := filepath.Join(staticDir, requestPath)
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				c.File(fullPath)
				return
			}
		}

		c.File(indexPath)
	}
}
