// Package i18n 는 go-i18n 기반의 다국어 메시지 지원을 제공.
// 환경 변수(COMMIT_CHECKER_LANG, LC_ALL, LC_MESSAGES, LANG) 또는
// 명시적 Init 호출로 언어를 설정함.
package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localesFS embed.FS

var (
	bundle   *goi18n.Bundle
	loc      *goi18n.Localizer
	mu       sync.RWMutex
	initOnce sync.Once
)

// Init 는 주어진 언어 태그(예: "ko", "en")로 번역기를 초기화.
// 빈 문자열이면 환경 변수에서 자동 감지.
func Init(lang string) {
	if lang == "" {
		lang = DetectLocale()
	}
	initOnce.Do(func() {
		bundle = goi18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
		_, _ = bundle.LoadMessageFileFS(localesFS, "locales/en.yaml")
		_, _ = bundle.LoadMessageFileFS(localesFS, "locales/ko.yaml")
	})
	mu.Lock()
	loc = goi18n.NewLocalizer(bundle, lang, "en")
	mu.Unlock()
}

// T 는 msgID에 해당하는 번역 문자열을 반환.
// data 는 템플릿에 사용할 필드를 포함하는 map 또는 struct.
// 번역 실패 시 msgID를 그대로 반환.
func T(msgID string, data interface{}) string {
	mu.RLock()
	l := loc
	mu.RUnlock()

	if l == nil {
		Init("")
		mu.RLock()
		l = loc
		mu.RUnlock()
	}

	msg, err := l.Localize(&goi18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		return fmt.Sprintf("[%s]", msgID)
	}
	return msg
}

// DetectLocale 는 환경 변수에서 언어 코드를 감지.
// 우선순위: COMMIT_CHECKER_LANG > LC_ALL > LC_MESSAGES > LANG.
// 지원 언어: ko, en. 감지 실패 시 "en" 반환.
func DetectLocale() string {
	for _, env := range []string{"COMMIT_CHECKER_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := os.Getenv(env); val != "" {
			if tag := parseLocale(val); tag != "" {
				return tag
			}
		}
	}
	return "en"
}

// parseLocale 는 "ko_KR.UTF-8" 형식의 로케일 문자열에서 언어 코드를 추출.
func parseLocale(val string) string {
	val = strings.ToLower(val)
	// 언더스코어나 점 앞부분만 사용
	if idx := strings.IndexAny(val, "_."); idx > 0 {
		val = val[:idx]
	}
	switch val {
	case "ko":
		return "ko"
	case "en", "c", "posix":
		return "en"
	default:
		return "en"
	}
}
