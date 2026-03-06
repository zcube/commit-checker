package langdetect_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/langdetect"
)

// ---- LocaleToLanguage coverage ---------------------------------------------

func TestLocaleToLanguage_AllMappings(t *testing.T) {
	cases := []struct {
		locale string
		want   string
	}{
		{"ko", "korean"},
		{"en", "english"},
		{"ja", "japanese"},
		{"zh", "chinese"},
		{"zh-hans", "chinese"},
		{"zh-hant", "chinese"},
		{"unknown", ""},
		{"", ""},
		{"fr", ""},
		{"KO", ""}, // case-sensitive: only lowercase supported
	}
	for _, tc := range cases {
		got := langdetect.LocaleToLanguage(tc.locale)
		if got != tc.want {
			t.Errorf("langdetect.LocaleToLanguage(%q) = %q, want %q", tc.locale, got, tc.want)
		}
	}
}

// ---- HasNaturalLanguageContent coverage ------------------------------------

func TestHasNaturalLanguageContent_TooFewLetters(t *testing.T) {
	// Text with fewer letter-class chars than minLetters
	ok := langdetect.HasNaturalLanguageContent("123!@#", 5, nil)
	if ok {
		t.Error("text with no letters should return false")
	}
}

func TestHasNaturalLanguageContent_ExactlyMinLength_Runes(t *testing.T) {
	// 5 Korean letters — exactly at min_length
	ok := langdetect.HasNaturalLanguageContent("가나다라마", 5, nil)
	if !ok {
		t.Error("5 Korean letters should pass min_length=5")
	}
}

func TestHasNaturalLanguageContent_BelowMinLengthRunes(t *testing.T) {
	// 3 Korean letters < min_length=5
	ok := langdetect.HasNaturalLanguageContent("가나다", 5, nil)
	if ok {
		t.Error("3 Korean letters should fail min_length=5")
	}
}

func TestHasNaturalLanguageContent_BuiltinSkipPrefixes(t *testing.T) {
	skipped := []string{
		"TODO: fix this later",
		"FIXME: broken",
		"nolint:errcheck",
		"noqa: E501",
		"nosec: G401",
		"go:generate protoc",
		"go:build linux",
		"eslint-disable-next-line",
		"@param data the input",
		"@return result",
		"http://example.com",
		"https://example.com",
	}
	for _, s := range skipped {
		ok := langdetect.HasNaturalLanguageContent(s, 5, nil)
		if ok {
			t.Errorf("langdetect.HasNaturalLanguageContent(%q) = true, want false (builtin skip)", s)
		}
	}
}

func TestHasNaturalLanguageContent_ExtraSkip(t *testing.T) {
	ok := langdetect.HasNaturalLanguageContent("internal: this is skipped", 5, []string{"internal:"})
	if ok {
		t.Error("extra skip prefix 'internal:' should suppress content")
	}
}

func TestHasNaturalLanguageContent_ExtraSkipCaseInsensitive(t *testing.T) {
	ok := langdetect.HasNaturalLanguageContent("INTERNAL: this is skipped", 5, []string{"internal:"})
	if ok {
		t.Error("extra skip prefix check should be case-insensitive")
	}
}

// ---- Dominant coverage -----------------------------------------------------

func TestDominant_EmptyString_Unknown(t *testing.T) {
	got := langdetect.Dominant("")
	if got != "unknown" {
		t.Errorf("Dominant(\"\") = %q, want \"unknown\"", got)
	}
}

func TestDominant_NumbersOnly_Unknown(t *testing.T) {
	got := langdetect.Dominant("12345 !@#$%")
	if got != "unknown" {
		t.Errorf("Dominant(numbers only) = %q, want \"unknown\"", got)
	}
}

func TestDominant_MixedKoreanMajority(t *testing.T) {
	// Korean chars dominate
	got := langdetect.Dominant("안녕하세요 hello")
	if got != "korean" {
		t.Errorf("expected korean to dominate, got %q", got)
	}
}

// ---- IsRequiredLanguage edge cases -----------------------------------------

func TestIsRequiredLanguage_PunctuationOnly_NoContent(t *testing.T) {
	ok, hasContent := langdetect.IsRequiredLanguage("...---!!!", "korean", 5, nil)
	if hasContent {
		t.Error("punctuation only should have no content")
	}
	if !ok {
		t.Error("punctuation only should not fail (ok should be true when no content)")
	}
}

func TestIsRequiredLanguage_MixedKoreanEnglish_PassesKorean(t *testing.T) {
	// Mixed text: Korean dominant — should pass Korean check
	ok, hasContent := langdetect.IsRequiredLanguage("변수 name을 초기화합니다", "korean", 5, nil)
	if !hasContent {
		t.Error("expected hasContent=true")
	}
	if !ok {
		t.Error("mixed Korean/English should pass Korean check when Korean is present")
	}
}

func TestIsRequiredLanguage_EnglishOnly_FailsKorean(t *testing.T) {
	ok, hasContent := langdetect.IsRequiredLanguage("pure english text here", "korean", 5, nil)
	if !hasContent {
		t.Error("expected hasContent=true for English text")
	}
	if ok {
		t.Error("English-only text should fail Korean requirement")
	}
}
