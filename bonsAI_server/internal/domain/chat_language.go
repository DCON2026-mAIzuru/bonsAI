package domain

import (
	"strings"
	"unicode"
)

const (
	ReplyLanguageJapanese = "ja"
	ReplyLanguageEnglish  = "en"
)

func DetectReplyLanguage(message string) string {
	hasLatin := false

	for _, r := range message {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han):
			return ReplyLanguageJapanese
		case unicode.In(r, unicode.Latin):
			hasLatin = true
		}
	}

	if hasLatin {
		return ReplyLanguageEnglish
	}

	return ReplyLanguageJapanese
}

func NormalizeReplyLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case ReplyLanguageEnglish:
		return ReplyLanguageEnglish
	default:
		return ReplyLanguageJapanese
	}
}
