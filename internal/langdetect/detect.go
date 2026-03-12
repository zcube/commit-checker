package langdetect

import (
	"sort"
	"strings"
	"unicode"
)

// Language 는 설정에서 사용되는 자연어 식별자입니다.
type Language = string

const (
	Korean   Language = "korean"
	English  Language = "english"
	Japanese Language = "japanese"
	Chinese  Language = "chinese"
	Any      Language = "any"
)

// LocaleToLanguage 는 BCP-47 로케일 코드를 Language 상수로 변환합니다.
// 인식하지 못하는 로케일이면 "" 를 반환합니다.
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

// builtinSkipPrefixes 는 언어에 관계없이 항상 기술적/지시자 주석으로 처리되어 건너뛰는 접두사 목록입니다.
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

// HasNaturalLanguageContent 는 주석 텍스트가 언어 검사를 수행하기에 충분한 문자를 포함하는지,
// 그리고 순수 지시자가 아닌지 확인합니다.
// extraSkip 은 설정에서 지정한 프로젝트별 추가 접두사입니다.
func HasNaturalLanguageContent(text string, minLetters int, extraSkip []string) bool {
	if len([]rune(text)) < minLetters {
		return false
	}

	// XML/HTML 태그만 있는 주석은 건너뜁니다.
	// C# XML doc (/// <summary>, /// </summary>, /// <param name="x"/> 등)이 대표적입니다.
	if isXMLTagOnly(text) {
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

// isXMLTagOnly 는 텍스트가 XML/HTML 태그만으로 구성되어 있는지 확인합니다.
// C# XML doc 주석에서 CStyleParser 가 "/ " 접두사를 남기므로 이를 제거한 후 확인합니다.
// <summary>, </summary>, <param name="x"/>, <returns/> 등이 해당됩니다.
// 태그 사이에 일반 텍스트가 있으면 false 를 반환합니다.
func isXMLTagOnly(text string) bool {
	s := strings.TrimSpace(text)
	// CStyleParser 는 "///" 에서 "//" 를 제거하여 "/ <tag>" 형태로 남깁니다.
	if strings.HasPrefix(s, "/ ") {
		s = strings.TrimSpace(s[2:])
	} else if len(s) > 0 && s[0] == '/' {
		s = strings.TrimSpace(s[1:])
	}
	if s == "" || s[0] != '<' {
		return false
	}
	// '<' 로 시작하면서 전체가 태그로만 구성되어 있는지 스캔합니다.
	for len(s) > 0 {
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}
		if s[0] != '<' {
			return false // 태그 밖에 텍스트가 있음
		}
		end := strings.IndexByte(s, '>')
		if end == -1 {
			return false // 닫히지 않은 태그
		}
		s = s[end+1:]
	}
	return true
}

// IsRequiredLanguage 는 주석 텍스트가 필수 언어 요건을 충족하는지 확인합니다.
// (ok, hasContent) 반환:
//   - hasContent=false: 너무 짧거나 지시자이므로 건너뜀
//   - ok=false: 언어 검사 실패
func IsRequiredLanguage(text, required Language, minLetters int, extraSkip []string) (ok bool, hasContent bool) {
	if !HasNaturalLanguageContent(text, minLetters, extraSkip) {
		return true, false
	}
	if required == Any || required == "" {
		return true, true
	}

	// 혼합 언어 주석은 필수 언어가 포함되어 있으면 허용합니다.
	// 예: "// 변수 name을 설정합니다" 는 required=korean 일 때 통과.
	if hasScript(text, required) {
		return true, true
	}

	// 필수 언어 문자가 없음. 식별 가능한 언어가 있는지 확인합니다.
	dom := dominant(text)
	if dom == "" {
		return true, false // 구두점/숫자만 있는 경우
	}
	return false, true
}

// hasScript 는 텍스트에 특정 언어의 문자가 하나 이상 포함되는지 확인합니다.
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

// dominant 는 텍스트에서 지배적인 언어 스크립트를 반환합니다. 없으면 "" 를 반환합니다.
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

// Dominant 는 텍스트에서 감지된 지배적 자연어를 반환합니다 (오류 메시지용 공개 함수).
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

// StripAllowedWords 는 텍스트에서 허용 단어 목록에 해당하는 단어를 공백으로 대체합니다.
// 단어 경계 규칙: 앞뒤에 라틴 문자가 없어야 일치로 간주합니다.
// 예) "Java"는 "JavaScript"에서 일치하지 않고, "Java 언어"에서는 일치합니다.
// 긴 단어를 먼저 처리하여 부분 일치를 방지합니다.
func StripAllowedWords(text string, words []string) string {
	if len(words) == 0 {
		return text
	}

	// 긴 단어 먼저 처리 (부분 일치 방지)
	sorted := make([]string, len(words))
	copy(sorted, words)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	runes := []rune(text)
	lower := []rune(strings.ToLower(text))
	skip := make([]bool, len(runes))

	for _, word := range sorted {
		wRunes := []rune(strings.ToLower(word))
		wLen := len(wRunes)
		if wLen == 0 {
			continue
		}
		for i := range len(lower) - wLen + 1 {
			if skip[i] {
				continue
			}
			match := true
			for j := range wLen {
				if lower[i+j] != wRunes[j] {
					match = false
					break
				}
			}
			if !match {
				continue
			}
			// 단어 경계 확인: 앞뒤에 라틴 문자가 없어야 합니다.
			if i > 0 && isLatin(runes[i-1]) {
				continue
			}
			if i+wLen < len(runes) && isLatin(runes[i+wLen]) {
				continue
			}
			for j := range wLen {
				skip[i+j] = true
			}
		}
	}

	var sb strings.Builder
	for i, r := range runes {
		if skip[i] {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
