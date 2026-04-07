package demo

import (
	"context"
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

func (s *ChatStreamer) Translate(
	_ context.Context,
	request domain.ChatTranslationRequest,
) ([]domain.ChatTranslationResult, error) {
	targetLanguage := domain.NormalizeReplyLanguage(request.TargetLanguage)
	messages := request.Messages
	results := make([]domain.ChatTranslationResult, 0, len(messages))

	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if strings.TrimSpace(message.ID) == "" || content == "" {
			continue
		}

		results = append(results, domain.ChatTranslationResult{
			ID:      message.ID,
			Content: translateDemoText(content, targetLanguage),
		})
	}

	return results, nil
}

func replyChunks(message string, sensors domain.SensorSnapshot) []string {
	replyLanguage := domain.DetectReplyLanguage(message)
	lowerMessage := strings.ToLower(message)
	needsWater := sensors.SoilMoisture < 35
	brightEnough := sensors.Illuminance > 9000

	response := "こんにちは。盆栽の今の気配を見ながら、一緒に様子を整理してみます。"
	if replyLanguage == domain.ReplyLanguageEnglish {
		response = "Hello. Let me look over your bonsai's current condition with you."
	}

	switch {
	case strings.Contains(lowerMessage, "水"),
		strings.Contains(lowerMessage, "乾"),
		strings.Contains(lowerMessage, "water"),
		strings.Contains(lowerMessage, "dry"):
		if replyLanguage == domain.ReplyLanguageEnglish {
			if needsWater {
				response = "The soil moisture is a little low. A small amount of water may be worth considering today while you keep an eye on the surface soil."
			} else {
				response = "The soil still seems to be holding enough moisture. There is probably no need to rush more water right now if the topsoil is not drying too quickly."
			}
		} else if needsWater {
			response = "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。"
		} else {
			response = "土壌水分はまだ保たれています。今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。"
		}
	case strings.Contains(lowerMessage, "元気"),
		strings.Contains(lowerMessage, "状態"),
		strings.Contains(lowerMessage, "healthy"),
		strings.Contains(lowerMessage, "condition"),
		strings.Contains(lowerMessage, "doing"):
		if replyLanguage == domain.ReplyLanguageEnglish {
			if brightEnough {
				response = "The light level looks adequate, and the bonsai seems to be in a fairly calm environment. The balance of temperature, humidity, and soil moisture does not look badly off."
			} else {
				response = "I do not see a major warning sign, but the light looks a bit modest. A slightly brighter spot may make changes in its condition easier to notice."
			}
		} else if brightEnough {
			response = "照度は確保できていて、盆栽は比較的落ち着いた環境にいます。温湿度と土壌水分のバランスも大きく崩れていません。"
		} else {
			response = "強い異常は見えませんが、やや光量が控えめです。置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。"
		}
	case strings.Contains(lowerMessage, "ありがとう"),
		strings.Contains(lowerMessage, "thank you"),
		strings.Contains(lowerMessage, "thanks"):
		if replyLanguage == domain.ReplyLanguageEnglish {
			response = "You're welcome. One of the nicest parts of this bonsai UI is being able to notice small changes together."
		} else {
			response = "どういたしまして。小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。"
		}
	default:
		if replyLanguage == domain.ReplyLanguageEnglish {
			response = fmt.Sprintf(
				"%s Right now it is %.1f C with %.0f%% humidity, %.0f%% soil moisture, and %.0f lx of light. If you like, I can help interpret watering, light, or overall condition from here.",
				response,
				sensors.Temperature,
				sensors.Humidity,
				sensors.SoilMoisture,
				sensors.Illuminance,
			)
		} else {
			response = fmt.Sprintf(
				"%s 現在は%.1f℃、湿度%.0f%%、土壌水分%.0f%%、照度%.0flxです。質問があれば、水やり・日照・体調の見立てに寄せて返答できます。",
				response,
				sensors.Temperature,
				sensors.Humidity,
				sensors.SoilMoisture,
				sensors.Illuminance,
			)
		}
	}

	replacer := strings.NewReplacer("。", "。\n", "、", "、\n", ". ", ". \n")
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

func translateDemoText(content, targetLanguage string) string {
	if domain.DetectReplyLanguage(content) == targetLanguage {
		return content
	}

	if translated, ok := exactDemoTranslation(content, targetLanguage); ok {
		return translated
	}

	if targetLanguage == domain.ReplyLanguageEnglish {
		replacer := strings.NewReplacer(
			"こんにちは。", "Hello. ",
			"盆栽の今の気配を見ながら、一緒に様子を整理してみます。", "Let me look over your bonsai's current condition with you.",
			"土壌水分が少し低めです。", "The soil moisture is a little low. ",
			"今日は少量ずつ様子を見ながら水やりを検討してよさそうです。", "A small amount of water may be worth considering today while you keep an eye on the surface soil.",
			"土壌水分はまだ保たれています。", "The soil still seems to be holding enough moisture. ",
			"今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。", "There is probably no need to rush more water right now if the topsoil is not drying too quickly.",
			"照度は確保できていて、盆栽は比較的落ち着いた環境にいます。", "The light level looks adequate, and the bonsai seems to be in a fairly calm environment. ",
			"温湿度と土壌水分のバランスも大きく崩れていません。", "The balance of temperature, humidity, and soil moisture does not look badly off.",
			"強い異常は見えませんが、やや光量が控えめです。", "I do not see a major warning sign, but the light looks a bit modest. ",
			"置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。", "A slightly brighter spot may make changes in its condition easier to notice.",
			"どういたしまして。", "You're welcome. ",
			"小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。", "One of the nicest parts of this bonsai UI is being able to notice small changes together.",
			"今日の様子を教えて", "How is it doing today?",
			"水やりは必要？", "Does it need water?",
			"光は足りていそう？", "Is it getting enough light?",
		)
		return strings.TrimSpace(replacer.Replace(content))
	}

	replacer := strings.NewReplacer(
		"Hello. Let me look over your bonsai's current condition with you.", "こんにちは。盆栽の今の気配を見ながら、一緒に様子を整理してみます。",
		"The soil moisture is a little low. A small amount of water may be worth considering today while you keep an eye on the surface soil.", "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。",
		"The soil still seems to be holding enough moisture. There is probably no need to rush more water right now if the topsoil is not drying too quickly.", "土壌水分はまだ保たれています。今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。",
		"The light level looks adequate, and the bonsai seems to be in a fairly calm environment. The balance of temperature, humidity, and soil moisture does not look badly off.", "照度は確保できていて、盆栽は比較的落ち着いた環境にいます。温湿度と土壌水分のバランスも大きく崩れていません。",
		"I do not see a major warning sign, but the light looks a bit modest. A slightly brighter spot may make changes in its condition easier to notice.", "強い異常は見えませんが、やや光量が控えめです。置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。",
		"You're welcome. One of the nicest parts of this bonsai UI is being able to notice small changes together.", "どういたしまして。小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。",
		"How is it doing today?", "今日の様子を教えて",
		"Does it need water?", "水やりは必要？",
		"Is it getting enough light?", "光は足りていそう？",
	)
	return strings.TrimSpace(replacer.Replace(content))
}

func exactDemoTranslation(content, targetLanguage string) (string, bool) {
	translations := map[string]map[string]string{
		"今日の様子を教えて": {
			domain.ReplyLanguageEnglish: "How is it doing today?",
		},
		"水やりは必要？": {
			domain.ReplyLanguageEnglish: "Does it need water?",
		},
		"光は足りていそう？": {
			domain.ReplyLanguageEnglish: "Is it getting enough light?",
		},
		"How is it doing today?": {
			domain.ReplyLanguageJapanese: "今日の様子を教えて",
		},
		"Does it need water?": {
			domain.ReplyLanguageJapanese: "水やりは必要？",
		},
		"Is it getting enough light?": {
			domain.ReplyLanguageJapanese: "光は足りていそう？",
		},
	}

	targets, ok := translations[content]
	if !ok {
		return "", false
	}

	translated, ok := targets[targetLanguage]
	return translated, ok
}
