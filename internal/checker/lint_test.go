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
