package lint

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError: lint 검증 중 발견된 구문 오류.
type ValidationError struct {
	File    string
	Line    int
	Message string
}

func (e ValidationError) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// ValidateYAML: 콘텐츠가 유효한 YAML인지 검사.
func ValidateYAML(filename, content string) []ValidationError {
	dec := yaml.NewDecoder(strings.NewReader(content))
	var errs []ValidationError
	for {
		var node yaml.Node
		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, ValidationError{
				File:    filename,
				Message: fmt.Sprintf("YAML syntax error: %s", err),
			})
			break
		}
	}
	return errs
}

// ValidateJSON: 콘텐츠가 유효한 JSON인지 검사.
func ValidateJSON(filename, content string) []ValidationError {
	dec := json.NewDecoder(strings.NewReader(content))
	dec.UseNumber()
	// 모든 토큰을 디코딩하여 구문 오류 탐지
	var depth int
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []ValidationError{{
				File:    filename,
				Line:    countLines(content, int(dec.InputOffset())),
				Message: fmt.Sprintf("JSON syntax error: %s", err),
			}}
		}
		switch t {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
	if depth != 0 {
		return []ValidationError{{
			File:    filename,
			Message: "JSON syntax error: unexpected end of input",
		}}
	}
	return nil
}

// ValidateJSON5: JSON5 주석과 trailing comma를 제거한 후 JSON으로 검사.
func ValidateJSON5(filename, content string) []ValidationError {
	stripped, err := StripJSON5Comments(content)
	if err != nil {
		return []ValidationError{{
			File:    filename,
			Message: fmt.Sprintf("JSON5 syntax error: %s", err),
		}}
	}
	return ValidateJSON(filename, stripped)
}

// ValidateXML: 콘텐츠가 올바른 형식의 XML인지 검사.
func ValidateXML(filename, content string) []ValidationError {
	dec := xml.NewDecoder(bytes.NewReader([]byte(content)))
	dec.Strict = true
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return []ValidationError{{
				File:    filename,
				Line:    countLines(content, int(dec.InputOffset())),
				Message: fmt.Sprintf("XML syntax error: %s", err),
			}}
		}
	}
	return nil
}

// StripJSON5Comments: JSON5 콘텐츠에서 // 및 /* */ 주석을 제거하고
// } 또는 ] 앞의 trailing comma를 제거.
func StripJSON5Comments(input string) (string, error) {
	var out strings.Builder
	runes := []rune(input)
	n := len(runes)
	i := 0

	for i < n {
		ch := runes[i]
		switch {
		// 문자열 리터럴 (큰따옴표 또는 작은따옴표)
		case ch == '"' || ch == '\'':
			quote := ch
			out.WriteRune(ch)
			i++
			for i < n {
				c := runes[i]
				out.WriteRune(c)
				i++
				if c == '\\' && i < n {
					out.WriteRune(runes[i])
					i++
				} else if c == quote {
					break
				}
			}

		// 주석 시작 가능 여부 확인
		case ch == '/' && i+1 < n:
			next := runes[i+1]
			if next == '/' {
				// 한 줄 주석: 개행까지 건너뜀
				i += 2
				for i < n && runes[i] != '\n' {
					i++
				}
			} else if next == '*' {
				// 블록 주석: */ 까지 건너뜀
				i += 2
				found := false
				for i+1 < n {
					if runes[i] == '*' && runes[i+1] == '/' {
						i += 2
						found = true
						break
					}
					// 줄 번호 유지를 위해 개행 문자 보존
					if runes[i] == '\n' {
						out.WriteRune('\n')
					}
					i++
				}
				if !found {
					return "", fmt.Errorf("unterminated block comment")
				}
			} else {
				out.WriteRune(ch)
				i++
			}

		default:
			out.WriteRune(ch)
			i++
		}
	}

	// } 또는 ] 앞의 trailing comma 제거
	result := out.String()
	result = stripTrailingCommas(result)
	return result, nil
}

// stripTrailingCommas: } 또는 ] 바로 앞의 쉼표(옵션 공백 포함)를 제거.
func stripTrailingCommas(s string) string {
	runes := []rune(s)
	var out []rune
	n := len(runes)
	for i := 0; i < n; i++ {
		if runes[i] == ',' {
			// 앞을 살펴봄: 공백/개행을 건너뛰고 다음 비공백 문자가 } 또는 ]인지 확인
			j := i + 1
			for j < n && (runes[j] == ' ' || runes[j] == '\t' || runes[j] == '\n' || runes[j] == '\r') {
				j++
			}
			if j < n && (runes[j] == '}' || runes[j] == ']') {
				// trailing comma 건너뜀
				continue
			}
		}
		out = append(out, runes[i])
	}
	return string(out)
}

// countLines: 주어진 바이트 오프셋에서 1-based 줄 번호를 반환.
func countLines(content string, offset int) int {
	if offset > len(content) {
		offset = len(content)
	}
	return strings.Count(content[:offset], "\n") + 1
}

// DefaultJSONIgnoreFiles: JSON lint에서 기본적으로 제외할 파일 패턴.
// lock 파일 및 자동 생성되는 패키지 관리 파일.
var DefaultJSONIgnoreFiles = []string{
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"composer.lock",
	"Pipfile.lock",
	"Gemfile.lock",
	"Cargo.lock",
	"go.sum",
}
