package checker_test

// Integration tests for check_strings feature.
// Each test covers one exception/behavior case for string literal language checking.

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func checkStringsConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.CommentLanguage.Enabled = &t
	cfg.CommentLanguage.RequiredLanguage = "korean"
	cfg.CommentLanguage.MinLength = 5
	cfg.CommentLanguage.CheckMode = "diff"
	cfg.CommentLanguage.Extensions = []string{".go", ".ts", ".py"}
	cfg.CommentLanguage.CheckStrings = &t
	return cfg
}

// TestCheckStrings_Disabled: check_strings=false → 문자열 리터럴은 검사하지 않음.
func TestCheckStrings_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	msg := "This is an English string that should be ignored"
	_ = msg
}
`)
	f := false
	cfg := checkStringsConfig()
	cfg.CommentLanguage.CheckStrings = &f

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("check_strings=false should skip string literals, got: %v", errs)
	}
}

// TestCheckStrings_KoreanString_Pass: Korean string literal passes Korean requirement.
func TestCheckStrings_KoreanString_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	msg := "안녕하세요 사용자입니다"
	_ = msg
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Korean string literal should pass, got: %v", errs)
	}
}

// TestCheckStrings_EnglishString_Fail: English string literal fails Korean requirement.
func TestCheckStrings_EnglishString_Fail(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	msg := "This is an English message for the user"
	_ = msg
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("English string literal should fail Korean requirement")
	}
}

// TestCheckStrings_PathString_Skipped: 슬래시 포함 문자열은 경로로 판단해 건너뜀 (skip_technical_strings 기본값 true).
func TestCheckStrings_PathString_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	path := "/api/v1/users"
	_ = path
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("path string /api/v1/users should be skipped as technical, got: %v", errs)
	}
}

// TestCheckStrings_MimeType_Skipped: MIME 타입 문자열은 슬래시 포함으로 건너뜀.
func TestCheckStrings_MimeType_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	ct := "application/json"
	_ = ct
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("MIME type string should be skipped as technical, got: %v", errs)
	}
}

// TestCheckStrings_UppercaseConstant_Skipped: 소문자 없는 순수 대문자 ASCII 상수는 건너뜀.
func TestCheckStrings_UppercaseConstant_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	code := "ERR_TOKEN_INVALID"
	_ = code
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("uppercase constant string should be skipped as technical, got: %v", errs)
	}
}

// TestCheckStrings_SkipTechnicalStrings_Disabled: skip_technical_strings=false → 경로/상수도 검사함.
func TestCheckStrings_SkipTechnicalStrings_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	path := "/api/v1/users/list"
	_ = path
}
`)
	f := false
	cfg := checkStringsConfig()
	cfg.CommentLanguage.SkipTechnicalStrings = &f

	errs, err := checker.CheckDiff(cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	// 슬래시 포함 문자열이지만 skip_technical_strings=false이므로 검사됨 → 영어로 감지되어 실패
	if len(errs) == 0 {
		t.Error("skip_technical_strings=false should check path strings; expected Korean requirement failure")
	}
}

// TestCheckStrings_Python_EnglishString_Fail: Python 문자열 리터럴도 검사됨.
func TestCheckStrings_Python_EnglishString_Fail(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `def greet():
    msg = "Welcome to the application system"
    return msg
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("English Python string literal should fail Korean requirement")
	}
}

// TestCheckStrings_Python_KoreanString_Pass: Korean Python string literal passes.
func TestCheckStrings_Python_KoreanString_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `def greet():
    msg = "안녕하세요 사용자"
    return msg
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Korean Python string literal should pass, got: %v", errs)
	}
}

// TestCheckStrings_TypeScript_EnglishString_Fail: TypeScript 문자열 리터럴도 검사됨.
func TestCheckStrings_TypeScript_EnglishString_Fail(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service.ts", `const msg = "This English message will be shown to users";
export { msg };
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("English TypeScript string literal should fail Korean requirement")
	}
}

// TestCheckStrings_ShortString_Skipped: min_length 미만 문자열은 건너뜀.
func TestCheckStrings_ShortString_Skipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	code := "hi"
	_ = code
}
`)
	errs, err := checker.CheckDiff(checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("short string below min_length should be skipped, got: %v", errs)
	}
}
