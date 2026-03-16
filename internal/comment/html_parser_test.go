package comment_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

func TestHTMLParser_SingleLineComment(t *testing.T) {
	src := `<html>
<!-- This is a comment -->
<body></body>
</html>
`
	p := &comment.HTMLParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d: %+v", len(comments), comments)
	}
	if comments[0].Text != "This is a comment" {
		t.Errorf("unexpected comment text: %q", comments[0].Text)
	}
	if !comments[0].IsBlock {
		t.Errorf("expected IsBlock=true")
	}
}

func TestHTMLParser_MultiLineComment(t *testing.T) {
	src := `<html>
<!--
  Multi-line
  comment here
-->
</html>
`
	p := &comment.HTMLParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d: %+v", len(comments), comments)
	}
	if comments[0].Line != 2 {
		t.Errorf("expected comment to start at line 2, got %d", comments[0].Line)
	}
	if comments[0].EndLine != 5 {
		t.Errorf("expected comment to end at line 5, got %d", comments[0].EndLine)
	}
}

func TestHTMLParser_MultipleComments(t *testing.T) {
	src := `<!-- first -->
<p>text</p>
<!-- second -->
`
	p := &comment.HTMLParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d: %+v", len(comments), comments)
	}
}

func TestHTMLParser_NoComments(t *testing.T) {
	src := `<html><body><p>Hello</p></body></html>`
	p := &comment.HTMLParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestHTMLParser_SupportedExtensions(t *testing.T) {
	p := &comment.HTMLParser{}
	exts := p.SupportedExtensions()
	expected := map[string]bool{".html": true, ".htm": true, ".svg": true}
	for _, ext := range exts {
		if !expected[ext] {
			t.Errorf("unexpected extension: %q", ext)
		}
		delete(expected, ext)
	}
	for ext := range expected {
		t.Errorf("missing extension: %q", ext)
	}
}

func TestHTMLParser_KindIsComment(t *testing.T) {
	src := `<!-- hello world -->`
	p := &comment.HTMLParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Kind != comment.KindComment {
		t.Errorf("expected KindComment, got %v", comments[0].Kind)
	}
}
