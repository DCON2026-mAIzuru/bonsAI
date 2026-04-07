package http

import (
	"encoding/json"
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
	handler := NewSystemHandler(config.Config{PublicAPIBase: "/bonsai"}, nil)
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

func TestStatusReportsConnectedWhenLLMHealthResponds(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer llmServer.Close()

	router := gin.New()
	handler := NewSystemHandler(config.Config{
		LLMChatStreamURL: llmServer.URL + "/v1/chat/completions",
	}, llmServer.Client())
	router.GET("/api/system/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/system/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if payload["llmConnected"] != true {
		t.Fatalf("llmConnected = %v", payload["llmConnected"])
	}
}

func TestStatusReportsDisconnectedWhenLLMHealthFails(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer llmServer.Close()

	router := gin.New()
	handler := NewSystemHandler(config.Config{
		LLMChatStreamURL: llmServer.URL + "/v1/chat/completions",
	}, llmServer.Client())
	router.GET("/api/system/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/system/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if payload["llmConnected"] != false {
		t.Fatalf("llmConnected = %v", payload["llmConnected"])
	}
}
