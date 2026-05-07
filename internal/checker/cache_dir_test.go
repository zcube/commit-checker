package checker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

// TestCheckCacheDirStaged_NodeModules: package.json 옆 node_modules 안 파일 staged → 차단.
func TestCheckCacheDirStaged_NodeModules(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "package.json", "{}\n")
	stageFile(t, dir, "node_modules/lodash/index.js", "module.exports = {};\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected node_modules staged file to be blocked")
	}
}

// TestCheckCacheDirStaged_RegularSrcFile: 일반 소스 파일 → 통과.
func TestCheckCacheDirStaged_RegularSrcFile(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "src/main.go", "package main\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("regular src file should pass, got: %v", errs)
	}
}

// TestCheckCacheDirStaged_BuildWithoutIndicator: 부모에 인디케이터 없는 build/ → 통과 (false positive 방지).
func TestCheckCacheDirStaged_BuildWithoutIndicator(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "build/result.txt", "intentional build directory\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("build/ without indicator should not be flagged, got: %v", errs)
	}
}

// TestCheckCacheDirStaged_Disabled: enabled=false → 검사 건너뜀.
func TestCheckCacheDirStaged_Disabled(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "package.json", "{}\n")
	stageFile(t, dir, "node_modules/lodash/index.js", "module.exports = {};\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	disabled := false
	cfg := &config.Config{}
	cfg.CacheDir.Enabled = &disabled

	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("disabled check should return no errors, got: %v", errs)
	}
}

// TestCheckCacheDirStaged_IgnoreDirs: ignore_dirs 에 vendor 추가 → vendor/ 통과.
func TestCheckCacheDirStaged_IgnoreDirs(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "go.mod", "module test\n")
	stageFile(t, dir, "vendor/foo/foo.go", "package foo\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	cfg.CacheDir.IgnoreDirs = []string{"vendor"}

	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ignored vendor dir should pass, got: %v", errs)
	}
}

// TestCheckCacheDirStaged_DeduplicatesPerDir: 같은 디렉터리 내 다수 파일 → 한 번만 보고.
func TestCheckCacheDirStaged_DeduplicatesPerDir(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "package.json", "{}\n")
	stageFile(t, dir, "node_modules/a/index.js", "module.exports = {};\n")
	stageFile(t, dir, "node_modules/b/index.js", "module.exports = {};\n")
	stageFile(t, dir, "node_modules/c/index.js", "module.exports = {};\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirStaged(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 deduplicated error, got %d: %v", len(errs), errs)
	}
}

// TestCheckCacheDirCommitted_TrackedNodeModules: 이미 커밋된 node_modules → 보고.
func TestCheckCacheDirCommitted_TrackedNodeModules(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "package.json", "{}\n")
	seedCommit(t, dir, "node_modules/lodash/index.js", "module.exports = {};\n")

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirCommitted(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("committed node_modules should be reported")
	}
}

// TestCheckCacheDirCommitted_OnlyUntracked: 미추적 파일만 있으면 보고하지 않음.
func TestCheckCacheDirCommitted_OnlyUntracked(t *testing.T) {
	dir := newGitRepo(t)
	seedCommit(t, dir, "package.json", "{}\n")
	// node_modules 는 working tree 에만 존재, 커밋되지 않음
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "lib", "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(dir)

	cfg := &config.Config{}
	errs, err := checker.CheckCacheDirCommitted(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("only untracked cache dir should not be reported, got: %v", errs)
	}
}
