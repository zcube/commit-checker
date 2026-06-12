package checker_test

// check_strings 기능 통합 테스트.
// check_strings=true 여도 문자열 리터럴은 언어 감지 대상이 아님.
// 유니코드 검사는 CheckUnicode/RunUnicode 에서 별도 처리.

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

// TestCheckStrings_Disabled: check_strings=false → 문자열 리터럴은 검사 제외.
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

	errs, err := checker.CheckDiff(t.Context(), cfg)
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("check_strings=false should skip string literals, got: %v", errs)
	}
}

// TestCheckStrings_EnglishString_NoLangError: 영어 문자열 리터럴도 언어 오류를 발생시키지 않음.
func TestCheckStrings_EnglishString_NoLangError(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	msg := "This is an English message for the user"
	_ = msg
}
`)
	errs, err := checker.CheckDiff(t.Context(), checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("string literals should not trigger language errors, got: %v", errs)
	}
}

// TestCheckStrings_KoreanString_Pass: 한국어 문자열 리터럴도 언어 오류 없음.
func TestCheckStrings_KoreanString_Pass(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	msg := "안녕하세요 사용자입니다"
	_ = msg
}
`)
	errs, err := checker.CheckDiff(t.Context(), checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("Korean string literal should pass, got: %v", errs)
	}
}

// TestCheckStrings_Python_EnglishString_NoLangError: Python 영어 문자열도 언어 오류 없음.
func TestCheckStrings_Python_EnglishString_NoLangError(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "utils.py", `def greet():
    msg = "Welcome to the application system"
    return msg
`)
	errs, err := checker.CheckDiff(t.Context(), checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("string literals should not trigger language errors, got: %v", errs)
	}
}

// TestCheckStrings_TypeScript_EnglishString_NoLangError: TypeScript 영어 문자열도 언어 오류 없음.
func TestCheckStrings_TypeScript_EnglishString_NoLangError(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "service.ts", `const msg = "This English message will be shown to users";
export { msg };
`)
	errs, err := checker.CheckDiff(t.Context(), checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("string literals should not trigger language errors, got: %v", errs)
	}
}

// TestCheckStrings_ShortString_NoError: min_length 미만 문자열은 오류 없음.
func TestCheckStrings_ShortString_NoError(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", `package main

func main() {
	code := "hi"
	_ = code
}
`)
	errs, err := checker.CheckDiff(t.Context(), checkStringsConfig())
	if err != nil {
		t.Fatalf("CheckDiff error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("short string should produce no errors, got: %v", errs)
	}
}
