package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"bonsai_server/internal/config"
)

func TestRuntimeConfigUsesConfiguredAPIBase(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewSystemHandler(config.Config{PublicAPIBase: "/bonsai"})
	router.GET("/runtime-config.js", handler.RuntimeConfig)

	req := httptest.NewRequest(http.MethodGet, "/runtime-config.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `apiBase: "/bonsai"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
