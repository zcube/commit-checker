package editorconfig

import (
	"fmt"
	"strings"

	ec "github.com/editorconfig/editorconfig-core-go/v2"
)

// Violation: .editorconfig 규칙 위반 정보.
type Violation struct {
	File    string
	Line    int
	Message string
}

func (v Violation) String() string {
	if v.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", v.File, v.Line, v.Message)
	}
	return fmt.Sprintf("%s: %s", v.File, v.Message)
}

// GetDefinition: editorconfig-core-go 라이브러리를 사용하여
// 파일 경로에 대한 editorconfig 정의를 반환.
func GetDefinition(filePath string) (*ec.Definition, error) {
	return ec.GetDefinitionForFilename(filePath)
}

// Check: 파일 콘텐츠를 editorconfig 정의에 따라 검증.
func Check(filename string, content []byte, def *ec.Definition) []Violation {
	if def == nil {
		return nil
	}

	var violations []Violation
	text := string(content)
	lines := strings.Split(text, "\n")

	// charset 검사
	if def.Charset == "utf-8" {
		if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
			violations = append(violations, Violation{
				File:    filename,
				Line:    1,
				Message: "file has UTF-8 BOM but charset=utf-8 (no BOM expected)",
			})
		}
	}

	// 줄 끝 문자 검사
	if def.EndOfLine != "" {
		for i, line := range lines {
			if i == len(lines)-1 {
				break
			}
			hasCR := strings.HasSuffix(line, "\r")
			if def.EndOfLine == ec.EndOfLineLf && hasCR {
				violations = append(violations, Violation{
					File:    filename,
					Line:    i + 1,
					Message: "expected LF line ending, found CRLF",
				})
				break
			}
			if def.EndOfLine == ec.EndOfLineCrLf && !hasCR {
				violations = append(violations, Violation{
					File:    filename,
					Line:    i + 1,
					Message: "expected CRLF line ending, found LF",
				})
				break
			}
		}
	}

	// 파일 끝 개행 검사
	if def.InsertFinalNewline != nil && *def.InsertFinalNewline {
		if len(text) > 0 && text[len(text)-1] != '\n' {
			violations = append(violations, Violation{
				File:    filename,
				Message: "file must end with a newline",
			})
		}
	}

	// 후행 공백 검사
	if def.TrimTrailingWhitespace != nil && *def.TrimTrailingWhitespace {
		for i, line := range lines {
			trimmed := strings.TrimRight(line, " \t\r")
			if trimmed != strings.TrimRight(line, "\r") {
				violations = append(violations, Violation{
					File:    filename,
					Line:    i + 1,
					Message: "trailing whitespace",
				})
				break
			}
		}
	}

	// 들여쓰기 스타일 검사 (처음 100개 들여쓰기 줄 샘플)
	if def.IndentStyle != "" {
		checked := 0
		for i, line := range lines {
			if checked >= 100 {
				break
			}
			trimmed := strings.TrimLeft(line, " \t\r")
			if len(trimmed) == 0 || trimmed == line {
				continue
			}
			checked++
			indent := line[:len(line)-len(trimmed)]

			if def.IndentStyle == "space" && strings.Contains(indent, "\t") {
				violations = append(violations, Violation{
					File:    filename,
					Line:    i + 1,
					Message: "expected spaces for indentation, found tabs",
				})
				break
			}
			if def.IndentStyle == "tab" && !strings.HasPrefix(indent, "\t") && strings.Contains(indent, " ") {
				violations = append(violations, Violation{
					File:    filename,
					Line:    i + 1,
					Message: "expected tabs for indentation, found spaces",
				})
				break
			}
		}
	}

	return violations
}
