package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

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
