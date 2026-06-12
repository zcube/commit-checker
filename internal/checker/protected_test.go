package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func protectedPathsConfig(paths ...string) *config.Config {
	cfg := &config.Config{}
	cfg.ProtectedPaths.Enabled = true
	cfg.ProtectedPaths.Paths = paths
	return cfg
}

// TestCheckProtectedPaths_Disabled: enabled=false 이면 검사 건너뜀.
func TestCheckProtectedPaths_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "legacy/old.go", "package legacy\n")
	stageFile(t, dir, "legacy/old.go", "package legacy // modified\n")

	cfg := &config.Config{}
	cfg.ProtectedPaths.Enabled = false
	cfg.ProtectedPaths.Paths = []string{"legacy/**"}

	errs, err := checker.CheckProtectedPaths(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled check should return no errors, got: %v", errs)
	}
}

// TestCheckProtectedPaths_Modify: 보호 경로의 파일 수정은 차단.
func TestCheckProtectedPaths_Modify(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "legacy/old.go", "package legacy\n")
	stageFile(t, dir, "legacy/old.go", "package legacy // modified\n")

	errs, err := checker.CheckProtectedPaths(t.Context(), protectedPathsConfig("legacy/**"), stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("modification in protected path should be blocked, got %d errors: %v", len(errs), errs)
	}
}

// TestCheckProtectedPaths_Add: 보호 경로에 새 파일 추가도 차단 (append-only 와 달리 추가 불가).
func TestCheckProtectedPaths_Add(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "README.md", "init\n")
	stageFile(t, dir, "legacy/new.go", "package legacy\n")

	errs, err := checker.CheckProtectedPaths(t.Context(), protectedPathsConfig("legacy/**"), stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("new file in protected path should be blocked, got %d errors: %v", len(errs), errs)
	}
}

// TestCheckProtectedPaths_Delete: 보호 경로의 파일 삭제는 차단.
func TestCheckProtectedPaths_Delete(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "legacy/old.go", "package legacy\n")
	gitMust(t, dir, "git", "rm", "legacy/old.go")

	errs, err := checker.CheckProtectedPaths(t.Context(), protectedPathsConfig("legacy/**"), stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("file deletion in protected path should be blocked, got %d errors: %v", len(errs), errs)
	}
}

// TestCheckProtectedPaths_PathNotMatched: 패턴 외 경로는 검사 제외.
func TestCheckProtectedPaths_PathNotMatched(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "src/main.go", "package main\n")
	stageFile(t, dir, "src/main.go", "package main\n\nfunc main() {}\n")

	errs, err := checker.CheckProtectedPaths(t.Context(), protectedPathsConfig("legacy/**"), stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("file outside protected path should not be checked, got: %v", errs)
	}
}

// TestCheckProtectedPaths_GlobalIgnore: global_ignore 패턴은 검사 제외.
func TestCheckProtectedPaths_GlobalIgnore(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "legacy/old.go", "package legacy\n")
	stageFile(t, dir, "legacy/old.go", "package legacy // modified\n")

	cfg := protectedPathsConfig("legacy/**")
	cfg.Exceptions.GlobalIgnore = []string{"legacy/old.go"}

	errs, err := checker.CheckProtectedPaths(t.Context(), cfg, stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("globally ignored file should be skipped, got: %v", errs)
	}
}

// TestCheckProtectedPaths_MultipleViolations: 여러 파일 위반 시 모두 보고.
func TestCheckProtectedPaths_MultipleViolations(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "legacy/a.go", "package legacy\n\nvar A = 1\n")
	seedCommit(t, dir, "legacy/b.go", "package legacy\n\nvar B = 2\n")
	stageFile(t, dir, "legacy/a.go", "package legacy\n\nvar A = 100\n")
	gitMust(t, dir, "git", "rm", "legacy/b.go")
	// rename 감지를 피하기 위해 삭제된 파일과 전혀 다른 내용으로 새 파일을 추가
	stageFile(t, dir, "legacy/c.txt", "completely different new content\nnothing in common with b\nso git does not detect a rename\n")

	errs, err := checker.CheckProtectedPaths(t.Context(), protectedPathsConfig("legacy/**"), stagedDiff(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 3 {
		t.Errorf("expected 3 errors for 3 violations (modify/delete/add), got %d: %v", len(errs), errs)
	}
}
