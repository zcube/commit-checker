package gitdiff_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zcube/commit-checker/internal/gitdiff"
)

// newGitRepo creates a temp dir with an initialized git repo and changes CWD.
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	run("git", "config", "commit.gpgsign", "false")

	// initial commit so HEAD exists
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "chore: init")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return dir
}

func stageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", name)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
}

// TestGetStagedDiff_NoChanges: returns empty slice when nothing is staged.
func TestGetStagedDiff_NoChanges(t *testing.T) {
	newGitRepo(t)
	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for clean repo, got %d", len(diffs))
	}
}

// TestGetStagedDiff_WithStagedFile: staged file shows up in diffs.
func TestGetStagedDiff_WithStagedFile(t *testing.T) {
	dir := newGitRepo(t)
	stageFile(t, dir, "main.go", "package main\n\n// Hello world\nfunc main() {}\n")

	diffs, err := gitdiff.GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected at least 1 diff for staged file")
	}
	found := false
	for _, d := range diffs {
		if d.Path == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("main.go not found in diffs: %v", diffs)
	}
}

// TestGetStagedContent_ExistingFile: staged content is returned correctly.
func TestGetStagedContent_ExistingFile(t *testing.T) {
	dir := newGitRepo(t)
	content := "package main\n\n// staged content\n"
	stageFile(t, dir, "staged.go", content)

	got, err := gitdiff.GetStagedContent("staged.go")
	if err != nil {
		t.Fatalf("GetStagedContent: %v", err)
	}
	if got != content {
		t.Errorf("GetStagedContent returned %q, want %q", got, content)
	}
}

// TestGetStagedContent_NonExistentFile: returns error for unstaged path.
func TestGetStagedContent_NonExistentFile(t *testing.T) {
	newGitRepo(t)
	_, err := gitdiff.GetStagedContent("nonexistent_file_xyz.go")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// TestParseHunkHeader_EdgeCases covers lines 114 and 124 of diff.go.
func TestParseDiff_HunkHeader_NoPlus(t *testing.T) {
	// A hunk header with no '+' — parseHunkHeader returns 0.
	diff := "diff --git a/foo.go b/foo.go\n" +
		"+++ b/foo.go\n" +
		"@@ -1,3 @@\n" + // no '+' range
		"+package main\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
}

func TestParseDiff_HunkHeader_NonNumeric(t *testing.T) {
	// A hunk header where the new-file start is non-numeric — parseHunkHeader returns 0.
	diff := "diff --git a/foo.go b/foo.go\n" +
		"+++ b/foo.go\n" +
		"@@ -1,3 +abc,3 @@\n" +
		"+package main\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
}

func TestParseDiff_BeforeFirstDiffHeader(t *testing.T) {
	// Lines before the first "diff --git" header should be skipped (current==nil case).
	diff := "some preamble line\n" +
		"another line\n" +
		"diff --git a/foo.go b/foo.go\n" +
		"+++ b/foo.go\n" +
		"@@ -0,0 +1 @@\n" +
		"+package main\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
}

func TestParseDiff_RenameMetadata(t *testing.T) {
	// rename from/to lines should be treated as metadata and skipped.
	diff := "diff --git a/old.go b/new.go\n" +
		"rename from old.go\n" +
		"rename to new.go\n" +
		"+++ b/new.go\n" +
		"@@ -1 +1 @@\n" +
		" package main\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Path != "new.go" {
		t.Errorf("expected path new.go, got %q", diffs[0].Path)
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	// Binary files should be parsed without panic.
	diff := "diff --git a/img.png b/img.png\n" +
		"index abc..def 100644\n" +
		"Binary files a/img.png and b/img.png differ\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
}
