package gitdiff_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/gitdiff"
)

// In unified diff format, blank context lines are represented as a single space " ".
const sampleDiff = "diff --git a/foo.go b/foo.go\n" +
	"index abc1234..def5678 100644\n" +
	"--- a/foo.go\n" +
	"+++ b/foo.go\n" +
	"@@ -1,4 +1,6 @@\n" +
	" package main\n" +
	" \n" + // blank context line (space prefix)
	"+// 새로 추가된 주석\n" + // line 3
	"+\n" + // line 4
	" func old() {}\n" + // line 5 (context)
	"+func newFunc() {}\n" + // line 6
	"\n" +
	"diff --git a/bar.go b/bar.go\n" +
	"new file mode 100644\n" +
	"index 0000000..aabbcc1\n" +
	"--- /dev/null\n" +
	"+++ b/bar.go\n" +
	"@@ -0,0 +1,3 @@\n" +
	"+package main\n" + // line 1
	" \n" + // line 2 (blank context — note: for new file this won't appear, but test the parser)
	"+// 새 파일의 주석\n" // line 3

func TestParseDiff_AddedLines(t *testing.T) {
	diffs := gitdiff.ParseDiff(sampleDiff)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 file diffs, got %d", len(diffs))
	}

	foo := diffs[0]
	if foo.Path != "foo.go" {
		t.Errorf("expected foo.go, got %q", foo.Path)
	}
	// Lines 3, 4, 6 should be added (1-based in the new file)
	for _, wantLine := range []int{3, 4, 6} {
		if !foo.AddedLines[wantLine] {
			t.Errorf("expected line %d to be added in foo.go; addedLines=%v", wantLine, foo.AddedLines)
		}
	}
	// Lines 1, 2, 5 are context — not added
	for _, ctxLine := range []int{1, 2, 5} {
		if foo.AddedLines[ctxLine] {
			t.Errorf("line %d should not be marked as added in foo.go", ctxLine)
		}
	}

	bar := diffs[1]
	if bar.Path != "bar.go" {
		t.Errorf("expected bar.go, got %q", bar.Path)
	}
	if !bar.AddedLines[1] || !bar.AddedLines[3] {
		t.Errorf("bar.go: expected lines 1 and 3 to be added; addedLines=%v", bar.AddedLines)
	}
}

func TestParseDiff_DeletedFile(t *testing.T) {
	diff := "diff --git a/old.go b/old.go\n" +
		"deleted file mode 100644\n" +
		"--- a/old.go\n" +
		"+++ /dev/null\n" +
		"@@ -1,3 +0,0 @@\n" +
		"-package main\n" +
		" \n" +
		"-// removed\n"
	diffs := gitdiff.ParseDiff(diff)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if !diffs[0].IsDeleted {
		t.Error("expected file to be marked as deleted")
	}
}

func TestHasExtension(t *testing.T) {
	exts := []string{".go", ".ts", ".java"}
	cases := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"src/app.ts", true},
		{"Service.java", true},
		{"readme.md", false},
		{"Makefile", false},
	}
	for _, tc := range cases {
		got := gitdiff.HasExtension(tc.path, exts)
		if got != tc.want {
			t.Errorf("HasExtension(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
