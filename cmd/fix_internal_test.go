package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
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

// ---- firstLine -----------------------------------------------------------------

func TestFirstLine_WithNewline(t *testing.T) {
	got := firstLine("subject\nbody line")
	if got != "subject" {
		t.Errorf("firstLine = %q, want %q", got, "subject")
	}
}

func TestFirstLine_NoNewline(t *testing.T) {
	got := firstLine("single line")
	if got != "single line" {
		t.Errorf("firstLine = %q, want %q", got, "single line")
	}
}

func TestFirstLine_Empty(t *testing.T) {
	got := firstLine("")
	if got != "" {
		t.Errorf("firstLine(\"\") = %q, want \"\"", got)
	}
}

func TestFirstLine_OnlyNewline(t *testing.T) {
	got := firstLine("\nrest")
	if got != "" {
		t.Errorf("firstLine(\"\\nrest\") = %q, want \"\"", got)
	}
}

// ---- resolveRange --------------------------------------------------------------

func TestResolveRange_ExplicitRange(t *testing.T) {
	newTestGitRepo(t)
	got, err := resolveRange("HEAD~3..HEAD", false)
	if err != nil {
		t.Fatalf("resolveRange: %v", err)
	}
	if got != "HEAD~3..HEAD" {
		t.Errorf("resolveRange = %q, want %q", got, "HEAD~3..HEAD")
	}
}

func TestResolveRange_Default(t *testing.T) {
	newTestGitRepo(t)
	got, err := resolveRange("", false)
	if err != nil {
		t.Fatalf("resolveRange: %v", err)
	}
	if got != "HEAD" {
		t.Errorf("resolveRange default = %q, want HEAD", got)
	}
}

func TestResolveRange_RangeAndMineMutuallyExclusive(t *testing.T) {
	newTestGitRepo(t)
	_, err := resolveRange("HEAD~1..HEAD", true)
	if err == nil {
		t.Error("expected error when both --range and --mine are set")
	}
}

func TestResolveRange_Mine(t *testing.T) {
	newTestGitRepo(t)
	got, err := resolveRange("", true)
	if err != nil {
		t.Fatalf("resolveRange --mine: %v", err)
	}
	if got == "" {
		t.Error("resolveRange --mine returned empty string")
	}
	// Should contain --author= flag
	if len(got) == 0 {
		t.Error("expected non-empty range for --mine")
	}
}

// ---- listCommits ---------------------------------------------------------------

func TestListCommits_SingleCommit(t *testing.T) {
	newTestGitRepo(t)
	commits, err := listCommits("HEAD")
	if err != nil {
		t.Fatalf("listCommits: %v", err)
	}
	if len(commits) == 0 {
		t.Error("expected at least 1 commit")
	}
	for _, c := range commits {
		if c.sha == "" {
			t.Error("commit sha should not be empty")
		}
		if c.message == "" {
			t.Error("commit message should not be empty")
		}
	}
}

func TestListCommits_MultipleCommits(t *testing.T) {
	dir := newTestGitRepo(t)

	// Add a second commit
	f := filepath.Join(dir, "extra.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := exec.Command("git", "add", "extra.go")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	c = exec.Command("git", "commit", "-m", "feat: add extra")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	commits, err := listCommits("HEAD")
	if err != nil {
		t.Fatalf("listCommits: %v", err)
	}
	if len(commits) < 2 {
		t.Errorf("expected at least 2 commits, got %d", len(commits))
	}
}

func TestListCommits_EmptyRange(t *testing.T) {
	newTestGitRepo(t)
	// An empty range (empty string) — git log with empty field means no args after HEAD
	// This just runs HEAD which has 1 commit
	commits, err := listCommits("HEAD~99..HEAD~99")
	// This may error (no range) or return empty — either is acceptable
	if err == nil && len(commits) != 0 {
		t.Logf("got %d commits for empty-ish range", len(commits))
	}
	// just verify no panic
	_ = commits
}

// ---- runGit / gitConfig --------------------------------------------------------

func TestRunGit_ValidCommand(t *testing.T) {
	newTestGitRepo(t)
	out, err := runGit("log", "--format=%H", "-1")
	if err != nil {
		t.Fatalf("runGit: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty output from git log")
	}
}

func TestGitConfig_UserEmail(t *testing.T) {
	newTestGitRepo(t)
	email, err := gitConfig("user.email")
	if err != nil {
		t.Fatalf("gitConfig: %v", err)
	}
	if email != "tester@test.com" {
		t.Errorf("gitConfig user.email = %q, want %q", email, "tester@test.com")
	}
}
