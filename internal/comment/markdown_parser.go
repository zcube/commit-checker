package comment

import "strings"

// MarkdownParser 는 Markdown 파일에서 텍스트 내용을 추출.
// 펜스드 코드 블록(```/~~~)과 인라인 코드(`...`)는 건너뜀.
// 각 비어 있지 않은 텍스트 줄을 KindComment 로 추출.
type MarkdownParser struct{}

func (p *MarkdownParser) SupportedExtensions() []string {
	return []string{".md", ".markdown"}
}

func (p *MarkdownParser) ParseFile(content string) ([]Comment, error) {
	var result []Comment
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMarker := ""

	for lineNum, rawLine := range lines {
		lineNo := lineNum + 1
		trimmed := strings.TrimSpace(rawLine)

		// 펜스드 코드 블록 진입/탈출
		if !inFence {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = true
				fenceMarker = trimmed[:3]
				continue
			}
		} else {
			if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
			}
			continue
		}

		// 인라인 코드 제거 후 텍스트 추출
		text := strings.TrimSpace(stripInlineCode(rawLine))
		if text == "" {
			continue
		}

		result = append(result, Comment{
			Text:    text,
			Line:    lineNo,
			EndLine: lineNo,
			IsBlock: false,
			Kind:    KindComment,
		})
	}

	return result, nil
}

// stripInlineCode 는 `...` 인라인 코드 스팬을 제거.
func stripInlineCode(s string) string {
	var sb strings.Builder
	inCode := false
	for _, ch := range s {
		if ch == '`' {
			inCode = !inCode
			continue
		}
		if !inCode {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}
