package comment

import "strings"

// Comment represents an extracted comment from source code
type Comment struct {
	Text    string
	Line    int
	EndLine int
	IsBlock bool
}

// Parser extracts comments from source code
type Parser interface {
	ParseFile(content string) ([]Comment, error)
	SupportedExtensions() []string
}

// cleanBlockComment strips leading asterisks and whitespace from each line
// of a block comment body (handles JavaDoc / JSDoc style).
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
