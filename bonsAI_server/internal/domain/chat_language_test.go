package domain

import "testing"

func TestDetectReplyLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		message string
		want    string
	}{
		{name: "japanese text", message: "水やりは必要？", want: ReplyLanguageJapanese},
		{name: "english text", message: "Does it need water today?", want: ReplyLanguageEnglish},
		{name: "mixed prefers japanese", message: "Waterをあげるべき？", want: ReplyLanguageJapanese},
		{name: "symbols fallback japanese", message: "???", want: ReplyLanguageJapanese},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := DetectReplyLanguage(tc.message); got != tc.want {
				t.Fatalf("DetectReplyLanguage(%q) = %q, want %q", tc.message, got, tc.want)
			}
		})
	}
}

func TestNormalizeReplyLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		language string
		want     string
	}{
		{name: "english", language: "en", want: ReplyLanguageEnglish},
		{name: "english uppercase", language: " EN ", want: ReplyLanguageEnglish},
		{name: "japanese", language: "ja", want: ReplyLanguageJapanese},
		{name: "fallback to japanese", language: "fr", want: ReplyLanguageJapanese},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeReplyLanguage(tc.language); got != tc.want {
				t.Fatalf("NormalizeReplyLanguage(%q) = %q, want %q", tc.language, got, tc.want)
			}
		})
	}
}
