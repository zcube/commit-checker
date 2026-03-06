package comment_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

// ---- Go parser edge cases ---------------------------------------------------

func TestGoParser_BuildTag_ExtractedButDirective(t *testing.T) {
	// //go:build is extracted by go/parser as a comment, but langdetect skips it
	// because it starts with "go:build". This test verifies the parser extracts it
	// and also finds the real Korean comment alongside it.
	src := `//go:build linux

package main

// 실제 주석입니다
func main() {}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both the build tag and the Korean comment should be extracted.
	if len(comments) < 2 {
		t.Fatalf("expected at least 2 comments (build tag + Korean), got %d: %+v", len(comments), comments)
	}
	// The Korean comment should be present.
	found := false
	for _, c := range comments {
		if containsText(c.Text, "실제 주석") {
			found = true
		}
	}
	if !found {
		t.Error("Korean comment not found in parsed results")
	}
}

func TestGoParser_StringContainingCommentSyntax(t *testing.T) {
	src := `package main

func main() {
	msg := "// this is not a comment"
	other := "/* also not a comment */"
	_ = msg
	_ = other
}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 문자열 리터럴은 KindString으로 추출되어야 하며 KindComment로 추출되면 안 됨
	for _, c := range comments {
		if c.Kind != comment.KindComment {
			continue
		}
		if containsText(c.Text, "this is not a comment") {
			t.Error("string literal content was extracted as comment")
		}
		if containsText(c.Text, "also not a comment") {
			t.Error("block-comment-like string literal was extracted as comment")
		}
	}
}

func TestGoParser_BlockComment_MultiLine(t *testing.T) {
	src := `package main

/*
이것은 여러 줄
블록 주석입니다
*/
func main() {}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 block comment, got %d", len(comments))
	}
	if comments[0].EndLine <= comments[0].Line {
		t.Errorf("block comment should span multiple lines: line=%d endLine=%d",
			comments[0].Line, comments[0].EndLine)
	}
	if !comments[0].IsBlock {
		t.Error("expected IsBlock=true for block comment")
	}
}

func TestGoParser_LineNumbers(t *testing.T) {
	src := `package main

// line 3 comment
func a() {}

// line 6 comment
func b() {}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d: %+v", len(comments), comments)
	}
	if comments[0].Line != 3 {
		t.Errorf("first comment should be on line 3, got %d", comments[0].Line)
	}
	if comments[1].Line != 6 {
		t.Errorf("second comment should be on line 6, got %d", comments[1].Line)
	}
}

func TestGoParser_GodocComment(t *testing.T) {
	src := `package main

// Foo 함수는 무언가를 합니다.
// 두 번째 줄 설명입니다.
func Foo() {}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Go parser merges consecutive line comments into one comment group
	// but each // line is its own comment node
	if len(comments) < 1 {
		t.Fatal("expected at least one comment")
	}
}

// ---- TypeScript/C-style edge cases ------------------------------------------

func TestCStyleParser_TemplateLiteral_NotComment(t *testing.T) {
	// Content inside template literals should NOT be parsed as KindComment
	src := "const a = `hello // this is not a comment`;\n"
	p := comment.NewCStyleParser([]string{".ts"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range comments {
		if c.Kind != comment.KindComment {
			continue
		}
		if containsText(c.Text, "this is not a comment") {
			t.Error("content inside template literal should not be a comment")
		}
	}
}

func TestCStyleParser_DoubleQuoteString_NotComment(t *testing.T) {
	src := `const x = "// not a comment";
// 실제 주석
`
	p := comment.NewCStyleParser([]string{".ts"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// KindComment 는 1개(실제 주석)이어야 함; 문자열은 KindString으로 따로 추출됨
	var onlyComments []comment.Comment
	for _, c := range comments {
		if c.Kind == comment.KindComment {
			onlyComments = append(onlyComments, c)
		}
	}
	if len(onlyComments) != 1 {
		t.Errorf("expected exactly 1 comment (the real one), got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestCStyleParser_BlockComment_StarLines(t *testing.T) {
	// JavaDoc/TSDoc style with leading asterisks should have them stripped
	src := `/**
 * 이 함수는 데이터를 처리합니다.
 * @param data 입력 데이터
 */
function process(data) {}
`
	p := comment.NewCStyleParser([]string{".ts"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 block comment, got %d", len(comments))
	}
	// Leading asterisks should be stripped from comment text
	if containsText(comments[0].Text, "* 이") {
		t.Error("leading asterisks should be stripped from block comment text")
	}
	if !containsText(comments[0].Text, "이 함수는") {
		t.Error("Korean text should be present in comment")
	}
}

func TestCStyleParser_EscapedQuote_InString(t *testing.T) {
	src := `const x = "he said \"// not a comment\" here";
// 진짜 주석
`
	p := comment.NewCStyleParser([]string{".ts"}, true)
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
	if len(onlyComments) != 1 {
		t.Errorf("expected 1 comment, got %d: %+v", len(onlyComments), onlyComments)
	}
}

func TestCStyleParser_LineComment_AfterCode(t *testing.T) {
	src := `int x = 5; // 변수 x 초기화
`
	p := comment.NewCStyleParser([]string{".go"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if !containsText(comments[0].Text, "변수 x") {
		t.Errorf("expected Korean text in comment, got %q", comments[0].Text)
	}
}

func TestCStyleParser_Java_MultipleComments(t *testing.T) {
	src := `public class Main {
    // 첫 번째 메서드
    public void first() {
        // 내부 주석
    }

    // 두 번째 메서드
    public void second() {}
}
`
	p := comment.NewCStyleParser([]string{".java"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("expected 3 comments, got %d: %+v", len(comments), comments)
	}
}

func TestCStyleParser_Rust_LineComment(t *testing.T) {
	src := `// 러스트 함수
fn main() {
    // 내부 주석
    println!("hello");
}
`
	p := comment.NewCStyleParser([]string{".rs"}, false)
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
	if len(onlyComments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(onlyComments))
	}
}

func TestCStyleParser_SingleQuoteString_NotComment(t *testing.T) {
	src := `const ch = '// not a comment';
// 실제 주석
`
	p := comment.NewCStyleParser([]string{".ts"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// KindComment 에는 문자열 내용이 없어야 함
	for _, c := range comments {
		if c.Kind != comment.KindComment {
			continue
		}
		if containsText(c.Text, "not a comment") {
			t.Error("single-quoted string content should not be extracted as comment")
		}
	}
}

// ---- Python parser edge cases -----------------------------------------------

func TestPythonParser_HashInString_NotComment(t *testing.T) {
	src := `x = "# not a comment"
y = '# also not'
# 진짜 주석
`
	p := &comment.PythonParser{}
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
	if len(onlyComments) != 1 {
		t.Errorf("expected 1 comment, got %d: %+v", len(onlyComments), onlyComments)
	}
	if !containsText(onlyComments[0].Text, "진짜 주석") {
		t.Errorf("expected Korean comment, got %q", onlyComments[0].Text)
	}
}

func TestPythonParser_TripleQuoteString_NotComment(t *testing.T) {
	src := `def foo():
    """
    This docstring # is not a comment
    """
    pass
# 실제 주석
`
	p := &comment.PythonParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// docstring 내용은 KindComment로 추출되면 안 됨 (KindString으로는 추출 가능)
	for _, c := range comments {
		if c.Kind != comment.KindComment {
			continue
		}
		if containsText(c.Text, "not a comment") {
			t.Error("docstring content should not be extracted as a comment")
		}
	}
	found := false
	for _, c := range comments {
		if containsText(c.Text, "실제 주석") {
			found = true
		}
	}
	if !found {
		t.Error("real comment not found")
	}
}

func TestPythonParser_InlineComment(t *testing.T) {
	src := `x = 5  # 변수 초기화
`
	p := &comment.PythonParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(comments))
	}
	if !containsText(comments[0].Text, "변수 초기화") {
		t.Errorf("expected Korean text, got %q", comments[0].Text)
	}
}

func TestPythonParser_MultipleComments(t *testing.T) {
	src := `# 첫 번째 주석
def foo():
    # 두 번째 주석
    x = 1  # 세 번째 주석
    return x
`
	p := &comment.PythonParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("expected 3 comments, got %d: %+v", len(comments), comments)
	}
}

func TestPythonParser_LineNumbers(t *testing.T) {
	src := `x = 1
# 두 번째 줄 주석
y = 2
# 네 번째 줄 주석
`
	p := &comment.PythonParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].Line != 2 {
		t.Errorf("first comment should be line 2, got %d", comments[0].Line)
	}
	if comments[1].Line != 4 {
		t.Errorf("second comment should be line 4, got %d", comments[1].Line)
	}
}

// ---- import/include string exclusion tests ----------------------------------

func TestGoParser_ImportStrings_MarkedAsImport(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os/exec"
	"github.com/zcube/commit-checker/internal/comment"
)

func main() {
	msg := "일반 문자열"
	_ = msg
}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var importStrings, normalStrings []comment.Comment
	for _, c := range comments {
		switch c.Kind {
		case comment.KindImportString:
			importStrings = append(importStrings, c)
		case comment.KindString:
			normalStrings = append(normalStrings, c)
		}
	}
	if len(importStrings) != 3 {
		t.Errorf("expected 3 import strings (fmt, os/exec, github...), got %d: %+v", len(importStrings), importStrings)
	}
	if len(normalStrings) != 1 {
		t.Errorf("expected 1 normal string, got %d: %+v", len(normalStrings), normalStrings)
	}
	if len(normalStrings) > 0 && normalStrings[0].Text != "일반 문자열" {
		t.Errorf("expected normal string '일반 문자열', got %q", normalStrings[0].Text)
	}
}

func TestGoParser_SingleImport_MarkedAsImport(t *testing.T) {
	src := `package main

import "fmt"

func main() {}
`
	p := &comment.GoParser{}
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, c := range comments {
		if c.Kind == comment.KindImportString && c.Text == "fmt" {
			return // success
		}
	}
	t.Error("expected import string 'fmt' with KindImportString")
}

func TestCStyleParser_CInclude_MarkedAsImport(t *testing.T) {
	src := `#include "myheader.h"
#include <stdio.h>

int main() {
    char *msg = "일반 문자열";
    return 0;
}
`
	p := comment.NewCStyleParser([]string{".c"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var importStrings, normalStrings []comment.Comment
	for _, c := range comments {
		switch c.Kind {
		case comment.KindImportString:
			importStrings = append(importStrings, c)
		case comment.KindString:
			normalStrings = append(normalStrings, c)
		}
	}
	// #include "myheader.h" should be KindImportString
	if len(importStrings) != 1 {
		t.Errorf("expected 1 import string (myheader.h), got %d: %+v", len(importStrings), importStrings)
	}
	// "일반 문자열" should remain as KindString
	if len(normalStrings) != 1 {
		t.Errorf("expected 1 normal string, got %d: %+v", len(normalStrings), normalStrings)
	}
}

func TestCStyleParser_JSImport_MarkedAsImport(t *testing.T) {
	src := `import React from "react";
import { useState } from "react";
const msg = "일반 문자열";
const lib = require("lodash");
export { default } from "other-module";
`
	p := comment.NewCStyleParser([]string{".js"}, true)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var importStrings, normalStrings []comment.Comment
	for _, c := range comments {
		switch c.Kind {
		case comment.KindImportString:
			importStrings = append(importStrings, c)
		case comment.KindString:
			normalStrings = append(normalStrings, c)
		}
	}
	// "react", "react", "lodash", "other-module" should be import strings
	if len(importStrings) != 4 {
		t.Errorf("expected 4 import strings, got %d: %+v", len(importStrings), importStrings)
	}
	// "일반 문자열" should be normal string
	if len(normalStrings) != 1 {
		t.Errorf("expected 1 normal string, got %d: %+v", len(normalStrings), normalStrings)
	}
}

func TestCStyleParser_RustInclude_MarkedAsImport(t *testing.T) {
	src := `use std::io;

fn main() {
    let data = include_str!("data.txt");
    let msg = "일반 문자열";
}
`
	p := comment.NewCStyleParser([]string{".rs"}, false)
	comments, err := p.ParseFile(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var importStrings, normalStrings []comment.Comment
	for _, c := range comments {
		switch c.Kind {
		case comment.KindImportString:
			importStrings = append(importStrings, c)
		case comment.KindString:
			normalStrings = append(normalStrings, c)
		}
	}
	// "data.txt" should be import string
	if len(importStrings) != 1 {
		t.Errorf("expected 1 import string (data.txt), got %d: %+v", len(importStrings), importStrings)
	}
	// "일반 문자열" should be normal string
	if len(normalStrings) != 1 {
		t.Errorf("expected 1 normal string, got %d: %+v", len(normalStrings), normalStrings)
	}
}

// ---- helper -----------------------------------------------------------------

func containsText(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
