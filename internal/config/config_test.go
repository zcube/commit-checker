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
