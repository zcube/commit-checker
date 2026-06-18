package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestGitRepo creates a temp git repo with an initial commit and changes CWD.
func newTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		// GIT_CONFIG_GLOBAL을 빈 fixture로 교체해 전역 hook.* 설정을 차단한다.
		c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=testdata/empty.gitconfig")
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "tester@test.com")
	run("git", "config", "user.name", "Tester")
	run("git", "config", "commit.gpgsign", "false")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "chore: init")

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

// ---- getStagedFilesForFix ------------------------------------------------------

func TestGetStagedFilesForFix_Empty(t *testing.T) {
	newTestGitRepo(t)
	files, err := getStagedFilesForFix()
	if err != nil {
		t.Fatalf("getStagedFilesForFix: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 staged files on clean repo, got %v", files)
	}
}

func TestGetStagedFilesForFix_WithStagedFile(t *testing.T) {
	dir := newTestGitRepo(t)

	f := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := exec.Command("git", "add", "hello.go")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	files, err := getStagedFilesForFix()
	if err != nil {
		t.Fatalf("getStagedFilesForFix: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected staged file in result")
	}
	found := false
	for _, f := range files {
		if strings.Contains(f, "hello.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("hello.go not found in staged files: %v", files)
	}
}
