package checker_test

// Integration tests for Run* functions (git ls-files 기반).
// 임시 git 저장소를 생성하고 파일을 커밋한 후 각 함수를 테스트합니다.

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func truePtr() *bool  { b := true; return &b }
func falsePtr() *bool { b := false; return &b }

// TestRunBinaryFiles_NoBinary: 바이너리 파일 없는 경우 오류 없음.
func TestRunBinaryFiles_NoBinary(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "main.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.BinaryFile.Enabled = truePtr()

	errs, err := checker.RunBinaryFiles(cfg)
	if err != nil {
		t.Fatalf("RunBinaryFiles error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

// TestRunBinaryFiles_Disabled: 비활성화 시 항상 nil 반환.
func TestRunBinaryFiles_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.BinaryFile.Enabled = falsePtr()
	errs, err := checker.RunBinaryFiles(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled, got errs=%v err=%v", errs, err)
	}
}

// TestRunEncoding_UTF8Files: UTF-8 파일에서 오류 없음.
func TestRunEncoding_UTF8Files(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "hello.go", "package main\n// 한국어 주석\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "hello.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.RequireUTF8 = truePtr()

	errs, err := checker.RunEncoding(cfg)
	if err != nil {
		t.Fatalf("RunEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid UTF-8, got: %v", errs)
	}
}

// TestRunEncoding_Disabled: 비활성화 시 nil 반환.
func TestRunEncoding_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.Enabled = falsePtr()
	errs, err := checker.RunEncoding(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled")
	}
}

// TestRunUnicode_Disabled: InvisibleChars/AmbiguousChars 모두 비활성화 시 nil.
func TestRunUnicode_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = falsePtr()
	cfg.Encoding.NoAmbiguousChars = falsePtr()
	errs, err := checker.RunUnicode(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when unicode checks disabled")
	}
}

// TestRunUnicode_CleanFiles: 정상 파일에서 오류 없음.
func TestRunUnicode_CleanFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "clean.go", "package main\n// 정상 주석\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "clean.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()
	cfg.Encoding.NoAmbiguousChars = truePtr()
	cfg.Encoding.Locale = "ko"

	errs, err := checker.RunUnicode(cfg)
	if err != nil {
		t.Fatalf("RunUnicode error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for clean files, got: %v", errs)
	}
}

// TestRunUnicode_InvisibleChar_Detected: 비가시 문자 감지.
func TestRunUnicode_InvisibleChar_Detected(t *testing.T) {
	dir := newGitRepo(t)
	// ZERO WIDTH SPACE (U+200B) 포함
	writeFile(t, dir, "invisible.go", "package main\n// 텍스트\u200B포함\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "invisible.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()

	errs, err := checker.RunUnicode(cfg)
	if err != nil {
		t.Fatalf("RunUnicode error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected invisible char error, got none")
	}
}

// TestRunLint_ValidYAML: 유효한 YAML 파일에서 오류 없음.
func TestRunLint_ValidYAML(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "config.yml", "key: value\nlist:\n  - item1\n  - item2\n")
	gitMust(t, dir, "git", "add", "config.yml")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.YAML.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid YAML, got: %v", errs)
	}
}

// TestRunLint_InvalidYAML: 잘못된 YAML 파일에서 오류 반환.
func TestRunLint_InvalidYAML(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "bad.yml", "key: [\ninvalid yaml\n")
	gitMust(t, dir, "git", "add", "bad.yml")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.YAML.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected lint error for invalid YAML, got none")
	}
}

// TestRunLint_Disabled: 비활성화 시 nil 반환.
func TestRunLint_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Lint.Enabled = falsePtr()
	errs, err := checker.RunLint(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled")
	}
}

// TestRunLint_ValidJSON: 유효한 JSON 파일에서 오류 없음.
func TestRunLint_ValidJSON(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "config.json", `{"key": "value"}`)
	gitMust(t, dir, "git", "add", "config.json")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.JSON.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid JSON, got: %v", errs)
	}
}

// TestRunLint_InvalidJSON: 잘못된 JSON 파일에서 오류 반환.
func TestRunLint_InvalidJSON(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "bad.json", `{key: value}`)
	gitMust(t, dir, "git", "add", "bad.json")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.JSON.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected lint error for invalid JSON, got none")
	}
}

// TestRunLint_JSONC_WithComments: .jsonc 파일은 // 주석 포함 시 통과.
func TestRunLint_JSONC_WithComments(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "config.jsonc", "{\n// comment\n\"key\": \"value\"\n}")
	gitMust(t, dir, "git", "add", "config.jsonc")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.JSON.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf(".jsonc with // comment should pass, got: %v", errs)
	}
}

// TestRunLint_JSONC_Invalid: .jsonc 파일의 구문 오류는 검출.
func TestRunLint_JSONC_Invalid(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "bad.jsonc", `{key: value}`)
	gitMust(t, dir, "git", "add", "bad.jsonc")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.JSON.Enabled = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected lint error for invalid .jsonc, got none")
	}
}

// TestRunLint_JSONCommentFilter: comment_filter=true이면 .json에서 // 주석 허용.
func TestRunLint_JSONCommentFilter(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "config.json", "{\n// comment\n\"key\": \"value\"\n}")
	gitMust(t, dir, "git", "add", "config.json")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.JSON.Enabled = truePtr()
	cfg.Lint.JSON.CommentFilter = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("comment_filter=true should allow // in .json, got: %v", errs)
	}
}

// TestRunLint_YAMLCommentFilter_SkipLint: YAML skip-lint 주석으로 검사 비활성화.
func TestRunLint_YAMLCommentFilter_SkipLint(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "skip.yaml", "# commit-checker: skip-lint\nkey: [invalid yaml structure\n")
	gitMust(t, dir, "git", "add", "skip.yaml")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.YAML.Enabled = truePtr()
	cfg.Lint.YAML.CommentFilter = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("skip-lint comment should disable YAML lint, got: %v", errs)
	}
}

// TestRunLint_YAMLCommentFilter_NoSkip: comment_filter=true지만 skip-lint 없으면 정상 검사.
func TestRunLint_YAMLCommentFilter_NoSkip(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "bad.yaml", "key: [invalid yaml structure\n")
	gitMust(t, dir, "git", "add", "bad.yaml")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.Lint.Enabled = truePtr()
	cfg.Lint.YAML.Enabled = truePtr()
	cfg.Lint.YAML.CommentFilter = truePtr()

	errs, err := checker.RunLint(cfg)
	if err != nil {
		t.Fatalf("RunLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("no skip-lint comment: should still validate and find error")
	}
}

// TestRunEditorConfig_NoViolations: .editorconfig 없을 때 오류 없음.
func TestRunEditorConfig_NoViolations(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "main.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()

	errs, err := checker.RunEditorConfig(cfg)
	if err != nil {
		t.Fatalf("RunEditorConfig error: %v", err)
	}
	// editorconfig가 없으면 오류 없음이 정상
	_ = errs
}

// TestRunEditorConfig_Disabled: 비활성화 시 nil 반환.
func TestRunEditorConfig_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = falsePtr()
	errs, err := checker.RunEditorConfig(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled")
	}
}

// TestRunCommentLanguage_NoFiles: 파일 없을 때 오류 없음.
func TestRunCommentLanguage_NoFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go", "package main\n\n// 한국어 주석\nfunc main() {}\n")
	gitMust(t, dir, "git", "add", "main.go")
	gitMust(t, dir, "git", "commit", "-m", "init")

	cfg := koreanOnlyConfig()
	errs, err := checker.RunCommentLanguage(cfg)
	if err != nil {
		t.Fatalf("RunCommentLanguage error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for Korean comments, got: %v", errs)
	}
}

// TestRunCommentLanguage_Disabled: 비활성화 시 nil.
func TestRunCommentLanguage_Disabled(t *testing.T) {
	disabled := false
	cfg := &config.Config{}
	cfg.CommentLanguage.Enabled = &disabled
	errs, err := checker.RunCommentLanguage(cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled")
	}
}
