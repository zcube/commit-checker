package checker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func encodingConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.Encoding.Enabled = &t
	cfg.Encoding.RequireUTF8 = &t
	return cfg
}

// TestCheckEncoding_ValidUTF8: UTF-8 text passes.
func TestCheckEncoding_ValidUTF8(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\n// 한국어 주석\nfunc main() {}\n")

	errs, err := checker.CheckEncoding(encodingConfig())
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid UTF-8 should pass, got: %v", errs)
	}
}

// TestCheckEncoding_InvalidUTF8: non-UTF-8 text is flagged.
func TestCheckEncoding_InvalidUTF8(t *testing.T) {
	dir := newGitRepo(t)
	// Write Latin-1 encoded bytes directly
	full := filepath.Join(dir, "legacy.txt")
	latin1 := []byte("Hello W\xf6rld\n") // ö in Latin-1
	if err := os.WriteFile(full, latin1, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "legacy.txt")

	errs, err := checker.CheckEncoding(encodingConfig())
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected error for non-UTF-8 file")
	}
}

// TestCheckEncoding_Disabled: disabled encoding check returns no errors.
func TestCheckEncoding_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	full := filepath.Join(dir, "legacy.txt")
	latin1 := []byte("Hello W\xf6rld\n")
	if err := os.WriteFile(full, latin1, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "legacy.txt")

	f := false
	cfg := &config.Config{}
	cfg.Encoding.Enabled = &f

	errs, err := checker.CheckEncoding(cfg)
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled check should return no errors, got: %v", errs)
	}
}

// TestCheckEncoding_IgnoreFiles: files matching ignore patterns are skipped.
func TestCheckEncoding_IgnoreFiles(t *testing.T) {
	dir := newGitRepo(t)
	full := filepath.Join(dir, "legacy.txt")
	latin1 := []byte("Hello W\xf6rld\n")
	if err := os.WriteFile(full, latin1, 0644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, dir, "git", "add", "legacy.txt")

	cfg := encodingConfig()
	cfg.Encoding.IgnoreFiles = []string{"legacy.txt"}

	errs, err := checker.CheckEncoding(cfg)
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ignored file should not be flagged, got: %v", errs)
	}
}

// TestCheckEncoding_BinarySkipped: binary files are skipped by encoding check.
func TestCheckEncoding_BinarySkipped(t *testing.T) {
	dir := newGitRepo(t)
	writeBinaryFile(t, dir, "app.exe")
	gitMust(t, dir, "git", "add", "app.exe")

	errs, err := checker.CheckEncoding(encodingConfig())
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("binary files should be skipped by encoding check, got: %v", errs)
	}
}

// TestCheckEncoding_NoStagedChanges: no staged files means no errors.
func TestCheckEncoding_NoStagedChanges(t *testing.T) {
	_ = newGitRepo(t)

	errs, err := checker.CheckEncoding(encodingConfig())
	if err != nil {
		t.Fatalf("CheckEncoding error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("no staged changes should produce no errors, got: %v", errs)
	}
}
