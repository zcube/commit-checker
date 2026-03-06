package editorconfig

import (
	"testing"

	ec "github.com/editorconfig/editorconfig-core-go/v2"
)

func boolPtr(b bool) *bool { return &b }

// --- Check tests ---

func TestCheck_FinalNewline(t *testing.T) {
	def := &ec.Definition{}
	def.InsertFinalNewline = boolPtr(true)

	violations := Check("test.go", []byte("hello"), def)
	if len(violations) == 0 {
		t.Error("expected violation for missing final newline")
	}

	violations2 := Check("test.go", []byte("hello\n"), def)
	if len(violations2) != 0 {
		t.Errorf("expected no violation with final newline, got: %v", violations2)
	}
}

func TestCheck_TrailingWhitespace(t *testing.T) {
	def := &ec.Definition{}
	def.TrimTrailingWhitespace = boolPtr(true)

	violations := Check("test.go", []byte("hello   \nworld\n"), def)
	if len(violations) == 0 {
		t.Error("expected violation for trailing whitespace")
	}

	violations2 := Check("test.go", []byte("hello\nworld\n"), def)
	if len(violations2) != 0 {
		t.Errorf("expected no violation without trailing ws, got: %v", violations2)
	}
}

func TestCheck_IndentStyle_Space(t *testing.T) {
	def := &ec.Definition{IndentStyle: "space"}

	violations := Check("test.py", []byte("def foo():\n\treturn 1\n"), def)
	if len(violations) == 0 {
		t.Error("expected violation for tab indentation when space required")
	}

	violations2 := Check("test.py", []byte("def foo():\n    return 1\n"), def)
	if len(violations2) != 0 {
		t.Errorf("expected no violation for space indentation, got: %v", violations2)
	}
}

func TestCheck_IndentStyle_Tab(t *testing.T) {
	def := &ec.Definition{IndentStyle: "tab"}

	violations := Check("test.go", []byte("func foo() {\n    return 1\n}\n"), def)
	if len(violations) == 0 {
		t.Error("expected violation for space indentation when tab required")
	}

	violations2 := Check("test.go", []byte("func foo() {\n\treturn 1\n}\n"), def)
	if len(violations2) != 0 {
		t.Errorf("expected no violation for tab indentation, got: %v", violations2)
	}
}

func TestCheck_EndOfLine_LF(t *testing.T) {
	def := &ec.Definition{EndOfLine: ec.EndOfLineLf}

	violations := Check("test.go", []byte("hello\r\nworld\r\n"), def)
	if len(violations) == 0 {
		t.Error("expected violation for CRLF when LF required")
	}

	violations2 := Check("test.go", []byte("hello\nworld\n"), def)
	if len(violations2) != 0 {
		t.Errorf("expected no violation for LF, got: %v", violations2)
	}
}

func TestCheck_Charset_UTF8_NoBOM(t *testing.T) {
	def := &ec.Definition{Charset: "utf-8"}

	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte("hello\n")...)
	violations := Check("test.go", content, def)
	if len(violations) == 0 {
		t.Error("expected violation for BOM when charset=utf-8")
	}
}

func TestCheck_NilDefinition(t *testing.T) {
	violations := Check("test.go", []byte("hello\n"), nil)
	if len(violations) != 0 {
		t.Errorf("expected no violations for nil definition, got: %v", violations)
	}
}

func TestViolation_String(t *testing.T) {
	v := Violation{File: "test.go", Line: 5, Message: "trailing whitespace"}
	s := v.String()
	if s != "test.go:5: trailing whitespace" {
		t.Errorf("unexpected string: %s", s)
	}
}

func TestViolation_String_NoLine(t *testing.T) {
	v := Violation{File: "test.go", Line: 0, Message: "missing final newline"}
	s := v.String()
	if s != "test.go: missing final newline" {
		t.Errorf("unexpected string: %s", s)
	}
}
