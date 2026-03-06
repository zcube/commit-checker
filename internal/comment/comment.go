package comment

import "strings"

// Kind 은 추출된 항목의 종류를 나타냄
type Kind uint8

const (
	KindComment      Kind = iota // 주석
	KindString                   // 문자열 리터럴
	KindImportString             // import/include 문자열 (언어 검사 제외)
)

// Comment 는 소스 코드에서 추출된 주석 또는 문자열 리터럴을 나타냄
type Comment struct {
	Text    string
	Line    int
	EndLine int
	IsBlock bool
	Kind    Kind // KindComment 또는 KindString
}

// Parser 는 소스 코드에서 주석을 추출하는 인터페이스
type Parser interface {
	ParseFile(content string) ([]Comment, error)
	SupportedExtensions() []string
}

// cleanBlockComment 은 블록 주석 본문 각 줄의 선행 별표와 공백을 제거 (JavaDoc / JSDoc 스타일 처리).
func cleanBlockComment(raw string) string {
	lines := strings.Split(raw, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}
