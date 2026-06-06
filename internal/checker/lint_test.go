package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func lintConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.Lint.Enabled = &t
	return cfg
}

// TestCheckLint_Disabled: lint disabled returns no errors.
func TestCheckLint_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.yaml", "key: [invalid: yaml")
	f := false
	cfg := &config.Config{}
	cfg.Lint.Enabled = &f

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled lint should return no errors, got: %v", errs)
	}
}

// TestCheckLint_ValidYAML: valid YAML passes.
func TestCheckLint_ValidYAML(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.yaml", "key: value\nlist:\n  - item1\n")

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid YAML should pass, got: %v", errs)
	}
}

// TestCheckLint_InvalidYAML: invalid YAML is flagged.
func TestCheckLint_InvalidYAML(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.yaml", "key: [invalid: yaml: here")

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for invalid YAML")
	}
}

// TestCheckLint_ValidJSON: valid JSON passes.
func TestCheckLint_ValidJSON(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.json", `{"key": "value"}`)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid JSON should pass, got: %v", errs)
	}
}

// TestCheckLint_InvalidJSON: invalid JSON is flagged.
func TestCheckLint_InvalidJSON(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.json", `{key: value}`)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for invalid JSON")
	}
}

// TestCheckLint_JSON5Allowed: JSON5 with comments passes when allow_json5=true.
func TestCheckLint_JSON5Allowed(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.json", "{\n// comment\n\"key\": \"value\",\n}")

	cfg := lintConfig()
	tr := true
	cfg.Lint.JSON.AllowJSON5 = &tr

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("JSON5 should pass with allow_json5=true, got: %v", errs)
	}
}

// TestCheckLint_JSON5Disallowed: JSON with comments fails when allow_json5=false.
func TestCheckLint_JSON5Disallowed(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.json", "{\n// comment\n\"key\": \"value\"\n}")

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("JSON with comments should fail without allow_json5")
	}
}

// TestCheckLint_JSONC_WithComments: .jsonc 파일은 // 주석 포함 시 통과.
func TestCheckLint_JSONC_WithComments(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.jsonc", "{\n// comment\n\"key\": \"value\"\n}")

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf(".jsonc with // comment should pass, got: %v", errs)
	}
}

// TestCheckLint_JSONC_TrailingComma: .jsonc 파일은 trailing comma도 허용 (JSON5로 검사).
func TestCheckLint_JSONC_TrailingComma(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.jsonc", "{\n// comment\n\"key\": \"value\",\n}")

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf(".jsonc with trailing comma should pass, got: %v", errs)
	}
}

// TestCheckLint_JSONC_Invalid: .jsonc 파일의 구문 오류는 검출.
func TestCheckLint_JSONC_Invalid(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.jsonc", `{key: value}`)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for invalid .jsonc")
	}
}

// TestCheckLint_JSONCommentFilter: comment_filter=true이면 .json 파일에서 // 주석 허용 (trailing comma 불허).
func TestCheckLint_JSONCommentFilter(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.json", "{\n// comment\n\"key\": \"value\"\n}")

	cfg := lintConfig()
	tr := true
	cfg.Lint.JSON.CommentFilter = &tr

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("comment_filter=true should allow // in .json, got: %v", errs)
	}
}

// TestCheckLint_JSONCommentFilter_TrailingCommaRejected: comment_filter 모드에서 trailing comma는 오류.
func TestCheckLint_JSONCommentFilter_TrailingCommaRejected(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.json", "{\n// comment\n\"key\": \"value\",\n}")

	cfg := lintConfig()
	tr := true
	cfg.Lint.JSON.CommentFilter = &tr

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("comment_filter mode should reject trailing comma")
	}
}

// TestCheckLint_YAMLCommentFilter_SkipLint: comment_filter=true이면 skip-lint 주석으로 YAML 검사 비활성화.
func TestCheckLint_YAMLCommentFilter_SkipLint(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.yaml", "# commit-checker: skip-lint\nkey: [invalid yaml structure\n")

	cfg := lintConfig()
	tr := true
	cfg.Lint.YAML.CommentFilter = &tr

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("skip-lint comment should disable YAML lint, got: %v", errs)
	}
}

// TestCheckLint_ValidXML: valid XML passes.
func TestCheckLint_ValidXML(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "config.xml", `<?xml version="1.0"?><root><item>value</item></root>`)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid XML should pass, got: %v", errs)
	}
}

// TestCheckLint_InvalidXML: invalid XML is flagged.
func TestCheckLint_InvalidXML(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.xml", `<root><unclosed>`)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for invalid XML")
	}
}

// TestCheckLint_IgnoreFiles: files matching ignore patterns are skipped.
func TestCheckLint_IgnoreFiles(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "generated.json", `{invalid}`)

	cfg := lintConfig()
	cfg.Lint.JSON.IgnoreFiles = []string{"generated.json"}

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ignored file should not be flagged, got: %v", errs)
	}
}

// TestCheckLint_YAMLDisabled: YAML check can be disabled independently.
func TestCheckLint_YAMLDisabled(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "bad.yaml", "key: [invalid: yaml")

	cfg := lintConfig()
	f := false
	cfg.Lint.YAML.Enabled = &f

	errs, err := checker.CheckLint(cfg)
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled YAML lint should not flag errors, got: %v", errs)
	}
}

// TestCheckLint_NoStagedChanges: no staged files means no errors.
func TestCheckLint_NoStagedChanges(t *testing.T) {
	_ = newGitRepo(t)

	errs, err := checker.CheckLint(lintConfig())
	if err != nil {
		t.Fatalf("CheckLint error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("no staged changes should produce no errors, got: %v", errs)
	}
}
