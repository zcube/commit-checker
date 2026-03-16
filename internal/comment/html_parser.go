package comment

import "strings"

// HTMLParser 는 HTML/SVG 파일에서 <!-- --> 블록 주석을 추출.
type HTMLParser struct{}

func (p *HTMLParser) SupportedExtensions() []string {
	return []string{".html", ".htm", ".svg"}
}

func (p *HTMLParser) ParseFile(content string) ([]Comment, error) {
	var result []Comment
	runes := []rune(content)
	n := len(runes)
	line := 1

	i := 0
	for i < n {
		ch := runes[i]

		if ch == '\n' {
			line++
			i++
			continue
		}

		// <!-- 시작 탐지
		if ch == '<' && i+3 < n && runes[i+1] == '!' && runes[i+2] == '-' && runes[i+3] == '-' {
			commentLine := line
			i += 4 // <!-- 소비
			var buf strings.Builder

			// --> 끝 탐지
			for i < n {
				if runes[i] == '-' && i+2 < n && runes[i+1] == '-' && runes[i+2] == '>' {
					text := cleanBlockComment(buf.String())
					result = append(result, Comment{
						Text:    text,
						Line:    commentLine,
						EndLine: line,
						IsBlock: true,
						Kind:    KindComment,
					})
					i += 3 // --> 소비
					break
				}
				if runes[i] == '\n' {
					line++
				}
				buf.WriteRune(runes[i])
				i++
			}
			continue
		}

		i++
	}

	return result, nil
}
