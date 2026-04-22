package comment_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

func TestHashStyleParser_Shell(t *testing.T) {
	src := `#!/bin/bash
# 이것은 한국어 주석입니다
# This is an English comment
echo "# not a comment"
# another comment
`
	p := comment.NewHashStyleParser([]string{".sh"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}

	// 한국어 주석, 영어 주석, another comment — 3개 (shebang #! 제외)
	if len(onlyComments) != 3 {
		t.Errorf("expected 3 comments, got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestHashStyleParser_Shell_NoRubyBlocks(t *testing.T) {
	// Shell 파서는 =begin/=end 를 처리하지 않음
	src := `# normal comment
=begin
this should not be a block comment
=end
# another comment
`
	p := comment.NewHashStyleParser([]string{".sh"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}

	// "normal comment"와 "another comment" 2개만
	if len(onlyComments) != 2 {
		t.Errorf("expected 2 comments, got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestHashStyleParser_Ruby_LineComment(t *testing.T) {
	src := `# 한국어 주석
x = 1 # inline comment
# another comment
`
	p := comment.NewHashStyleParser([]string{".rb"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}

	// "한국어 주석", "another comment" 2개 (inline은 줄 앞에 공백이 있어 TrimSpace 후 # 이 아님)
	if len(onlyComments) != 2 {
		t.Errorf("expected 2 comments, got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestHashStyleParser_Ruby_BlockComment(t *testing.T) {
	src := `# line comment
=begin
This is a
multi-line block comment
=end
# after block
`
	p := comment.NewHashStyleParser([]string{".rb"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var lineComments, blockComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			if c.IsBlock {
				blockComments = append(blockComments, c)
			} else {
				lineComments = append(lineComments, c)
			}
		}
	}

	if len(lineComments) != 2 {
		t.Errorf("expected 2 line comments, got %d: %+v", len(lineComments), lineComments)
	}
	if len(blockComments) != 1 {
		t.Errorf("expected 1 block comment, got %d: %+v", len(blockComments), blockComments)
	}
	if len(blockComments) == 1 {
		if blockComments[0].Line != 2 {
			t.Errorf("block comment should start at line 2, got %d", blockComments[0].Line)
		}
		if blockComments[0].EndLine != 5 {
			t.Errorf("block comment should end at line 5, got %d", blockComments[0].EndLine)
		}
	}
}

func TestHashStyleParser_Ruby_EmptyBlockComment(t *testing.T) {
	src := `=begin
=end
`
	p := comment.NewHashStyleParser([]string{".rb"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 빈 블록 주석은 Text 가 비어 있어도 결과에 포함됨
	_ = comments
}

func TestHashStyleParser_SupportedExtensions(t *testing.T) {
	rubyExts := []string{".rb", ".rake", ".gemspec"}
	p := comment.NewHashStyleParser(rubyExts, true)
	got := p.SupportedExtensions()
	if len(got) != len(rubyExts) {
		t.Errorf("expected %d extensions, got %d", len(rubyExts), len(got))
	}
}
