package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bonsai_server/internal/domain"
)

func TestQdrantMemoryStoreSaveConversationTreatsExistingCollectionAsReady(t *testing.T) {
	t.Parallel()

	createCalls := 0
	upsertCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/test-memory":
			createCalls++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"status":{"error":"already exists"}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/collections/test-memory/points":
			upsertCalls++

			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode upsert request: %v", err)
			}

			points, ok := payload["points"].([]any)
			if !ok || len(points) != 1 {
				t.Fatalf("points = %#v", payload["points"])
			}

			point := points[0].(map[string]any)
			if _, ok := point["id"].(float64); !ok {
				t.Fatalf("point id should be numeric, got %#v", point["id"])
			}

			pointPayload := point["payload"].(map[string]any)
			if pointPayload["session_id"] != defaultMemorySessionID {
				t.Fatalf("session_id = %#v", pointPayload["session_id"])
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantMemoryStore(QdrantMemoryConfig{
		Endpoint:    server.URL,
		Collection:  "test-memory",
		SearchLimit: 3,
		VectorSize:  16,
		Client:      server.Client(),
	})

	err := store.SaveConversation(t.Context(), domain.ChatMemoryEntry{
		UserMessage:      "水やりは必要？",
		AssistantMessage: "今日は急がなくて大丈夫そうです。",
	})
	if err != nil {
		t.Fatalf("SaveConversation() error = %v", err)
	}

	if createCalls != 1 {
		t.Fatalf("createCalls = %d", createCalls)
	}
	if upsertCalls != 1 {
		t.Fatalf("upsertCalls = %d", upsertCalls)
	}
}

func TestQdrantMemoryStoreRecallUsesSessionFilter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/test-memory":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/collections/test-memory/points/query":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode query request: %v", err)
			}

			filter := payload["filter"].(map[string]any)
			must := filter["must"].([]any)
			match := must[0].(map[string]any)["match"].(map[string]any)
			if match["value"] != "session-1" {
				t.Fatalf("filter session value = %#v", match["value"])
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"result": {
					"points": [
						{
							"score": 0.88,
							"payload": {
								"session_id": "session-1",
								"user_message": "昨日は水やりを控えた",
								"assistant_message": "表土を見ながら少し待つ方針でした。",
								"created_at": "2026-04-07T20:00:00+09:00"
							}
						}
					]
				}
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantMemoryStore(QdrantMemoryConfig{
		Endpoint:    server.URL,
		Collection:  "test-memory",
		SearchLimit: 3,
		VectorSize:  16,
		Client:      server.Client(),
	})

	memories, err := store.Recall(t.Context(), "session-1", "今日は水やりした方がいい？")
	if err != nil {
		t.Fatalf("Recall() error = %v", err)
	}

	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d", len(memories))
	}
	if memories[0].AssistantMessage != "表土を見ながら少し待つ方針でした。" {
		t.Fatalf("assistant message = %q", memories[0].AssistantMessage)
	}
}

func TestQdrantMemoryStoreListRecentScrollsPayloadAndVectorPreview(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/test-memory":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/collections/test-memory/points/scroll":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode scroll request: %v", err)
			}

			if payload["with_vector"] != true {
				t.Fatalf("with_vector = %#v", payload["with_vector"])
			}
			if payload["limit"] != float64(5) {
				t.Fatalf("limit = %#v", payload["limit"])
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"result": {
					"points": [
						{
							"id": 42,
							"vector": [0.1, -0.2, 0.3],
							"payload": {
								"session_id": "session-1",
								"user_message": "こまめを覚えて",
								"assistant_message": "覚えておきます。",
								"created_at": "2026-04-10T10:00:00+09:00"
							}
						}
					]
				}
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantMemoryStore(QdrantMemoryConfig{
		Endpoint:    server.URL,
		Collection:  "test-memory",
		SearchLimit: 3,
		VectorSize:  16,
		Client:      server.Client(),
	})

	memories, err := store.ListRecent(t.Context(), 5)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}

	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d", len(memories))
	}
	if memories[0].PointID != "42" {
		t.Fatalf("point id = %q", memories[0].PointID)
	}
	if memories[0].VectorSize != 3 {
		t.Fatalf("vector size = %d", memories[0].VectorSize)
	}
	if len(memories[0].VectorPreview) != 3 || memories[0].VectorPreview[1] != -0.2 {
		t.Fatalf("vector preview = %#v", memories[0].VectorPreview)
	}
}
