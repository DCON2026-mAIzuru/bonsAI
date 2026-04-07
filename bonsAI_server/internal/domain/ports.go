package domain

import "context"

type SensorSource interface {
	Current(ctx context.Context) (SensorSnapshot, error)
}

type StreamWriter interface {
	SetHeader(key, value string)
	WriteHeader(status int)
	WriteChunk(chunk []byte) (int, error)
	WriteEvent(event string, payload any) error
	Flush()
}

type ChatStreamer interface {
	Stream(ctx context.Context, request ChatRequest, sensors SensorSnapshot, writer StreamWriter) error
}

type ChatTranslator interface {
	Translate(ctx context.Context, request ChatTranslationRequest) ([]ChatTranslationResult, error)
}
