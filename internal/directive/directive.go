// commit-checker:disable
//
// Package directive: 소스 코드 주석에 포함된 commit-checker 인라인 디렉티브를 파싱.
// 디렉티브는 "commit-checker:" 접두사(대소문자 무관)로 시작하는 주석 텍스트.
//
// 지원되는 디렉티브:
//   - disable: 이 지점부터 언어 검사 비활성화
//   - disable:lang=<언어>: 기본 비활성화 후 지정 언어 사용
//   - enable: 언어 검사 재활성화
//   - ignore: 바로 다음 주석 건너뜀
//   - lang=<언어>: 이 지점부터 필수 언어 전환
//   - file-lang=<언어>: 파일 전체의 필수 언어 설정
//
// 언어 값은 korean, english, japanese, chinese, any 또는
// 로케일 코드 (ko, en, ja, zh, zh-hans, zh-hant) 사용 가능.
//
// commit-checker:enable
package directive

import (
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/langdetect"
)

const prefix = "commit-checker:"

// CommentState: 디렉티브 처리 후 개별 주석의 처리 방식을 기술.
type CommentState struct {
	// Skip: true이면 해당 주석의 언어 검사를 완전히 건너뜀.
	Skip bool
	// Language: 이 주석에 적용되는 유효 필수 언어.
	// 빈 문자열이면 "호출자의 기본값 사용"을 의미.
	Language string
}

// Analyze: 주석을 소스 순서대로 순회하며 각각에 대한 CommentState를 반환.
// defaultLang은 파일별 필수 언어 (설정에서 이미 결정됨).
func Analyze(comments []comment.Comment, defaultLang string) []CommentState {
	states := make([]CommentState, len(comments))

	disabled := false    // commit-checker:disable 활성 상태
	disabledLang := ""   // 비활성화 중 언어 오버라이드 (비어 있으면 완전 건너뜀)
	skipNext := false    // commit-checker:ignore 감지; 다음 실제 주석 건너뜀
	langOverride := ""   // commit-checker:lang= 오버라이드 (비어 있으면 defaultLang 사용)
	fileLang := ""       // commit-checker:file-lang= 파일 전체에 설정

	for i, c := range comments {
		text := strings.TrimSpace(c.Text)

		if !isDirective(text) {
			if fileLang != "" {
				// file-lang은 활성 disable을 제외한 모든 것을 오버라이드
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

		// 디렉티브 자체는 항상 검사 대상에서 제외.
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

	// 비어 있는 Language 필드를 defaultLang으로 해결.
	for i := range states {
		if !states[i].Skip && states[i].Language == "" {
			states[i].Language = defaultLang
		}
	}

	return states
}

// IsDirective: 주석 텍스트가 commit-checker 디렉티브인지 확인.
func IsDirective(text string) bool {
	return isDirective(strings.TrimSpace(text))
}

func isDirective(text string) bool {
	return strings.HasPrefix(strings.ToLower(text), prefix)
}

// resolveLanguage: 언어 값을 정규화 — 로케일 코드(ko, en, ja, zh…)를
// 전체 이름으로 매핑; 알 수 없는 값은 소문자화하여 그대로 반환.
func resolveLanguage(raw string) string {
	raw = strings.TrimSpace(raw)
	if mapped := langdetect.LocaleToLanguage(strings.ToLower(raw)); mapped != "" {
		return mapped
	}
	return strings.ToLower(raw)
}
