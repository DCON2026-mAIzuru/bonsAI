package sseutil

import (
	"encoding/json"
	"fmt"
)

func MarshalSSE(event string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, body)), nil
}
