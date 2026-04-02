package domain

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message string          `json:"message"`
	History []ChatMessage   `json:"history"`
	Sensors *SensorSnapshot `json:"sensors"`
}
