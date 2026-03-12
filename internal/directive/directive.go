// commit-checker:file-lang=any
// Package directive 는 소스 코드 주석에 삽입된 commit-checker 인라인 지시자를 파싱합니다.
// 지시자는 "commit-checker:" 접두사(대소문자 무관)로 시작하는 주석 텍스트입니다.
//
// 지원 지시자
//
//	- commit-checker:disable            여기서부터 언어 검사 비활성화
//	- commit-checker:disable:lang=<L>   비활성화 + 언어 L 로 교체
//	- commit-checker:enable             언어 검사 재활성화
//	- commit-checker:ignore             바로 다음 주석 건너뜀
//	- commit-checker:lang=<L>           여기서부터 필수 언어를 L 로 변경
//	- commit-checker:file-lang=<L>      파일 전체의 필수 언어 설정
//
// <L> 은 required_language 와 같은 값(korean, english, japanese, chinese, any) 및
// 로케일 코드(ko, en, ja, zh, zh-hans, zh-hant)를 허용합니다.
package directive

import (
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/langdetect"
)

const prefix = "commit-checker:"

// CommentState 는 지시자 처리 후 각 주석을 어떻게 처리할지 설명합니다.
type CommentState struct {
	// Skip 은 해당 주석을 언어 검사하지 않아야 할 때 true 입니다.
	Skip bool
	// Language 는 이 주석에 적용할 필수 언어입니다.
	// 빈 문자열은 "호출자의 기본값 사용"을 의미합니다.
	Language string
}

// Analyze 는 소스 순서로 주석을 순회하며 각 주석에 대한 CommentState 를 반환합니다.
// defaultLang 은 설정에서 이미 결정된 파일별 필수 언어입니다.
func Analyze(comments []comment.Comment, defaultLang string) []CommentState {
	states := make([]CommentState, len(comments))

	disabled := false    // commit-checker:disable 활성 여부
	disabledLang := ""   // 비활성 중 언어 재정의 (빈 문자열 = 완전히 건너뜀)
	skipNext := false    // commit-checker:ignore 감지 시 다음 주석 건너뜀
	langOverride := ""   // commit-checker:lang= 재정의 (빈 문자열 = defaultLang 사용)
	fileLang := ""       // commit-checker:file-lang= 이 파일 전체에 적용

	for i, c := range comments {
		text := strings.TrimSpace(c.Text)

		if !isDirective(text) {
			if fileLang != "" {
				// file-lang 은 활성 disable 을 제외한 모든 것을 재정의
				if disabled {
					states[i] = CommentState{Skip: disabledLang == "", Language: disabledLang}
				} else if skipNext {
					states[i] = CommentState{Skip: true}
					skipNext = false
				} else {
					lang := fileLang
					if langOverride != "" {
						lang = langOverride
					}
					states[i] = CommentState{Language: lang}
				}
			} else if disabled {
				states[i] = CommentState{Skip: disabledLang == "", Language: disabledLang}
				skipNext = false
			} else if skipNext {
				states[i] = CommentState{Skip: true}
				skipNext = false
			} else {
				states[i] = CommentState{Language: langOverride}
			}
			continue
		}

		// 지시자 자체는 항상 검사 대상에서 제외됩니다.
		states[i] = CommentState{Skip: true}

		lower := strings.ToLower(text)

		switch {
		case strings.HasPrefix(lower, prefix+"file-lang="):
			fileLang = resolveLanguage(text[len(prefix+"file-lang="):])

		case strings.HasPrefix(lower, prefix+"disable:lang="):
			disabled = true
			disabledLang = resolveLanguage(text[len(prefix+"disable:lang="):])

		case strings.HasPrefix(lower, prefix+"disable"):
			disabled = true
			disabledLang = ""

		case strings.HasPrefix(lower, prefix+"enable"):
			disabled = false
			disabledLang = ""

		case strings.HasPrefix(lower, prefix+"ignore"):
			skipNext = true

		case strings.HasPrefix(lower, prefix+"lang="):
			langOverride = resolveLanguage(text[len(prefix+"lang="):])
		}
	}

	// 빈 Language 필드를 defaultLang 으로 채웁니다.
	for i := range states {
		if !states[i].Skip && states[i].Language == "" {
			states[i].Language = defaultLang
		}
	}

	return states
}

// IsDirective 는 주석 텍스트가 commit-checker 지시자인지 확인합니다.
func IsDirective(text string) bool {
	return isDirective(strings.TrimSpace(text))
}

func isDirective(text string) bool {
	return strings.HasPrefix(strings.ToLower(text), prefix)
}

// resolveLanguage 는 언어 값을 정규화합니다: 로케일 코드(ko, en, ja, zh 등)는
// 전체 이름으로 매핑되고, 알 수 없는 값은 소문자로 그대로 반환됩니다.
func resolveLanguage(raw string) string {
	raw = strings.TrimSpace(raw)
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(raw)); mapped != "" {
		return mapped
	}
	return strings.ToLower(raw)
}
