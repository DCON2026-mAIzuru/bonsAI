package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bonsai_server/internal/domain"
)

type ChatStreamer struct {
	chunkDelay time.Duration
}

func NewChatStreamer(chunkDelay time.Duration) *ChatStreamer {
	return &ChatStreamer{chunkDelay: chunkDelay}
}

func (s *ChatStreamer) Stream(_ context.Context, request domain.ChatRequest, sensors domain.SensorSnapshot, writer domain.StreamWriter) error {
	writer.SetHeader("Content-Type", "text/event-stream; charset=utf-8")
	writer.SetHeader("Cache-Control", "no-cache")
	writer.SetHeader("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)

	for _, chunk := range replyChunks(request.Message, sensors) {
		if err := writer.WriteEvent("message", map[string]any{"delta": chunk}); err != nil {
			return err
		}
		writer.Flush()
		time.Sleep(s.chunkDelay)
	}

	if err := writer.WriteEvent("done", map[string]any{"done": true}); err != nil {
		return err
	}
	writer.Flush()
	return nil
}

func replyChunks(message string, sensors domain.SensorSnapshot) []string {
	lowerMessage := strings.ToLower(message)
	needsWater := sensors.SoilMoisture < 35
	brightEnough := sensors.Illuminance > 9000

	response := "こんにちは。盆栽の今の気配を見ながら、一緒に様子を整理してみます。"

	switch {
	case strings.Contains(lowerMessage, "水"), strings.Contains(lowerMessage, "乾"):
		if needsWater {
			response = "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。"
		} else {
			response = "土壌水分はまだ保たれています。今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。"
		}
	case strings.Contains(lowerMessage, "元気"), strings.Contains(lowerMessage, "状態"):
		if brightEnough {
			response = "照度は確保できていて、盆栽は比較的落ち着いた環境にいます。温湿度と土壌水分のバランスも大きく崩れていません。"
		} else {
			response = "強い異常は見えませんが、やや光量が控えめです。置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。"
		}
	case strings.Contains(lowerMessage, "ありがとう"):
		response = "どういたしまして。小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。"
	default:
		response = fmt.Sprintf(
			"%s 現在は%.1f℃、湿度%.0f%%、土壌水分%.0f%%、照度%.0flxです。質問があれば、水やり・日照・体調の見立てに寄せて返答できます。",
			response,
			sensors.Temperature,
			sensors.Humidity,
			sensors.SoilMoisture,
			sensors.Illuminance,
		)
	}

	replacer := strings.NewReplacer("。", "。\n", "、", "、\n")
	parts := strings.Split(replacer.Replace(response), "\n")

	chunks := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		chunks = append(chunks, part)
	}
	return chunks
}

func MarshalSSE(event string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, body)), nil
}
