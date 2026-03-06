package lint

import (
	"testing"
)

// --- YAML tests ---

func TestValidateYAML_Valid(t *testing.T) {
	content := "key: value\nlist:\n  - item1\n  - item2\n"
	errs := ValidateYAML("test.yaml", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid YAML, got: %v", errs)
	}
}

func TestValidateYAML_Invalid(t *testing.T) {
	content := "key: [invalid: yaml: here"
	errs := ValidateYAML("test.yaml", content)
	if len(errs) == 0 {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateYAML_Empty(t *testing.T) {
	errs := ValidateYAML("test.yaml", "")
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty YAML, got: %v", errs)
	}
}

func TestValidateYAML_MultipleDocuments(t *testing.T) {
	content := "---\nfoo: bar\n---\nbaz: qux\n"
	errs := ValidateYAML("test.yaml", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for multi-doc YAML, got: %v", errs)
	}
}

// --- JSON tests ---

func TestValidateJSON_Valid(t *testing.T) {
	content := `{"key": "value", "list": [1, 2, 3]}`
	errs := ValidateJSON("test.json", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid JSON, got: %v", errs)
	}
}

func TestValidateJSON_Invalid(t *testing.T) {
	content := `{"key": "value",}`
	errs := ValidateJSON("test.json", content)
	if len(errs) == 0 {
		t.Error("expected error for JSON with trailing comma")
	}
}

func TestValidateJSON_InvalidSyntax(t *testing.T) {
	content := `{key: value}`
	errs := ValidateJSON("test.json", content)
	if len(errs) == 0 {
		t.Error("expected error for JSON without quoted keys")
	}
}

func TestValidateJSON_Empty(t *testing.T) {
	// Empty string is technically not valid JSON but we handle it gracefully
	errs := ValidateJSON("test.json", "")
	// Empty is EOF, no tokens - should not crash
	if len(errs) != 0 {
		t.Errorf("expected no error for empty file, got: %v", errs)
	}
}

// --- JSON5 tests ---

func TestValidateJSON5_WithComments(t *testing.T) {
	content := `{
  // single-line comment
  "key": "value",
  /* multi-line
     comment */
  "list": [1, 2, 3],
}`
	errs := ValidateJSON5("test.json", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for JSON5 with comments, got: %v", errs)
	}
}

func TestValidateJSON5_TrailingComma(t *testing.T) {
	content := `{"key": "value", "list": [1, 2, 3,],}`
	errs := ValidateJSON5("test.json", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for JSON5 with trailing commas, got: %v", errs)
	}
}

func TestValidateJSON5_Invalid(t *testing.T) {
	content := `{key: }`
	errs := ValidateJSON5("test.json", content)
	if len(errs) == 0 {
		t.Error("expected error for invalid JSON5")
	}
}

func TestValidateJSON5_CommentInString(t *testing.T) {
	content := `{"url": "http://example.com"}`
	errs := ValidateJSON5("test.json", content)
	if len(errs) != 0 {
		t.Errorf("// inside string should not be treated as comment, got: %v", errs)
	}
}

func TestValidateJSON5_UnterminatedComment(t *testing.T) {
	content := `{"key": "value" /* unterminated`
	errs := ValidateJSON5("test.json", content)
	if len(errs) == 0 {
		t.Error("expected error for unterminated block comment")
	}
}

// --- XML tests ---

func TestValidateXML_Valid(t *testing.T) {
	content := `<?xml version="1.0"?><root><item>value</item></root>`
	errs := ValidateXML("test.xml", content)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid XML, got: %v", errs)
	}
}

func TestValidateXML_Invalid(t *testing.T) {
	content := `<root><unclosed>`
	errs := ValidateXML("test.xml", content)
	if len(errs) == 0 {
		t.Error("expected error for unclosed XML tag")
	}
}

func TestValidateXML_Empty(t *testing.T) {
	errs := ValidateXML("test.xml", "")
	if len(errs) != 0 {
		t.Errorf("expected no errors for empty XML, got: %v", errs)
	}
}

// --- StripJSON5Comments tests ---

func TestStripJSON5Comments_SingleLine(t *testing.T) {
	input := "{\n// comment\n\"key\": \"value\"\n}"
	result, err := StripJSON5Comments(input)
	if err != nil {
		t.Fatalf("StripJSON5Comments error: %v", err)
	}
	if contains(result, "// comment") {
		t.Errorf("comment should be stripped, got: %s", result)
	}
	if !contains(result, "\"key\"") {
		t.Errorf("key should be preserved, got: %s", result)
	}
}

func TestStripJSON5Comments_MultiLine(t *testing.T) {
	input := "{\n/* block\ncomment */\n\"key\": \"value\"\n}"
	result, err := StripJSON5Comments(input)
	if err != nil {
		t.Fatalf("StripJSON5Comments error: %v", err)
	}
	if contains(result, "block") || contains(result, "comment */") {
		t.Errorf("block comment should be stripped, got: %s", result)
	}
}

func TestStripJSON5Comments_PreservesStrings(t *testing.T) {
	input := `{"url": "http://example.com/path"}`
	result, err := StripJSON5Comments(input)
	if err != nil {
		t.Fatalf("StripJSON5Comments error: %v", err)
	}
	if result != input {
		t.Errorf("string content should be preserved, got: %s", result)
	}
}

func TestStripTrailingCommas(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`[1, 2, 3,]`, `[1, 2, 3]`},
		{`{"a": 1,}`, `{"a": 1}`},
		{`[1, 2, 3]`, `[1, 2, 3]`},
		{`{"a": 1, "b": 2,}`, `{"a": 1, "b": 2}`},
	}
	for _, tt := range tests {
		result := stripTrailingCommas(tt.input)
		if result != tt.expected {
			t.Errorf("stripTrailingCommas(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// --- ValidationError ---

func TestValidationError_String(t *testing.T) {
	e := ValidationError{File: "test.json", Line: 5, Message: "syntax error"}
	s := e.String()
	if s != "test.json:5: syntax error" {
		t.Errorf("unexpected string: %s", s)
	}

	e2 := ValidationError{File: "test.json", Message: "error"}
	s2 := e2.String()
	if s2 != "test.json: error" {
		t.Errorf("unexpected string for zero line: %s", s2)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
