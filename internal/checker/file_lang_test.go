package checker_test

// Integration tests for per-file language rules (file_languages config)
// and in-code directives (commit-checker:*).
// Each test spins up a real temporary git repo.

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

// ---- file_languages config tests --------------------------------------------

// TestCheckDiff_FileLanguages_AnyAllows: files matching a "language: any" rule
// are not language-checked regardless of their comment language.
func TestCheckDiff_FileLanguages_AnyAllows(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "locales/en.go", `package locales

// This is an English translation file
const Hello = "Hello"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "locales/**", Language: "any"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("locales/** with language=any should pass, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_EnglishOverride: specific path requires English.
func TestCheckDiff_FileLanguages_EnglishOverride(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "i18n/messages.go", `package i18n

// All comments here must be in English
const Greeting = "Hello"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "i18n/**", Language: "english"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("i18n/** with language=english should accept English, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_EnglishOverride_KoreanFails: Korean comment in
// an English-overridden file must fail.
func TestCheckDiff_FileLanguages_EnglishOverride_KoreanFails(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "i18n/messages.go", `package i18n

// 한국어 주석은 이 파일에서 실패해야 합니다
const Greeting = "Hello"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "i18n/**", Language: "english"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("Korean comment in English-override file should fail")
	}
}

// TestCheckDiff_FileLanguages_LocaleCode: locale code "en" works like "english".
func TestCheckDiff_FileLanguages_LocaleCode(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "translations/en.go", `package translations

// English translation comments are fine here
const Key = "value"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "translations/**", Language: "en"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("locale code 'en' should work as English override, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_Japanese: Japanese file rule accepts Japanese.
func TestCheckDiff_FileLanguages_Japanese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "locale/ja/messages.go", `package ja

// これは日本語の翻訳ファイルです
const Hello = "こんにちは"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "locale/ja/**", Language: "japanese"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Japanese file with language=japanese should pass, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_Chinese: Chinese file rule accepts Chinese.
func TestCheckDiff_FileLanguages_Chinese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "locale/zh/messages.go", `package zh

// 这是中文翻译文件的注释
const Hello = "你好"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "locale/zh/**", Language: "chinese"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Chinese file with language=chinese should pass, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_FirstMatchWins: rules are checked in order.
func TestCheckDiff_FileLanguages_FirstMatchWins(t *testing.T) {
	dir := newGitRepo(t)
	// File matches both rules; first (any) should win.
	stageFile(t, dir, "i18n/en/msg.go", `package msg

// English comment in a file that matches multiple patterns
const Key = "value"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "i18n/**", Language: "any"},   // first match → any
		{Pattern: "i18n/en/**", Language: "english"}, // would also match but loses
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("first matching rule (any) should win, got: %v", errs)
	}
}

// TestCheckDiff_FileLanguages_NoMatchUsesDefault: unmatched files use global default.
func TestCheckDiff_FileLanguages_NoMatchUsesDefault(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service/handler.go", `package handler

// This English comment is not in an overridden path
func Handle() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "i18n/**", Language: "english"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("unmatched file should use global Korean default and fail for English")
	}
}

// ---- in-code directive tests ------------------------------------------------

// TestCheckDiff_Directive_Ignore: commit-checker:ignore skips the next comment.
func TestCheckDiff_Directive_Ignore(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:ignore
// This English comment should be ignored by the checker
func skipThis() {}

// 이 한국어 주석은 체크됩니다
func checkThis() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("commit-checker:ignore should suppress the following comment, got: %v", errs)
	}
}

// TestCheckDiff_Directive_Disable_Enable: region between disable/enable is skipped.
func TestCheckDiff_Directive_Disable_Enable(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// 한국어 주석 — 체크됨
func before() {}

// commit-checker:disable
// This English comment is inside a disabled region
// Another English comment also disabled
// commit-checker:enable

// 한국어 재개 — 체크됨
func after() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disable/enable region should suppress English comments, got: %v", errs)
	}
}

// TestCheckDiff_Directive_Disable_WithLang: disable:lang= allows another language.
func TestCheckDiff_Directive_Disable_WithLang(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:disable:lang=english
// This section requires English comments
// All English here is intentional
// commit-checker:enable

// 여기서 한국어로 돌아옴
func resumeKorean() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disable:lang=english region should accept English, got: %v", errs)
	}
}

// TestCheckDiff_Directive_FileLang: commit-checker:file-lang= sets language for whole file.
func TestCheckDiff_Directive_FileLang(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:file-lang=english

// This file uses English comments throughout
// All comments here are intentionally in English
func process() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file-lang=english should accept all English comments, got: %v", errs)
	}
}

// TestCheckDiff_Directive_FileLang_Korean_Fails: file-lang=english; Korean fails.
func TestCheckDiff_Directive_FileLang_Korean_Fails(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:file-lang=english

// 이 파일은 영어를 요구하므로 한국어는 실패해야 합니다
func process() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("file-lang=english: Korean comment should fail English requirement")
	}
}

// TestCheckDiff_Directive_TypeScript_Ignore: directives work in TypeScript too.
func TestCheckDiff_Directive_TypeScript_Ignore(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service.ts", `// commit-checker:ignore
// This English TypeScript comment is intentionally ignored
const x = 1;

// 이 한국어 주석은 체크됩니다
const y = 2;
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("TypeScript :ignore directive should work, got: %v", errs)
	}
}

// TestCheckDiff_Directive_Python_Disable: directives work in Python.
func TestCheckDiff_Directive_Python_Disable(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `# commit-checker:disable
# This English Python comment is in a disabled region
# commit-checker:enable

# 한국어 주석은 체크됩니다
def process():
    pass
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Python :disable/:enable should work, got: %v", errs)
	}
}

// TestCheckDiff_Directive_FileLang_Japanese: file-lang=ja makes Japanese required.
func TestCheckDiff_Directive_FileLang_Japanese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:file-lang=ja

// これはひらがなとカタカナで書かれたコメントです
// ユーザー処理のロジック
func process() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file-lang=ja: Japanese kana comments should pass, got: %v", errs)
	}
}

// TestCheckDiff_Directive_FileLang_Chinese: file-lang=zh makes Chinese required.
func TestCheckDiff_Directive_FileLang_Chinese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// commit-checker:file-lang=zh

// 这是一个中文注释的示例内容
func process() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file-lang=zh: Chinese comments should pass, got: %v", errs)
	}
}

// TestCheckDiff_Directive_And_FileLanguages_Interaction: file_languages config
// sets the base; directives can further override within the file.
func TestCheckDiff_Directive_And_FileLanguages_Interaction(t *testing.T) {
	dir := newGitRepo(t)
	// i18n file is English by config, but has a Korean section via directive.
	stageFile(t, dir, "i18n/base.go", `package i18n

// English comment — OK for this file (rule: english)
const A = "a"

// commit-checker:disable:lang=korean
// 한국어 섹션 — 디렉티브로 허용
// commit-checker:enable

// Another English comment — back to file rule
const B = "b"
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.FileLanguages = []config.FileLanguageRule{
		{Pattern: "i18n/**", Language: "english"},
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("directive should override file_languages rule within region, got: %v", errs)
	}
}
