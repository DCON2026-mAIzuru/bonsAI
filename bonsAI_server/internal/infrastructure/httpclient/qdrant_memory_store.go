package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"bonsai_server/internal/domain"
)

const defaultMemorySessionID = "default"

type QdrantMemoryConfig struct {
	Endpoint    string
	Collection  string
	SearchLimit int
	VectorSize  int
	Client      *http.Client
}

type QdrantMemoryStore struct {
	endpoint    string
	collection  string
	searchLimit int
	vectorSize  int
	client      *http.Client

	mu    sync.Mutex
	ready bool
}

type qdrantQueryResponse struct {
	Result struct {
		Points []qdrantPoint `json:"points"`
	} `json:"result"`
}

type qdrantPoint struct {
	ID      json.RawMessage `json:"id"`
	Score   float64         `json:"score"`
	Payload map[string]any  `json:"payload"`
	Vector  []float64       `json:"vector"`
}

func NewQdrantMemoryStore(cfg QdrantMemoryConfig) *QdrantMemoryStore {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil
	}

	collection := strings.TrimSpace(cfg.Collection)
	if collection == "" {
		collection = "bonsai-memory"
	}

	searchLimit := cfg.SearchLimit
	if searchLimit <= 0 {
		searchLimit = 3
	}

	vectorSize := cfg.VectorSize
	if vectorSize <= 0 {
		vectorSize = 192
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 1500 * time.Millisecond}
	}

	return &QdrantMemoryStore{
		endpoint:    strings.TrimRight(endpoint, "/"),
		collection:  collection,
		searchLimit: searchLimit,
		vectorSize:  vectorSize,
		client:      client,
	}
}

func (s *QdrantMemoryStore) EnsureReady(ctx context.Context) error {
	return s.ensureCollection(ctx)
}

func (s *QdrantMemoryStore) Recall(ctx context.Context, sessionID, message string) ([]domain.ChatMemory, error) {
	if strings.TrimSpace(message) == "" {
		return nil, nil
	}
	if err := s.ensureCollection(ctx); err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]any{
		"query":        embedText(message, s.vectorSize),
		"limit":        s.searchLimit,
		"with_payload": true,
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key": "session_id",
					"match": map[string]any{
						"value": normalizeMemorySessionID(sessionID),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.collectionURL("/points/query"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant query returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var payload qdrantQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	memories := make([]domain.ChatMemory, 0, len(payload.Result.Points))
	for _, point := range payload.Result.Points {
		userMessage := stringPayload(point.Payload, "user_message")
		assistantMessage := stringPayload(point.Payload, "assistant_message")
		if userMessage == "" || assistantMessage == "" {
			continue
		}

		memories = append(memories, domain.ChatMemory{
			SessionID:        stringPayload(point.Payload, "session_id"),
			UserMessage:      userMessage,
			AssistantMessage: assistantMessage,
			CreatedAt:        stringPayload(point.Payload, "created_at"),
			Score:            point.Score,
		})
	}

	return memories, nil
}

func (s *QdrantMemoryStore) ListRecent(ctx context.Context, limit int) ([]domain.ChatMemory, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if err := s.ensureCollection(ctx); err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]any{
		"limit":        limit,
		"with_payload": true,
		"with_vector":  true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.collectionURL("/points/scroll"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant scroll returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var payload qdrantQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	memories := make([]domain.ChatMemory, 0, len(payload.Result.Points))
	for _, point := range payload.Result.Points {
		userMessage := stringPayload(point.Payload, "user_message")
		assistantMessage := stringPayload(point.Payload, "assistant_message")
		if userMessage == "" && assistantMessage == "" {
			continue
		}

		memories = append(memories, domain.ChatMemory{
			SessionID:        stringPayload(point.Payload, "session_id"),
			UserMessage:      userMessage,
			AssistantMessage: assistantMessage,
			CreatedAt:        stringPayload(point.Payload, "created_at"),
			PointID:          rawPointID(point.ID),
			VectorSize:       len(point.Vector),
			VectorPreview:    vectorPreview(point.Vector, 8),
		})
	}

	return memories, nil
}

func (s *QdrantMemoryStore) SaveConversation(ctx context.Context, entry domain.ChatMemoryEntry) error {
	if strings.TrimSpace(entry.UserMessage) == "" || strings.TrimSpace(entry.AssistantMessage) == "" {
		return nil
	}
	if err := s.ensureCollection(ctx); err != nil {
		return err
	}

	normalizedSessionID := normalizeMemorySessionID(entry.SessionID)
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = time.Now().Format(time.RFC3339)
	}

	body, err := json.Marshal(map[string]any{
		"points": []map[string]any{
			{
				"id":     buildMemoryPointID(normalizedSessionID, entry.UserMessage, entry.CreatedAt),
				"vector": embedText(entry.UserMessage+"\n"+entry.AssistantMessage, s.vectorSize),
				"payload": map[string]any{
					"session_id":        normalizedSessionID,
					"user_message":      strings.TrimSpace(entry.UserMessage),
					"assistant_message": strings.TrimSpace(entry.AssistantMessage),
					"created_at":        entry.CreatedAt,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.collectionURL("/points"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant upsert returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (s *QdrantMemoryStore) ensureCollection(ctx context.Context) error {
	s.mu.Lock()
	if s.ready {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	body, err := json.Marshal(map[string]any{
		"vectors": map[string]any{
			"size":     s.vectorSize,
			"distance": "Cosine",
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.collectionURL(""), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		s.mu.Lock()
		s.ready = true
		s.mu.Unlock()
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant create collection returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	return nil
}

func (s *QdrantMemoryStore) collectionURL(suffix string) string {
	return s.endpoint + "/collections/" + url.PathEscape(s.collection) + suffix
}

func normalizeMemorySessionID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return defaultMemorySessionID
	}
	return sessionID
}

func stringPayload(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func rawPointID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	return strings.TrimSpace(string(raw))
}

func vectorPreview(vector []float64, limit int) []float64 {
	if len(vector) == 0 || limit <= 0 {
		return nil
	}
	if len(vector) < limit {
		limit = len(vector)
	}

	preview := make([]float64, limit)
	copy(preview, vector[:limit])
	return preview
}

func buildMemoryPointID(sessionID, userMessage, createdAt string) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(sessionID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(userMessage))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(createdAt))
	return hash.Sum64()
}

func embedText(text string, size int) []float32 {
	cleaned := normalizeEmbeddingText(text)
	if cleaned == "" || size <= 0 {
		return nil
	}

	runes := []rune(cleaned)
	vector := make([]float64, size)
	for n := 1; n <= 3; n++ {
		if len(runes) < n {
			continue
		}
		for i := 0; i <= len(runes)-n; i++ {
			gram := string(runes[i : i+n])
			hash := fnv.New64a()
			_, _ = hash.Write([]byte(gram))
			sum := hash.Sum64()
			index := int(sum % uint64(size))
			value := 1.0
			if sum&1 == 1 {
				value = -1.0
			}
			vector[index] += value
		}
	}

	var norm float64
	for _, value := range vector {
		norm += value * value
	}
	if norm == 0 {
		return nil
	}

	norm = sqrt(norm)
	embedded := make([]float32, len(vector))
	for i, value := range vector {
		embedded[i] = float32(value / norm)
	}

	return embedded
}

func normalizeEmbeddingText(text string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(fields) == 0 {
		return ""
	}

	cleaned := strings.Join(fields, " ")
	builder := strings.Builder{}
	builder.Grow(len(cleaned))

	lastWasSpace := false
	for _, r := range cleaned {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
			lastWasSpace = false
		case unicode.IsSpace(r):
			if !lastWasSpace {
				builder.WriteRune(' ')
				lastWasSpace = true
			}
		default:
			builder.WriteRune(r)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(builder.String())
}

func sqrt(value float64) float64 {
	if value <= 0 {
		return 0
	}

	z := value
	for i := 0; i < 10; i++ {
		z -= (z*z - value) / (2 * z)
	}
	return z
}
