package usecase

import (
	"strings"
	"sync"

	"bonsai_server/internal/domain"
)

type capturingStreamWriter struct {
	writer  domain.StreamWriter
	mu      sync.Mutex
	builder strings.Builder
}

func newCapturingStreamWriter(writer domain.StreamWriter) *capturingStreamWriter {
	return &capturingStreamWriter{writer: writer}
}

func (w *capturingStreamWriter) SetHeader(key, value string) {
	w.writer.SetHeader(key, value)
}

func (w *capturingStreamWriter) WriteHeader(status int) {
	w.writer.WriteHeader(status)
}

func (w *capturingStreamWriter) WriteChunk(chunk []byte) (int, error) {
	return w.writer.WriteChunk(chunk)
}

func (w *capturingStreamWriter) WriteEvent(event string, payload any) error {
	if event == "message" {
		w.appendDelta(deltaFromPayload(payload))
	}
	return w.writer.WriteEvent(event, payload)
}

func (w *capturingStreamWriter) Flush() {
	w.writer.Flush()
}

func (w *capturingStreamWriter) Content() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.builder.String()
}

func (w *capturingStreamWriter) appendDelta(delta string) {
	if delta == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.builder.WriteString(delta)
}

func deltaFromPayload(payload any) string {
	switch typed := payload.(type) {
	case map[string]any:
		value, _ := typed["delta"].(string)
		return value
	case map[string]string:
		return typed["delta"]
	default:
		return ""
	}
}
