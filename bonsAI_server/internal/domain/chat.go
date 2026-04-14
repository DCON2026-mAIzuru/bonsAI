package domain

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	SessionID string          `json:"sessionId"`
	Message   string          `json:"message"`
	History   []ChatMessage   `json:"history"`
	Sensors   *SensorSnapshot `json:"sensors"`
	Memories  []ChatMemory    `json:"-"`
}

type ChatTranslationMessage struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatTranslationRequest struct {
	Messages       []ChatTranslationMessage `json:"messages"`
	TargetLanguage string                   `json:"targetLanguage"`
}

type ChatTranslationResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}
