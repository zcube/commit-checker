package comment_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

func TestGoParser(t *testing.T) {
	src := `package main

// 이것은 한국어 주석입니다
// This is an English comment

/* 블록 주석
 * 여러 줄
 */

func main() {
	// nolint:errcheck
	x := "// not a comment"
	_ = x
}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) == 0 {
		t.Fatal("expected comments, got none")
	}
	// Check that string content is not picked up as a comment
	for _, c := range comments {
		if c.Text == "not a comment" {
			t.Errorf("string literal was extracted as comment: %q", c.Text)
		}
	}
}

func TestCStyleParser_TypeScript(t *testing.T) {
	src := `// 한국어 주석
const msg = "// 이건 주석 아님";
/* block comment */
const tmpl = ` + "`" + `hello // also not a comment` + "`" + `;
`
	p := comment.NewCStyleParser([]string{".ts"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should find exactly 2 KindComment items: line + block (strings are KindString)
	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}
	if len(onlyComments) != 2 {
		t.Errorf("expected 2 comments, got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestPythonParser(t *testing.T) {
	src := `# 한국어 주석
x = "# 이건 주석 아님"
y = '# 이것도 아님'
z = """
triple quoted
"""
# another comment
`
	p := &comment.PythonParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 문자열 리터럴(KindString)을 제외하고 주석(KindComment)만 2개여야 함
	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}
	if len(onlyComments) != 2 {
		t.Errorf("expected 2 comments, got %d: %+v", len(onlyComments), onlyComments)
	}
}
