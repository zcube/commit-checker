package langdetect

import (
	"strings"
	"unicode"
)

// Language: 설정에서 사용되는 자연어 식별자.
type Language = string

const (
	Korean   Language = "korean"
	English  Language = "english"
	Japanese Language = "japanese"
	Chinese  Language = "chinese"
	Any      Language = "any"
)

// LocaleToLanguage: BCP-47 로케일 코드를 Language 상수로 매핑.
// 인식되지 않는 로케일이면 ""을 반환.
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

// builtinSkipPrefixes: 언어와 무관하게 항상 기술적/디렉티브 주석으로
// 간주하여 건너뛰는 주석 접두사 목록.
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

// HasNaturalLanguageContent: 주석 텍스트에 언어 검사가 필요한 충분한
// 문자 문자가 포함되어 있는지, 순수 디렉티브가 아닌지 확인.
// extraSkip은 설정의 프로젝트별 접두사를 추가.
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

	// 문자 클래스 문자 수 계산
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			count++
		}
	}
	return count >= minLetters
}

// IsRequiredLanguage: 주석 텍스트가 필수 언어를 충족하는지 검사.
// 반환값 (ok, hasContent):
//   - hasContent=false: 주석이 건너뛰어짐 (너무 짧거나 디렉티브)
//   - ok=false: 언어 검사 실패
func IsRequiredLanguage(text, required Language, minLetters int, extraSkip []string) (ok bool, hasContent bool) {
	if !HasNaturalLanguageContent(text, minLetters, extraSkip) {
		return true, false
	}
	if required == Any || required == "" {
		return true, true
	}

	// 필수 언어가 포함되어 있으면 혼합 언어 주석도 허용.
	// 예: "// 변수 name을 설정합니다"는 required=korean일 때 통과.
	if hasScript(text, required) {
		return true, true
	}

	// 필수 언어 스크립트를 찾지 못함; 식별 가능한 내용이 있는지 확인.
	dominant := dominant(text)
	if dominant == "" {
		return true, false // 구두점 / 숫자만 존재
	}
	return false, true
}

// hasScript: 텍스트에 주어진 언어 스크립트의 문자가 하나 이상 포함되어 있는지 확인.
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

// dominant: 텍스트에서 가장 많은 언어 스크립트를 반환. 없으면 "".
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

// Dominant: 텍스트에서 감지된 주요 자연어를 반환 (에러 메시지용 외부 노출).
func Dominant(text string) Language {
	d := dominant(text)
	if d == "" {
		return "unknown"
	}
	return d
}

func isKorean(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7A3) || // 한글 음절
		(r >= 0x1100 && r <= 0x11FF) || // 한글 자모
		(r >= 0x3130 && r <= 0x318F) || // 한글 호환 자모
		(r >= 0xA960 && r <= 0xA97F) || // 한글 자모 확장-A
		(r >= 0xD7B0 && r <= 0xD7FF) // 한글 자모 확장-B
}

func isJapanese(r rune) bool {
	return (r >= 0x3041 && r <= 0x309F) || // 히라가나
		(r >= 0x30A0 && r <= 0x30FF) || // 가타카나
		(r >= 0x31F0 && r <= 0x31FF) // 가타카나 음성 확장
}

func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK 통합 한자
		(r >= 0x3400 && r <= 0x4DBF) || // CJK 확장 A
		(r >= 0x20000 && r <= 0x2A6DF) // CJK 확장 B
}

func isLatin(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
