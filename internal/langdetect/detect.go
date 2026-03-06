package langdetect

import (
	"strings"
	"unicode"
)

// Language represents a natural language identifier used in config.
type Language = string

const (
	Korean   Language = "korean"
	English  Language = "english"
	Japanese Language = "japanese"
	Chinese  Language = "chinese"
	Any      Language = "any"
)

// LocaleToLanguage maps a BCP-47 locale code to a Language constant.
// Returns "" if the locale is not recognised.
func LocaleToLanguage(locale string) Language {
	switch locale {
	case "ko":
		return Korean
	case "en":
		return English
	case "ja":
		return Japanese
	case "zh", "zh-hans", "zh-hant":
		return Chinese
	default:
		return ""
	}
}

// builtinSkipPrefixes are comment prefixes that are always treated as
// technical/directive comments and skipped regardless of language.
var builtinSkipPrefixes = []string{
	"todo", "fixme", "hack", "note:", "xxx", "bug",
	"nolint", "noqa", "nosec", "noinspection",
	"go:generate", "go:build", "go:embed", "go:linkname",
	"+build", "eslint-", "tslint:", "prettier-ignore",
	"http://", "https://", "ftp://",
	"@param", "@return", "@throws", "@type", "@deprecated",
	"@ts-ignore", "@ts-nocheck", "@ts-expect-error",
	"suppress warnings",
}

// HasNaturalLanguageContent reports whether the comment text contains enough
// letter characters to warrant a language check, and is not a pure directive.
// extraSkip adds project-specific prefixes from config.
func HasNaturalLanguageContent(text string, minLetters int, extraSkip []string) bool {
	if len([]rune(text)) < minLetters {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(text))
	for _, prefix := range builtinSkipPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	for _, prefix := range extraSkip {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return false
		}
	}

	// Count letter-class characters
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			count++
		}
	}
	return count >= minLetters
}

// IsRequiredLanguage checks whether the comment text satisfies the required language.
// Returns (ok, hasContent):
//   - hasContent=false means the comment was skipped (too short or directive)
//   - ok=false means it failed the language check
func IsRequiredLanguage(text, required Language, minLetters int, extraSkip []string) (ok bool, hasContent bool) {
	if !HasNaturalLanguageContent(text, minLetters, extraSkip) {
		return true, false
	}
	if required == Any || required == "" {
		return true, true
	}

	// Allow mixed-language comments as long as the required language is present.
	// e.g. "// 변수 name을 설정합니다" is OK when required=korean.
	if hasScript(text, required) {
		return true, true
	}

	// No required language script found; check if there's any identifiable content.
	dominant := dominant(text)
	if dominant == "" {
		return true, false // only punctuation / numbers
	}
	return false, true
}

// hasScript reports whether text contains at least one character of the given language's script.
func hasScript(text, lang Language) bool {
	for _, r := range text {
		switch lang {
		case Korean:
			if isKorean(r) {
				return true
			}
		case Japanese:
			if isJapanese(r) {
				return true
			}
		case Chinese:
			if isChinese(r) {
				return true
			}
		case English:
			if isLatin(r) {
				return true
			}
		}
	}
	return false
}

// dominant returns the dominant language script in text, or "" if none.
func dominant(text string) Language {
	var korean, japanese, chinese, latin int
	for _, r := range text {
		switch {
		case isKorean(r):
			korean++
		case isJapanese(r):
			japanese++
		case isChinese(r):
			chinese++
		case isLatin(r):
			latin++
		}
	}
	max := korean
	dom := Language(Korean)
	if japanese > max {
		max = japanese
		dom = Japanese
	}
	if chinese > max {
		max = chinese
		dom = Chinese
	}
	if latin > max {
		max = latin
		dom = English
	}
	if max == 0 {
		return ""
	}
	return dom
}

// Dominant returns the dominant natural language detected in text (exported for error messages).
func Dominant(text string) Language {
	d := dominant(text)
	if d == "" {
		return "unknown"
	}
	return d
}

func isKorean(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
		(r >= 0x1100 && r <= 0x11FF) || // Hangul Jamo
		(r >= 0x3130 && r <= 0x318F) || // Hangul Compatibility Jamo
		(r >= 0xA960 && r <= 0xA97F) || // Hangul Jamo Extended-A
		(r >= 0xD7B0 && r <= 0xD7FF) // Hangul Jamo Extended-B
}

func isJapanese(r rune) bool {
	return (r >= 0x3041 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0x31F0 && r <= 0x31FF) // Katakana Phonetic Extensions
}

func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) // CJK Extension B
}

func isLatin(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
