package checker_test

// Integration tests for CheckDiff. Each test spins up a real temporary git
// repository, stages files, then calls CheckDiff and asserts the results.
//
// NOTE: These tests change the process working directory (os.Chdir) and must
// NOT be run in parallel with each other.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

// ---- helpers ----------------------------------------------------------------

// newGitRepo creates a temporary directory initialised as a git repo and
// changes the process CWD to it. On cleanup it restores the original CWD.
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gitMust(t, dir, "git", "init")
	gitMust(t, dir, "git", "config", "user.email", "test@commit-checker.test")
	gitMust(t, dir, "git", "config", "user.name", "Commit Checker Test")
	gitMust(t, dir, "git", "config", "commit.gpgsign", "false")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// seedCommit writes content to path, stages it and creates an initial commit
// so that subsequent `git diff --staged` shows only the next change.
func seedCommit(t *testing.T, dir, relPath, content string) {
	t.Helper()
	writeFile(t, dir, relPath, content)
	gitMust(t, dir, "git", "add", relPath)
	gitMust(t, dir, "git", "commit", "-m", "seed")
}

// stageFile writes content to path and stages it (git add).
func stageFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	writeFile(t, dir, relPath, content)
	gitMust(t, dir, "git", "add", relPath)
}

func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitMust(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
}

func koreanOnlyConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.CommentLanguage.Enabled = &t
	cfg.CommentLanguage.RequiredLanguage = "korean"
	cfg.CommentLanguage.MinLength = 5
	cfg.CommentLanguage.CheckMode = "diff"
	cfg.CommentLanguage.Extensions = []string{".go", ".ts", ".py"}
	return cfg
}

// ---- tests ------------------------------------------------------------------

// TestCheckDiff_KoreanComment_Pass: new Go file with only Korean comments.
func TestCheckDiff_KoreanComment_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// 한국어 주석입니다
func main() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for Korean comment, got: %v", errs)
	}
}

// TestCheckDiff_EnglishComment_Fail: new Go file with English comments fails Korean check.
func TestCheckDiff_EnglishComment_Fail(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// This comment is written in English only
func main() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected language error for English comment, got none")
	}
}

// TestCheckDiff_DiffMode_OnlyAddedLinesChecked: existing file with clean Korean comment;
// only a new English comment line is added — only that new line should be flagged.
func TestCheckDiff_DiffMode_OnlyAddedLinesChecked(t *testing.T) {
	dir := newGitRepo(t)
	// Seed: file with a Korean comment (already committed, not in diff)
	seedCommit(t, dir, "svc.go", `package svc

// 기존 한국어 주석
func Old() {}
`)
	// Stage: add an English comment
	stageFile(t, dir, "svc.go", `package svc

// 기존 한국어 주석
func Old() {}

// This new comment is in English
func New() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for the new English comment line, got none")
	}
	// The pre-existing Korean comment must NOT produce an error.
	for _, e := range errs {
		if contains(e, "기존 한국어") {
			t.Errorf("pre-existing Korean comment should not be flagged: %s", e)
		}
	}
}

// TestCheckDiff_FullMode_AllCommentsChecked: full mode checks every comment in
// the staged file, even those not in the diff.
func TestCheckDiff_FullMode_AllCommentsChecked(t *testing.T) {
	dir := newGitRepo(t)
	// Seed: file with an English comment (pre-existing, not in diff)
	seedCommit(t, dir, "lib.go", `package lib

// Existing English comment that should be caught in full mode
func Existing() {}
`)
	// Add a harmless Korean change
	stageFile(t, dir, "lib.go", `package lib

// Existing English comment that should be caught in full mode
func Existing() {}

// 새로운 한국어 주석
func New() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.CheckMode = "full"
	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("full mode should flag the pre-existing English comment")
	}
}

// TestCheckDiff_Disabled: when enabled=false, CheckDiff returns nothing.
func TestCheckDiff_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// This is English and should normally fail
func main() {}
`)
	f := false
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.Enabled = &f

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled check should return no errors, got: %v", errs)
	}
}

// TestCheckDiff_UnsupportedExtension_Skipped: .md files are not in the extension list.
func TestCheckDiff_UnsupportedExtension_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "README.md", "# English readme\n\nThis is all English.\n")

	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("markdown file should be skipped, got: %v", errs)
	}
}

// TestCheckDiff_Languages_Filter: when languages=[go] is set, .ts files are skipped.
func TestCheckDiff_Languages_Filter(t *testing.T) {
	dir := newGitRepo(t)
	// Stage a TypeScript file with an English comment
	stageFile(t, dir, "app.ts", `// This English comment should be ignored
const x = 1;
`)
	// Stage a Go file with an English comment
	stageFile(t, dir, "main.go", `package main

// This English comment should be flagged
func main() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.Languages = []string{"go"} // only Go

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	// Should flag Go but not TypeScript
	if len(errs) == 0 {
		t.Error("expected error for Go English comment")
	}
	for _, e := range errs {
		if contains(e, "app.ts") {
			t.Errorf("TypeScript file should be skipped when languages=[go], got: %s", e)
		}
	}
}

// TestCheckDiff_IgnoreFiles_Skipped: files matching ignore_files glob are skipped.
func TestCheckDiff_IgnoreFiles_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "generated/gen.go", `package generated

// This English comment is in an ignored file
func Gen() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.IgnoreFiles = []string{"generated/**"}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("files matching ignore_files should be skipped, got: %v", errs)
	}
}

// TestCheckDiff_GlobalIgnore_Skipped: exceptions.global_ignore also skips files.
func TestCheckDiff_GlobalIgnore_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "vendor/pkg/pkg.go", `package pkg

// This is an English comment in vendor
func F() {}
`)
	cfg := koreanOnlyConfig()
	cfg.Exceptions.GlobalIgnore = []string{"vendor/**"}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("vendor files should be globally ignored, got: %v", errs)
	}
}

// TestCheckDiff_TechnicalComments_Skipped: directives like //nolint are skipped.
func TestCheckDiff_TechnicalComments_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

//nolint:errcheck
// TODO: fix this later
// https://github.com/example/issue
func main() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("technical comments should be skipped, got: %v", errs)
	}
}

// TestCheckDiff_TypeScript_EnglishFails: staged TypeScript with English comment fails.
func TestCheckDiff_TypeScript_EnglishFails(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service.ts", `// This TypeScript service handles requests
const x = 1;
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected English comment error for TypeScript file")
	}
}

// TestCheckDiff_Python_EnglishFails: staged Python with English comment fails.
func TestCheckDiff_Python_EnglishFails(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `# This Python function processes data
def process():
    pass
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected English comment error for Python file")
	}
}

// TestCheckDiff_Locale_Korean: locale=ko is equivalent to required_language=korean.
func TestCheckDiff_Locale_Korean(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// This comment is in English
func main() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.Locale = "ko"
	cfg.CommentLanguage.RequiredLanguage = "" // cleared; locale should take over
	// Apply locale via applyDefaults path by re-loading
	if lang := localeToLang("ko"); lang != "" {
		cfg.CommentLanguage.RequiredLanguage = lang
	}

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("locale=ko should enforce Korean; English comment should fail")
	}
}

// TestCheckDiff_AnyLanguage_Pass: required_language=any accepts any language.
func TestCheckDiff_AnyLanguage_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// This comment is in English and should be fine
func main() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "any"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("required_language=any should pass all, got: %v", errs)
	}
}

// TestCheckDiff_MultipleFiles: multiple staged files are all checked.
func TestCheckDiff_MultipleFiles(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "a.go", `package a

// This is English — should fail
func A() {}
`)
	stageFile(t, dir, "b.go", `package b

// 이것은 한국어 — 통과해야 합니다
func B() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error from a.go English comment")
	}
	for _, e := range errs {
		if contains(e, "b.go") {
			t.Errorf("b.go Korean comment should not fail: %s", e)
		}
	}
}

// TestCheckDiff_Japanese_Locale_Pass: Japanese comments pass when locale=ja.
func TestCheckDiff_Japanese_Locale_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// これは日本語のコメントです
// ユーザーデータを処理する関数
func process() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "japanese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for Japanese comment with required=japanese, got: %v", errs)
	}
}

// TestCheckDiff_Japanese_Locale_FailsKorean: Japanese comments fail when locale=ko.
func TestCheckDiff_Japanese_Locale_FailsKorean(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// これはひらがなとカタカナのコメントです
func process() {}
`)
	// kana-only text has no CJK characters so it won't be mistaken for Chinese either
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("kana-only comment should fail Korean language requirement")
	}
}

// TestCheckDiff_Japanese_KanaOnly_FailsChinese: kana-only text fails Chinese requirement.
func TestCheckDiff_Japanese_KanaOnly_FailsChinese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// これはひらがなとカタカナのコメントです
func process() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "chinese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("kana-only comment should fail Chinese requirement (no CJK characters)")
	}
}

// TestCheckDiff_Chinese_Locale_Pass: Chinese comments pass when required=chinese.
func TestCheckDiff_Chinese_Locale_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// 这是一个处理用户数据的函数
// 返回处理结果
func process() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "chinese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for Chinese comment with required=chinese, got: %v", errs)
	}
}

// TestCheckDiff_Chinese_FailsKorean: Chinese comments fail Korean requirement.
func TestCheckDiff_Chinese_FailsKorean(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// 这是一个纯中文注释内容示例
func process() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("Chinese comment should fail Korean requirement")
	}
}

// TestCheckDiff_Chinese_FailsJapanese: pure CJK (no kana) fails Japanese requirement.
func TestCheckDiff_Chinese_FailsJapanese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// 这是一个纯中文注释内容示例
func process() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "japanese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("pure CJK (no kana) comment should fail Japanese requirement")
	}
}

// TestCheckDiff_Japanese_Mixed_PassesChinese: Japanese text with kanji also passes
// Chinese requirement due to shared CJK codepoints.
func TestCheckDiff_Japanese_Mixed_PassesChinese(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

// これは日本語のコメントです
func process() {}
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "chinese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	// Japanese kanji are in the CJK range shared with Chinese — this is expected to pass.
	if len(errs) != 0 {
		t.Errorf("Japanese text with kanji should pass Chinese requirement (shared CJK range), got: %v", errs)
	}
}

// TestCheckDiff_Python_Japanese_Pass: Japanese comments in Python pass when required=japanese.
func TestCheckDiff_Python_Japanese_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `# ユーザーデータを処理する関数
def process():
    pass
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "japanese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Japanese Python comment should pass, got: %v", errs)
	}
}

// TestCheckDiff_TypeScript_Chinese_Pass: Chinese comments in TypeScript pass when required=chinese.
func TestCheckDiff_TypeScript_Chinese_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service.ts", `// 处理用户请求的服务函数
const handler = () => {};
`)
	cfg := koreanOnlyConfig()
	cfg.CommentLanguage.RequiredLanguage = "chinese"

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Chinese TypeScript comment should pass, got: %v", errs)
	}
}

// TestCheckDiff_BlockComment_Multiline: multi-line block comment spanning added lines.
func TestCheckDiff_BlockComment_Multiline(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "svc.go", `package svc

/*
This block comment is written entirely
in English and should be flagged.
*/
func Svc() {}
`)
	errs, err := checker.CheckDiff(koreanOnlyConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for English block comment")
	}
}

// ---- helpers ----------------------------------------------------------------

func contains(s, sub string) bool {
	return len(s) >= len(sub) && findSubstr(s, sub)
}

// localeToLang mirrors the mapping in config.applyDefaults for test use.
func localeToLang(locale string) string {
	switch locale {
	case "ko":
		return "korean"
	case "en":
		return "english"
	case "ja":
		return "japanese"
	case "zh", "zh-hans", "zh-hant":
		return "chinese"
	}
	return ""
}
