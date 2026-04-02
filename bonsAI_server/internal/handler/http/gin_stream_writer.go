package http

import (
	"github.com/gin-gonic/gin"

	"bonsai_server/internal/domain"
	"bonsai_server/internal/sseutil"
)

type ginStreamWriter struct {
	ctx *gin.Context
}

func newGinStreamWriter(ctx *gin.Context) domain.StreamWriter {
	return &ginStreamWriter{ctx: ctx}
}

func (w *ginStreamWriter) SetHeader(key, value string) {
	w.ctx.Header(key, value)
}

func (w *ginStreamWriter) WriteHeader(status int) {
	w.ctx.Status(status)
}

func (w *ginStreamWriter) WriteChunk(chunk []byte) (int, error) {
	return w.ctx.Writer.Write(chunk)
}

func (w *ginStreamWriter) WriteEvent(event string, payload any) error {
	body, err := sseutil.MarshalSSE(event, payload)
	if err != nil {
		return err
	}
	_, err = w.WriteChunk(body)
	return err
}

func (w *ginStreamWriter) Flush() {
	w.ctx.Writer.Flush()
}
