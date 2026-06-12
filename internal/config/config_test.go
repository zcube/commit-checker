package config_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

// setGlobalHome: HOME 을 tmpHome 으로 바꾸고, 상위 우선순위 전역 경로
// (COMMIT_CHECKER_GLOBAL_CONFIG, XDG_CONFIG_HOME)는 빈 임시 경로로 격리하여
// legacy 경로(~/.commit-checker.yml)의 전역 설정만 보이도록 한다.
func setGlobalHome(t *testing.T, tmpHome string) {
	t.Helper()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, "xdg-empty"))
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", "")
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".commit-checker.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Defaults_WhenFileAbsent(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/.commit-checker.yml")
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "korean" {
		t.Errorf("default required_language = %q, want korean", cfg.CommentLanguage.RequiredLanguage)
	}
	if cfg.CommentLanguage.MinLength != 5 {
		t.Errorf("default min_length = %d, want 5", cfg.CommentLanguage.MinLength)
	}
	if cfg.CommentLanguage.CheckMode != "diff" {
		t.Errorf("default check_mode = %q, want diff", cfg.CommentLanguage.CheckMode)
	}
	if len(cfg.CommentLanguage.Extensions) == 0 {
		t.Error("default extensions should not be empty")
	}
	if cfg.CommitMessage.Locale != "ko" {
		t.Errorf("default commit_message.locale = %q, want ko", cfg.CommitMessage.Locale)
	}
	if !cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("default no_ai_coauthor should be true")
	}
	if !cfg.CommitMessage.IsNoUnicodeSpaces() {
		t.Error("default no_unicode_spaces should be true")
	}
	if !cfg.CommitMessage.IsNoAmbiguousChars() {
		t.Error("default no_ambiguous_chars should be true")
	}
	if !cfg.CommitMessage.IsNoBadRunes() {
		t.Error("default no_bad_runes should be true")
	}
	if cfg.CommitMessage.LanguageCheck.IsEnabled() {
		t.Error("default language_check.enabled should be false")
	}
}

func TestLoad_LocaleOverridesRequiredLanguage(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  locale: ko
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "korean" {
		t.Errorf("locale=ko should set required_language=korean, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_LocaleJapanese(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  locale: ja
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "japanese" {
		t.Errorf("locale=ja should set required_language=japanese, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_LocaleEnglish(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  locale: en
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "english" {
		t.Errorf("locale=en should set required_language=english, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_LocaleChineseVariants(t *testing.T) {
	for _, locale := range []string{"zh", "zh-hans", "zh-hant"} {
		path := writeConfig(t, "comment_language:\n  locale: "+locale+"\n")
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load(%s): %v", locale, err)
		}
		if cfg.CommentLanguage.RequiredLanguage != "chinese" {
			t.Errorf("locale=%s should set required_language=chinese, got %q", locale, cfg.CommentLanguage.RequiredLanguage)
		}
	}
}

func TestLoad_RequiredLanguageExplicit(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  required_language: english
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "english" {
		t.Errorf("got %q, want english", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_LanguagesFilter(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  languages:
    - go
    - typescript
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// When languages is set, extensions default is NOT applied.
	if len(cfg.CommentLanguage.Extensions) != 0 {
		t.Errorf("extensions should be empty when languages is set, got %v", cfg.CommentLanguage.Extensions)
	}
	if len(cfg.CommentLanguage.Languages) != 2 {
		t.Errorf("expected 2 languages, got %v", cfg.CommentLanguage.Languages)
	}
}

func TestLoad_CheckModeFull(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  check_mode: full
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CommentLanguage.IsFullMode() {
		t.Error("expected check_mode=full to be active")
	}
}

func TestLoad_DisabledCommentLanguage(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  enabled: false
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.IsEnabled() {
		t.Error("expected comment_language.enabled=false")
	}
}

func TestLoad_NoAICoauthorDisabled(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  no_ai_coauthor: false
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("expected no_ai_coauthor=false")
	}
}

func TestLoad_CommitMessageLanguageCheck(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  language_check:
    enabled: true
    required_language: english
    min_length: 10
    skip_prefixes:
      - "Merge"
      - "WIP"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	lc := cfg.CommitMessage.LanguageCheck
	if !lc.IsEnabled() {
		t.Error("expected language_check.enabled=true")
	}
	if lc.RequiredLanguage != "english" {
		t.Errorf("expected english, got %q", lc.RequiredLanguage)
	}
	if lc.MinLength != 10 {
		t.Errorf("expected min_length=10, got %d", lc.MinLength)
	}
	if len(lc.SkipPrefixes) != 2 {
		t.Errorf("expected 2 skip_prefixes, got %v", lc.SkipPrefixes)
	}
}

func TestLoad_Exceptions(t *testing.T) {
	path := writeConfig(t, `
exceptions:
  global_ignore:
    - "vendor/**"
    - "third_party/**"
  comment_language_ignore:
    - "legacy/**"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Exceptions.GlobalIgnore) != 2 {
		t.Errorf("expected 2 global_ignore entries, got %v", cfg.Exceptions.GlobalIgnore)
	}
	if len(cfg.Exceptions.CommentLanguageIgnore) != 1 {
		t.Errorf("expected 1 comment_language_ignore, got %v", cfg.Exceptions.CommentLanguageIgnore)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "comment_language: [invalid: yaml: here")
	_, err := config.Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_CommitMessageLocale(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  locale: ja
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommitMessage.Locale != "ja" {
		t.Errorf("expected locale=ja, got %q", cfg.CommitMessage.Locale)
	}
}

func TestLoad_IgnoreFiles(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  ignore_files:
    - "*.pb.go"
    - "internal/generated/**"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.CommentLanguage.IgnoreFiles) != 2 {
		t.Errorf("expected 2 ignore_files, got %v", cfg.CommentLanguage.IgnoreFiles)
	}
}

func TestCommentLanguageConfig_IsNoEmoji(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  no_emoji: true
`)
	cfg, _ := config.Load(path)
	if !cfg.CommentLanguage.IsNoEmoji() {
		t.Error("expected no_emoji=true")
	}

	path2 := writeConfig(t, `comment_language: {}`)
	cfg2, _ := config.Load(path2)
	if cfg2.CommentLanguage.IsNoEmoji() {
		t.Error("default no_emoji should be false")
	}
}

func TestCommentLanguageConfig_IsCheckStrings(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  check_strings: true
`)
	cfg, _ := config.Load(path)
	if !cfg.CommentLanguage.IsCheckStrings() {
		t.Error("expected check_strings=true")
	}

	path2 := writeConfig(t, `comment_language: {}`)
	cfg2, _ := config.Load(path2)
	if cfg2.CommentLanguage.IsCheckStrings() {
		t.Error("default check_strings should be false")
	}
}

func TestCommentLanguageConfig_IsSkipTechnicalStrings(t *testing.T) {
	path := writeConfig(t, `
comment_language:
  skip_technical_strings: false
`)
	cfg, _ := config.Load(path)
	if cfg.CommentLanguage.IsSkipTechnicalStrings() {
		t.Error("expected skip_technical_strings=false")
	}

	path2 := writeConfig(t, `comment_language: {}`)
	cfg2, _ := config.Load(path2)
	if !cfg2.CommentLanguage.IsSkipTechnicalStrings() {
		t.Error("default skip_technical_strings should be true")
	}
}

func TestConventionalCommitConfig_Defaults(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  conventional_commit:
    enabled: true
`)
	cfg, _ := config.Load(path)
	cc := cfg.CommitMessage.ConventionalCommit
	if !cc.IsEnabled() {
		t.Error("expected enabled=true")
	}
	if cc.IsRequireScope() {
		t.Error("default require_scope should be false")
	}
	if !cc.IsAllowMergeCommits() {
		t.Error("default allow_merge_commits should be true")
	}
	if !cc.IsAllowRevertCommits() {
		t.Error("default allow_revert_commits should be true")
	}
	types := cc.GetTypes()
	if len(types) == 0 {
		t.Error("default types should not be empty")
	}
}

func TestConventionalCommitConfig_Overrides(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  conventional_commit:
    enabled: true
    require_scope: true
    allow_merge_commits: false
    allow_revert_commits: false
    types:
      - feat
      - fix
`)
	cfg, _ := config.Load(path)
	cc := cfg.CommitMessage.ConventionalCommit
	if !cc.IsRequireScope() {
		t.Error("expected require_scope=true")
	}
	if cc.IsAllowMergeCommits() {
		t.Error("expected allow_merge_commits=false")
	}
	if cc.IsAllowRevertCommits() {
		t.Error("expected allow_revert_commits=false")
	}
	if len(cc.GetTypes()) != 2 {
		t.Errorf("expected 2 types, got %v", cc.GetTypes())
	}
}

func TestConventionalCommitConfig_GetTypeAliases(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  conventional_commit:
    locale: ko
`)
	cfg, _ := config.Load(path)
	cc := cfg.CommitMessage.ConventionalCommit
	aliases := cc.GetTypeAliases()
	if len(aliases) == 0 {
		t.Error("locale=ko should have built-in aliases")
	}

	// 사용자 정의 aliases
	path2 := writeConfig(t, `
commit_message:
  conventional_commit:
    type_aliases:
      "기능": feat
`)
	cfg2, _ := config.Load(path2)
	cc2 := cfg2.CommitMessage.ConventionalCommit
	if cc2.GetTypeAliases()["기능"] != "feat" {
		t.Error("expected custom alias 기능=feat")
	}
}

func TestConventionalCommitConfig_GetAllAllowedTypes(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  conventional_commit:
    locale: ko
`)
	cfg, _ := config.Load(path)
	cc := cfg.CommitMessage.ConventionalCommit
	all := cc.GetAllAllowedTypes()
	if len(all) <= len(config.DefaultConventionalTypes) {
		t.Error("expected more types when locale aliases are included")
	}
}

func TestConventionalCommitConfig_ResolveType(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  conventional_commit:
    locale: ko
`)
	cfg, _ := config.Load(path)
	cc := cfg.CommitMessage.ConventionalCommit
	if cc.ResolveType("기능") != "feat" {
		t.Errorf("expected feat, got %q", cc.ResolveType("기능"))
	}
	if cc.ResolveType("feat") != "feat" {
		t.Errorf("standard type should pass through, got %q", cc.ResolveType("feat"))
	}
}

func TestCommitMessageConfig_IsEnabled(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  enabled: false
`)
	cfg, _ := config.Load(path)
	if cfg.CommitMessage.IsEnabled() {
		t.Error("expected enabled=false")
	}
}

func TestCommitMessageConfig_IsNoEmoji(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  no_emoji: true
`)
	cfg, _ := config.Load(path)
	if !cfg.CommitMessage.IsNoEmoji() {
		t.Error("expected no_emoji=true")
	}
}

func TestCommitMessageConfig_CoauthorShouldRemove(t *testing.T) {
	path := writeConfig(t, `
commit_message:
  coauthor_remove_emails:
    - "*@myai.com"
`)
	cfg, _ := config.Load(path)
	cm := cfg.CommitMessage

	// 내장 AI 패턴
	if !cm.CoauthorShouldRemove("noreply@anthropic.com") {
		t.Error("anthropic should be removed")
	}
	// 사용자 정의 패턴
	if !cm.CoauthorShouldRemove("bot@myai.com") {
		t.Error("custom pattern should match")
	}
	// 일반 사람
	if cm.CoauthorShouldRemove("developer@example.com") {
		t.Error("regular developer should not be removed")
	}
}

func TestExtractCoauthorEmail(t *testing.T) {
	cases := []struct {
		line  string
		email string
	}{
		{"Co-authored-by: Bot <bot@example.com>", "bot@example.com"},
		{"Co-authored-by: No Email", ""},
		{"", ""},
		{"<only-email@test.com>", "only-email@test.com"},
	}
	for _, tc := range cases {
		got := config.ExtractCoauthorEmail(tc.line)
		if got != tc.email {
			t.Errorf("ExtractCoauthorEmail(%q) = %q, want %q", tc.line, got, tc.email)
		}
	}
}

func TestBinaryFileConfig_IsEnabled(t *testing.T) {
	path := writeConfig(t, `
binary_file:
  enabled: false
`)
	cfg, _ := config.Load(path)
	if cfg.BinaryFile.IsEnabled() {
		t.Error("expected binary_file.enabled=false")
	}
}

func TestCustomRules_CommitMessage(t *testing.T) {
	path := writeConfig(t, `
custom_rules:
  commit_message:
    - name: no-wip
      pattern: "(?i)^WIP"
      message: "WIP 접두사를 제거하세요"
    - name: need-ticket
      pattern: "\\[PROJ-\\d+\\]"
      message: "티켓 ID가 필요합니다"
      required: true
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CustomRules.CommitMessage) != 2 {
		t.Fatalf("expected 2 custom rules, got %d", len(cfg.CustomRules.CommitMessage))
	}
	if cfg.CustomRules.CommitMessage[0].Name != "no-wip" {
		t.Errorf("rule[0].Name = %q, want no-wip", cfg.CustomRules.CommitMessage[0].Name)
	}
	if cfg.CustomRules.CommitMessage[0].Required {
		t.Error("rule[0].Required should be false (default)")
	}
	if !cfg.CustomRules.CommitMessage[1].Required {
		t.Error("rule[1].Required should be true")
	}
}

func TestCustomRules_Diff(t *testing.T) {
	path := writeConfig(t, `
custom_rules:
  diff:
    - name: no-api-key
      pattern: "(?i)api_key\\s*=\\s*['\"][^'\"]{10,}"
      message: "API 키가 감지되었습니다"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CustomRules.Diff) != 1 {
		t.Fatalf("expected 1 diff rule, got %d", len(cfg.CustomRules.Diff))
	}
	if cfg.CustomRules.Diff[0].Name != "no-api-key" {
		t.Errorf("rule.Name = %q, want no-api-key", cfg.CustomRules.Diff[0].Name)
	}
}

func TestLoad_GlobalAllowedWords_프로젝트존재시무시(t *testing.T) {
	// 프로젝트 설정이 있으면 전역의 allowed_words 는 전혀 병합되지 않아야 함
	path := writeConfig(t, `
comment_language:
  allowed_words:
    - ProjectWord
`)
	// Temporarily override HOME to inject a global config
	tmpHome := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(tmpHome, ".commit-checker.yml"),
		[]byte("comment_language:\n  allowed_words:\n    - GlobalWord\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	setGlobalHome(t, tmpHome)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	words := cfg.CommentLanguage.AllowedWords
	hasGlobal := false
	hasProject := false
	for _, w := range words {
		if w == "GlobalWord" {
			hasGlobal = true
		}
		if w == "ProjectWord" {
			hasProject = true
		}
	}
	if hasGlobal {
		t.Errorf("프로젝트 설정 존재 시 전역 GlobalWord 가 섞이면 안 됨: %v", words)
	}
	if !hasProject {
		t.Errorf("expected ProjectWord from project config in AllowedWords, got %v", words)
	}
}

func TestLoad_GlobalCustomRules_프로젝트존재시무시(t *testing.T) {
	// 프로젝트 설정이 있으면 전역의 custom_rules 도 합쳐지지 않아야 함
	path := writeConfig(t, `
custom_rules:
  commit_message:
    - name: project-rule
      pattern: "FORBIDDEN"
`)
	tmpHome := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(tmpHome, ".commit-checker.yml"),
		[]byte("custom_rules:\n  commit_message:\n    - name: global-rule\n      pattern: \"SECRET\"\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	setGlobalHome(t, tmpHome)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CustomRules.CommitMessage) != 1 {
		t.Fatalf("expected 1 rule (project only, global ignored), got %d: %v",
			len(cfg.CustomRules.CommitMessage), cfg.CustomRules.CommitMessage)
	}
	if cfg.CustomRules.CommitMessage[0].Name != "project-rule" {
		t.Errorf("only project-rule should remain, got %s", cfg.CustomRules.CommitMessage[0].Name)
	}
}

func TestLoad_Preset_OverridesDefault(t *testing.T) {
	// 프리셋: required_language=english 설정
	presetYAML := `
comment_language:
  required_language: english
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(presetYAML))
	}))
	defer srv.Close()

	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "english" {
		t.Errorf("preset required_language should be english, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_Preset_ProjectOverridesPreset(t *testing.T) {
	// 프리셋: required_language=english, 프로젝트: required_language=korean
	presetYAML := `
comment_language:
  required_language: english
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(presetYAML))
	}))
	defer srv.Close()

	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
comment_language:
  required_language: korean
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.RequiredLanguage != "korean" {
		t.Errorf("project should override preset: want korean, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_Preset_MigratesOldSchema(t *testing.T) {
	// 프리셋이 구 버전(no_coauthor) 스키마인 경우 마이그레이션 후 로드
	presetYAML := `
commit_message:
  no_coauthor: true
`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(presetYAML))
	}))
	defer srv.Close()

	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 마이그레이션 후 no_ai_coauthor가 true여야 함
	if !cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("preset no_coauthor should be migrated to no_ai_coauthor=true")
	}
}

func TestLoad_Preset_NoNestedPreset(t *testing.T) {
	// 프리셋 안에 preset.url이 있으면 에러 (중첩/무한루프 방지)
	innerCalled := false
	inner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		_, _ = w.Write([]byte("comment_language:\n  required_language: japanese\n"))
	}))
	defer inner.Close()

	presetYAML := `
preset:
  url: ` + inner.URL + `
comment_language:
  required_language: english
`
	outer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(presetYAML))
	}))
	defer outer.Close()

	path := writeConfig(t, `
preset:
  url: `+outer.URL+`
`)
	_, err := config.Load(path)
	if err == nil {
		t.Error("nested preset should return error")
	}
	if innerCalled {
		t.Error("inner preset URL must not be fetched")
	}
}

func TestLoad_Preset_Cache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_, _ = w.Write([]byte("comment_language:\n  required_language: english\n"))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	projectYAML := `
preset:
  url: ` + srv.URL + `
  cache:
    enabled: true
    ttl: 1h
    dir: ` + cacheDir + `
`
	path := writeConfig(t, projectYAML)

	// 첫 번째 로드: 서버 호출
	_, err := config.Load(path)
	if err != nil {
		t.Fatalf("first Load: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 server call, got %d", callCount)
	}

	// 두 번째 로드: 캐시에서 읽어야 함
	_, err = config.Load(path)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected no additional server call (cached), got %d total", callCount)
	}
}

func TestLoad_Preset_ErrorOnFetch(t *testing.T) {
	path := writeConfig(t, `
preset:
  url: http://127.0.0.1:1
`)
	_, err := config.Load(path)
	if err == nil {
		t.Error("expected error when preset URL is unreachable")
	}
}

func TestLoad_Preset_프로젝트존재시_전역무시(t *testing.T) {
	// 전역: required_language=japanese, 프리셋: required_language=english
	// 프로젝트 설정이 존재하므로 전역은 로드 자체가 생략되고 프리셋 값이 적용되어야 함
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("comment_language:\n  required_language: english\n"))
	}))
	defer srv.Close()

	tmpHome := t.TempDir()
	globalCfg := filepath.Join(tmpHome, ".commit-checker.yml")
	if err := os.WriteFile(globalCfg, []byte("comment_language:\n  required_language: japanese\n"), 0644); err != nil {
		t.Fatal(err)
	}
	setGlobalHome(t, tmpHome)

	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 전역(japanese)은 무시되고 preset(english)이 적용되어야 함
	if cfg.CommentLanguage.RequiredLanguage != "english" {
		t.Errorf("global must be ignored, preset applies: want english, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}

func TestLoad_Preset_프로젝트우선_전역무시(t *testing.T) {
	// 전역: min_length=1, 프리셋: min_length=2, 프로젝트: min_length=3
	// 우선순위: 프로젝트 > 프리셋 (전역은 프로젝트 설정 존재 시 완전 무시)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("comment_language:\n  min_length: 2\n"))
	}))
	defer srv.Close()

	tmpHome := t.TempDir()
	globalCfg := filepath.Join(tmpHome, ".commit-checker.yml")
	if err := os.WriteFile(globalCfg, []byte("comment_language:\n  min_length: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	setGlobalHome(t, tmpHome)

	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
comment_language:
  min_length: 3
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.MinLength != 3 {
		t.Errorf("project should win: want min_length=3, got %d", cfg.CommentLanguage.MinLength)
	}

	// 프로젝트가 min_length를 설정하지 않은 경우 프리셋(2)이 사용되어야 함 (전역 1 은 무시)
	path2 := writeConfig(t, `
preset:
  url: `+srv.URL+`
`)
	cfg2, err := config.Load(path2)
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if cfg2.CommentLanguage.MinLength != 2 {
		t.Errorf("preset applies, global ignored: want min_length=2, got %d", cfg2.CommentLanguage.MinLength)
	}
}

func TestLoad_Preset_OldSchemaWithPreset_MigratesAndLoads(t *testing.T) {
	// preset + 구형 필드(no_coauthor) 조합: 마이그레이션 후 정상 로드 확인
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("comment_language:\n  required_language: english\n"))
	}))
	defer srv.Close()

	// 구형 no_coauthor + preset.url 혼합 설정
	path := writeConfig(t, `
preset:
  url: `+srv.URL+`
commit_message:
  no_coauthor: true
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// no_coauthor → no_ai_coauthor 마이그레이션 동작 확인
	if !cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("no_coauthor should be migrated to no_ai_coauthor=true")
	}
	// preset의 required_language=english가 적용되어야 함
	if cfg.CommentLanguage.RequiredLanguage != "english" {
		t.Errorf("preset required_language should be english, got %q", cfg.CommentLanguage.RequiredLanguage)
	}
}
