package i18n

import (
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// loadLocaleKeys 는 embed 된 locale 파일에서 최상위 메시지 키 집합을 추출.
func loadLocaleKeys(t *testing.T, name string) map[string]bool {
	t.Helper()
	data, err := localesFS.ReadFile("locales/" + name + ".yaml")
	if err != nil {
		t.Fatalf("locale 파일 읽기 실패 (%s): %v", name, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("locale 파일 파싱 실패 (%s): %v", name, err)
	}
	keys := make(map[string]bool, len(doc))
	for k := range doc {
		keys[k] = true
	}
	return keys
}

// setLocalizer 는 테스트용으로 localizer 를 교체하고 t.Cleanup 으로 원복.
func setLocalizer(t *testing.T, lang string) {
	t.Helper()
	mu.RLock()
	old := loc
	mu.RUnlock()
	t.Cleanup(func() {
		mu.Lock()
		loc = old
		mu.Unlock()
	})
	Init(lang)
}

// --- locale 파일 키 일치 검증 (가장 중요) ---

// TestLocaleKeysConsistency 는 4개 locale 파일의 키 집합이 en 기준으로
// 완전히 일치하는지 검증. 누락(missing)/잉여(extra) 키를 모두 보고.
func TestLocaleKeysConsistency(t *testing.T) {
	base := loadLocaleKeys(t, "en")
	if len(base) == 0 {
		t.Fatal("en.yaml 에서 키를 하나도 읽지 못함")
	}

	for _, name := range []string{"ko", "ja", "zh"} {
		t.Run(name, func(t *testing.T) {
			keys := loadLocaleKeys(t, name)

			var missing, extra []string
			for k := range base {
				if !keys[k] {
					missing = append(missing, k)
				}
			}
			for k := range keys {
				if !base[k] {
					extra = append(extra, k)
				}
			}
			sort.Strings(missing)
			sort.Strings(extra)

			if len(missing) > 0 {
				t.Errorf("%s.yaml 에 누락된 키 %d개 (en 기준):\n  %s",
					name, len(missing), strings.Join(missing, "\n  "))
			}
			if len(extra) > 0 {
				t.Errorf("%s.yaml 에 잉여 키 %d개 (en 에 없음):\n  %s",
					name, len(extra), strings.Join(extra, "\n  "))
			}
		})
	}
}

// TestLocaleValuesNotEmpty 는 모든 locale 의 메시지 값("other")이 비어 있지 않은지 검증.
func TestLocaleValuesNotEmpty(t *testing.T) {
	for _, name := range []string{"en", "ko", "ja", "zh"} {
		t.Run(name, func(t *testing.T) {
			data, err := localesFS.ReadFile("locales/" + name + ".yaml")
			if err != nil {
				t.Fatalf("locale 파일 읽기 실패: %v", err)
			}
			var doc map[string]map[string]string
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("locale 파일 파싱 실패: %v", err)
			}
			for key, msg := range doc {
				if strings.TrimSpace(msg["other"]) == "" {
					t.Errorf("%s.yaml: 키 %q 의 other 값이 비어 있음", name, key)
				}
			}
		})
	}
}

// --- T() 동작 검증 ---

// TestT_ExistingKey 는 존재하는 키가 locale 별로 올바르게 번역되는지 검증.
func TestT_ExistingKey(t *testing.T) {
	cases := []struct {
		lang string
		want string
	}{
		{"en", "comment"},
		{"ko", "주석"},
		{"ja", "コメント"},
		{"zh", "注释"},
	}
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			setLocalizer(t, tc.lang)
			if got := T("diff.kind_comment", nil); got != tc.want {
				t.Errorf("T(diff.kind_comment) = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestT_TemplateData 는 map[string]any 템플릿 변수가 치환되는지 검증.
func TestT_TemplateData(t *testing.T) {
	setLocalizer(t, "en")
	got := T("msg.subject_too_long", map[string]any{"Length": 80, "Max": 72})
	want := "commit subject is too long (80 chars > max 72)"
	if got != want {
		t.Errorf("T(msg.subject_too_long) = %q, want %q", got, want)
	}
}

// TestT_TemplateDataKorean 는 한국어 locale 에서도 템플릿 치환이 동작하는지 검증.
func TestT_TemplateDataKorean(t *testing.T) {
	setLocalizer(t, "ko")
	got := T("init.created", map[string]any{"Path": ".commit-checker.yml"})
	if !strings.Contains(got, ".commit-checker.yml") {
		t.Errorf("T(init.created) = %q, Path 치환 결과가 포함되어야 함", got)
	}
}

// TestT_UnknownKey 는 존재하지 않는 키가 "[msgID]" 형식으로 반환되는지 검증.
func TestT_UnknownKey(t *testing.T) {
	setLocalizer(t, "en")
	if got := T("no.such.key", nil); got != "[no.such.key]" {
		t.Errorf("T(no.such.key) = %q, want %q", got, "[no.such.key]")
	}
}

// TestT_LazyInit 는 localizer 미초기화 상태에서 T 호출 시 자동 Init 되는지 검증.
func TestT_LazyInit(t *testing.T) {
	mu.RLock()
	old := loc
	mu.RUnlock()
	t.Cleanup(func() {
		mu.Lock()
		loc = old
		mu.Unlock()
	})

	mu.Lock()
	loc = nil
	mu.Unlock()

	t.Setenv("COMMIT_CHECKER_LANG", "en")
	if got := T("diff.kind_comment", nil); got != "comment" {
		t.Errorf("미초기화 상태의 T() = %q, want %q", got, "comment")
	}
}

// --- locale 전환(Init) 검증 ---

// TestInit_SwitchLocale 는 Init 으로 locale 전환 시 번역 결과가 바뀌는지 검증.
func TestInit_SwitchLocale(t *testing.T) {
	setLocalizer(t, "en")
	if got := T("diff.kind_string_literal", nil); got != "string literal" {
		t.Errorf("en: T(diff.kind_string_literal) = %q", got)
	}

	Init("ko")
	if got := T("diff.kind_string_literal", nil); got != "문자열 리터럴" {
		t.Errorf("ko 전환 후: T(diff.kind_string_literal) = %q", got)
	}
}

// TestInit_UnsupportedLangFallsBackToEnglish 는 미지원 언어 태그 시 en 으로 폴백되는지 검증.
func TestInit_UnsupportedLangFallsBackToEnglish(t *testing.T) {
	setLocalizer(t, "fr")
	if got := T("diff.kind_comment", nil); got != "comment" {
		t.Errorf("미지원 locale: T(diff.kind_comment) = %q, want %q", got, "comment")
	}
}

// TestInit_EmptyLangDetectsFromEnv 는 빈 문자열 Init 시 환경 변수에서 감지하는지 검증.
func TestInit_EmptyLangDetectsFromEnv(t *testing.T) {
	t.Setenv("COMMIT_CHECKER_LANG", "ja")
	setLocalizer(t, "")
	if got := T("diff.kind_comment", nil); got != "コメント" {
		t.Errorf("env 감지 Init: T(diff.kind_comment) = %q, want %q", got, "コメント")
	}
}

// --- DetectLocale / parseLocale 검증 ---

// clearLocaleEnv 는 locale 관련 환경 변수를 모두 비움 (t.Setenv 가 원복 처리).
func clearLocaleEnv(t *testing.T) {
	t.Helper()
	for _, env := range []string{"COMMIT_CHECKER_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(env, "")
	}
}

// TestDetectLocale_Priority 는 환경 변수 우선순위(COMMIT_CHECKER_LANG > LC_ALL > LC_MESSAGES > LANG)를 검증.
func TestDetectLocale_Priority(t *testing.T) {
	clearLocaleEnv(t)
	t.Setenv("LANG", "ja_JP.UTF-8")
	t.Setenv("LC_MESSAGES", "zh_CN.UTF-8")
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("COMMIT_CHECKER_LANG", "ko")

	if got := DetectLocale(); got != "ko" {
		t.Errorf("COMMIT_CHECKER_LANG 우선이어야 함: got %q", got)
	}

	t.Setenv("COMMIT_CHECKER_LANG", "")
	if got := DetectLocale(); got != "en" {
		t.Errorf("LC_ALL 차순위여야 함: got %q", got)
	}

	t.Setenv("LC_ALL", "")
	if got := DetectLocale(); got != "zh" {
		t.Errorf("LC_MESSAGES 차차순위여야 함: got %q", got)
	}

	t.Setenv("LC_MESSAGES", "")
	if got := DetectLocale(); got != "ja" {
		t.Errorf("LANG 최후순위여야 함: got %q", got)
	}
}

// TestDetectLocale_Default 는 환경 변수가 없을 때 "en" 을 반환하는지 검증.
func TestDetectLocale_Default(t *testing.T) {
	clearLocaleEnv(t)
	if got := DetectLocale(); got != "en" {
		t.Errorf("기본값은 en 이어야 함: got %q", got)
	}
}

// TestParseLocale 는 다양한 로케일 문자열 형식에서 언어 코드 추출을 검증.
func TestParseLocale(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ko", "ko"},
		{"ko_KR.UTF-8", "ko"},
		{"KO_KR", "ko"},
		{"ja_JP.UTF-8", "ja"},
		{"zh_CN", "zh"},
		{"zh.UTF-8", "zh"},
		{"en_US.UTF-8", "en"},
		{"C", "en"},
		{"C.UTF-8", "en"},
		{"POSIX", "en"},
		{"fr_FR.UTF-8", "en"}, // 미지원 언어는 en 폴백
		{"de", "en"},
	}
	for _, tc := range cases {
		if got := parseLocale(tc.in); got != tc.want {
			t.Errorf("parseLocale(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
