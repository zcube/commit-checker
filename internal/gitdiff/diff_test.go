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

func TestParseDiff_Submodule(t *testing.T) {
	raw := "diff --git a/vendor/mod b/vendor/mod\n" +
		"new file mode 160000\n" +
		"index 0000000..abc1234\n" +
		"--- /dev/null\n" +
		"+++ b/vendor/mod\n" +
		"@@ -0,0 +1 @@\n" +
		"+Subproject commit abc1234\n"
	diffs := gitdiff.ParseDiff(raw)
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if !diffs[0].IsSubmodule {
		t.Error("expected IsSubmodule=true")
	}
}

func TestParseDiff_Symlink(t *testing.T) {
	raw := "diff --git a/link b/link\n" +
		"new file mode 120000\n" +
		"index 0000000..abc1234\n" +
		"--- /dev/null\n" +
		"+++ b/link\n" +
		"@@ -0,0 +1 @@\n" +
		"+target/path\n"
	diffs := gitdiff.ParseDiff(raw)
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(diffs))
	}
	if !diffs[0].IsSymlink {
		t.Error("expected IsSymlink=true")
	}
}

// TestParseDiff_QuotedPath: git 이 C-스타일로 인용한 diff 헤더 경로가
// 원형으로 복원되어야 함. git 은 `"`·백슬래시·제어문자 포함 경로를
// core.quotepath 설정과 무관하게 항상 인용한다.
func TestParseDiff_QuotedPath(t *testing.T) {
	t.Run("따옴표 포함 파일명", func(t *testing.T) {
		raw := "diff --git \"a/with\\\"quote.txt\" \"b/with\\\"quote.txt\"\n" +
			"new file mode 100644\n" +
			"index 0000000..abc1234\n" +
			"--- /dev/null\n" +
			"+++ \"b/with\\\"quote.txt\"\n" +
			"@@ -0,0 +1 @@\n" +
			"+x\n"
		diffs := gitdiff.ParseDiff(raw)
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}
		if diffs[0].Path != `with"quote.txt` {
			t.Errorf("Path = %q, want %q", diffs[0].Path, `with"quote.txt`)
		}
	})

	t.Run("옥탈 이스케이프 한글 파일명", func(t *testing.T) {
		// "한글.md" 의 UTF-8 옥탈 표현 (core.quotepath=true 출력 형태)
		raw := "diff --git \"a/\\355\\225\\234\\352\\270\\200.md\" \"b/\\355\\225\\234\\352\\270\\200.md\"\n" +
			"index abc1234..def5678 100644\n" +
			"--- \"a/\\355\\225\\234\\352\\270\\200.md\"\n" +
			"+++ \"b/\\355\\225\\234\\352\\270\\200.md\"\n" +
			"@@ -1 +1,2 @@\n" +
			" # 제목\n" +
			"+본문\n"
		diffs := gitdiff.ParseDiff(raw)
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}
		if diffs[0].Path != "한글.md" {
			t.Errorf("Path = %q, want %q", diffs[0].Path, "한글.md")
		}
		if !diffs[0].AddedLines[2] {
			t.Errorf("line 2 should be added; addedLines=%v", diffs[0].AddedLines)
		}
	})

	t.Run("인용된 삭제 파일 (+++ /dev/null, diff --git 헤더에서 추출)", func(t *testing.T) {
		raw := "diff --git \"a/del\\\"eted.txt\" \"b/del\\\"eted.txt\"\n" +
			"deleted file mode 100644\n" +
			"index abc1234..0000000\n" +
			"--- \"a/del\\\"eted.txt\"\n" +
			"+++ /dev/null\n" +
			"@@ -1 +0,0 @@\n" +
			"-x\n"
		diffs := gitdiff.ParseDiff(raw)
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}
		if diffs[0].Path != `del"eted.txt` {
			t.Errorf("Path = %q, want %q", diffs[0].Path, `del"eted.txt`)
		}
		if !diffs[0].IsDeleted {
			t.Error("expected IsDeleted=true")
		}
	})

	t.Run("비인용 경로는 기존 동작 유지", func(t *testing.T) {
		diffs := gitdiff.ParseDiff(sampleDiff)
		if len(diffs) != 2 || diffs[0].Path != "foo.go" || diffs[1].Path != "bar.go" {
			t.Errorf("unquoted paths broken: %+v", diffs)
		}
	})
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

func TestParseDiff_AppendOnlyFields(t *testing.T) {
	t.Run("new file has IsNew=true and no removed lines", func(t *testing.T) {
		diff := "diff --git a/migrations/001.sql b/migrations/001.sql\n" +
			"new file mode 100644\n" +
			"index 0000000..abc1234\n" +
			"--- /dev/null\n" +
			"+++ b/migrations/001.sql\n" +
			"@@ -0,0 +1,3 @@\n" +
			"+CREATE TABLE users (\n" +
			"+  id SERIAL PRIMARY KEY\n" +
			"+);\n"
		files := gitdiff.ParseDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		f := files[0]
		if !f.IsNew {
			t.Error("IsNew should be true for new file")
		}
		if f.HasRemovedLines {
			t.Error("HasRemovedLines should be false for new file")
		}
	})

	t.Run("deleted file has IsDeleted=true", func(t *testing.T) {
		diff := "diff --git a/migrations/001.sql b/migrations/001.sql\n" +
			"deleted file mode 100644\n" +
			"index abc1234..0000000\n" +
			"--- a/migrations/001.sql\n" +
			"+++ /dev/null\n" +
			"@@ -1,3 +0,0 @@\n" +
			"-CREATE TABLE users (\n" +
			"-  id SERIAL PRIMARY KEY\n" +
			"-);\n"
		files := gitdiff.ParseDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		f := files[0]
		if !f.IsDeleted {
			t.Error("IsDeleted should be true")
		}
		if !f.HasRemovedLines {
			t.Error("HasRemovedLines should be true for deleted file")
		}
	})

	t.Run("append at end has no removed lines", func(t *testing.T) {
		diff := "diff --git a/migrations/001.sql b/migrations/001.sql\n" +
			"index abc1234..def5678 100644\n" +
			"--- a/migrations/001.sql\n" +
			"+++ b/migrations/001.sql\n" +
			"@@ -3,0 +4,2 @@\n" +
			" last existing line\n" +
			"+ALTER TABLE users ADD COLUMN email TEXT;\n" +
			"+ALTER TABLE users ADD COLUMN created_at TIMESTAMP;\n"
		files := gitdiff.ParseDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		f := files[0]
		if f.HasRemovedLines {
			t.Error("HasRemovedLines should be false for pure append")
		}
	})

	t.Run("modification has HasRemovedLines=true", func(t *testing.T) {
		diff := "diff --git a/migrations/001.sql b/migrations/001.sql\n" +
			"index abc1234..def5678 100644\n" +
			"--- a/migrations/001.sql\n" +
			"+++ b/migrations/001.sql\n" +
			"@@ -1,3 +1,3 @@\n" +
			" CREATE TABLE users (\n" +
			"-  id INT PRIMARY KEY\n" +
			"+  id SERIAL PRIMARY KEY\n" +
			" );\n"
		files := gitdiff.ParseDiff(diff)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		f := files[0]
		if !f.HasRemovedLines {
			t.Error("HasRemovedLines should be true for modification")
		}
	})
}
