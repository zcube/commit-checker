package checker_test

// Integration tests for CheckDiffCustomRules, CheckEditorConfig, CheckUnicode.
// Each test spins up a real temporary git repository.

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

// ---- CheckDiffCustomRules ---------------------------------------------------

func TestCheckDiffCustomRules_NoRules(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors when no rules configured, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_ForbiddenPattern_Detected(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {\n\t// TODO: fix this later\n}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "no-todo", Pattern: `TODO`, Message: "TODO를 제거하세요"},
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for TODO in added line, got none")
	}
}

func TestCheckDiffCustomRules_ForbiddenPattern_NoMatch(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "no-todo", Pattern: `TODO`, Message: "TODO를 제거하세요"},
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_RequiredRulesSkipped(t *testing.T) {
	// required=true 규칙은 diff에서 지원하지 않으므로 무시됨
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "must-have", Pattern: `TICKET-\d+`, Required: true},
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("required rules should be skipped in diff check, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "vendor/pkg.go", "package pkg\n\n// TODO: remove\nfunc F() {}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "no-todo", Pattern: `TODO`},
	}
	cfg.Exceptions.GlobalIgnore = []string{"vendor/**"}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("vendor files should be globally ignored, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_InvalidRegexSkipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "bad-regex", Pattern: `[invalid`},
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("invalid regex should be skipped, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_EmptyPatternSkipped(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "empty-pattern", Pattern: ""},
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("empty pattern should be skipped, got: %v", errs)
	}
}

func TestCheckDiffCustomRules_DefaultMessage(t *testing.T) {
	// Message가 비어있을 때 pattern을 메시지로 사용하는지 확인
	dir := newGitRepo(t)
	seedCommit(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {\n\t// FIXME\n}\n")

	cfg := &config.Config{}
	cfg.CustomRules.Diff = []config.CustomRule{
		{Name: "no-fixme", Pattern: `FIXME`}, // 메시지 없음
	}

	errs, err := checker.CheckDiffCustomRules(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for FIXME pattern, got none")
	}
}

// ---- CheckEditorConfig ------------------------------------------------------

func TestCheckEditorConfig_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = falsePtr()

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled, got errs=%v err=%v", errs, err)
	}
}

func TestCheckEditorConfig_NoEditorconfig(t *testing.T) {
	// .editorconfig 없으면 검사 건너뜀
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors when no .editorconfig, got: %v", errs)
	}
}

func TestCheckEditorConfig_NoViolations(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".editorconfig", "[*]\nend_of_line = lf\ninsert_final_newline = true\n")
	gitMust(t, dir, "git", "add", ".editorconfig")
	gitMust(t, dir, "git", "commit", "-m", "add editorconfig")
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no violations for compliant file, got: %v", errs)
	}
}

func TestCheckEditorConfig_MissingFinalNewline(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".editorconfig", "[*]\ninsert_final_newline = true\n")
	gitMust(t, dir, "git", "add", ".editorconfig")
	gitMust(t, dir, "git", "commit", "-m", "add editorconfig")
	// 파일 끝에 개행 없음
	stageFile(t, dir, "main.go", "package main\n\nfunc main() {}")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected violation for missing final newline, got none")
	}
}

func TestCheckEditorConfig_TrailingWhitespace(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".editorconfig", "[*]\ntrim_trailing_whitespace = true\n")
	gitMust(t, dir, "git", "add", ".editorconfig")
	gitMust(t, dir, "git", "commit", "-m", "add editorconfig")
	stageFile(t, dir, "main.go", "package main   \n\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected violation for trailing whitespace, got none")
	}
}

func TestCheckEditorConfig_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".editorconfig", "[*]\ninsert_final_newline = true\n")
	gitMust(t, dir, "git", "add", ".editorconfig")
	gitMust(t, dir, "git", "commit", "-m", "add editorconfig")
	stageFile(t, dir, "vendor/gen.go", "package gen\n\nfunc Gen() {}")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()
	cfg.Exceptions.GlobalIgnore = []string{"vendor/**"}

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("vendor files should be globally ignored, got: %v", errs)
	}
}

func TestCheckEditorConfig_IgnoreFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".editorconfig", "[*]\ninsert_final_newline = true\n")
	gitMust(t, dir, "git", "add", ".editorconfig")
	gitMust(t, dir, "git", "commit", "-m", "add editorconfig")
	stageFile(t, dir, "generated/gen.go", "package gen\n\nfunc Gen() {}")

	cfg := &config.Config{}
	cfg.EditorConfig.Enabled = truePtr()
	cfg.EditorConfig.IgnoreFiles = []string{"generated/**"}

	errs, err := checker.CheckEditorConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("generated files should be ignored via EditorConfig.IgnoreFiles, got: %v", errs)
	}
}

// ---- CheckUnicode -----------------------------------------------------------

func TestCheckUnicode_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.Enabled = falsePtr()

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when disabled, got errs=%v err=%v", errs, err)
	}
}

func TestCheckUnicode_BothChecksDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = falsePtr()
	cfg.Encoding.NoAmbiguousChars = falsePtr()

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil || len(errs) != 0 {
		t.Errorf("expected nil when both checks disabled, got errs=%v err=%v", errs, err)
	}
}

func TestCheckUnicode_CleanFile(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\n// 정상 주석\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()
	cfg.Encoding.NoAmbiguousChars = truePtr()
	cfg.Encoding.Locale = "ko"

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors for clean file, got: %v", errs)
	}
}

func TestCheckUnicode_InvisibleChar_Detected(t *testing.T) {
	dir := newGitRepo(t)
	// ZERO WIDTH SPACE (U+200B) 포함
	stageFile(t, dir, "main.go", "package main\n\n// 텍스트\u200B포함\nfunc main() {}\n")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected invisible char error, got none")
	}
}

func TestCheckUnicode_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "vendor/lib.go", "package lib\n\n// 텍스트\u200B포함\nfunc F() {}\n")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()
	cfg.Exceptions.GlobalIgnore = []string{"vendor/**"}

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("vendor files should be globally ignored, got: %v", errs)
	}
}

func TestCheckUnicode_IgnoreFiles(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "generated/gen.go", "package gen\n\n// 텍스트\u200B포함\nfunc G() {}\n")

	cfg := &config.Config{}
	cfg.Encoding.Enabled = truePtr()
	cfg.Encoding.NoInvisibleChars = truePtr()
	cfg.Encoding.IgnoreFiles = []string{"generated/**"}

	errs, err := checker.CheckUnicode(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("generated files should be ignored, got: %v", errs)
	}
}
