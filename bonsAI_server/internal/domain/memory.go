package domain

type ChatMemory struct {
	SessionID        string    `json:"sessionId"`
	UserMessage      string    `json:"userMessage"`
	AssistantMessage string    `json:"assistantMessage"`
	CreatedAt        string    `json:"createdAt"`
	Score            float64   `json:"score"`
	PointID          string    `json:"pointId,omitempty"`
	VectorSize       int       `json:"vectorSize,omitempty"`
	VectorPreview    []float64 `json:"vectorPreview,omitempty"`
}

type ChatMemoryEntry struct {
	SessionID        string
	UserMessage      string
	AssistantMessage string
	CreatedAt        string
}
