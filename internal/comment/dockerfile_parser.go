package comment

import "strings"

// DockerfileParser 는 Dockerfile에서 # 줄 주석을 추출.
// Dockerfile 은 문자열 리터럴 개념이 없으므로 주석만 파싱.
type DockerfileParser struct{}

func (p *DockerfileParser) SupportedExtensions() []string {
	// "dockerfile" 은 확장자가 아닌 파일명 패턴을 나타내는 특수 식별자.
	// GetParser()와 HasExtension()에서 별도로 처리됨.
	return []string{"dockerfile"}
}

func (p *DockerfileParser) ParseFile(content string) ([]Comment, error) {
	var result []Comment
	runes := []rune(content)
	n := len(runes)
	line := 1

	var buf strings.Builder
	inComment := false
	commentLine := 0

	for i := range n {
		ch := runes[i]

		if inComment {
			if ch == '\n' {
				text := strings.TrimSpace(buf.String())
				if text != "" {
					result = append(result, Comment{
						Text:    text,
						Line:    commentLine,
						EndLine: line,
						IsBlock: false,
						Kind:    KindComment,
					})
				}
				buf.Reset()
				inComment = false
				line++
			} else {
				buf.WriteRune(ch)
			}
			continue
		}

		switch ch {
		case '\n':
			line++
		case '#':
			inComment = true
			commentLine = line
			buf.Reset()
		}
	}

	// 파일이 개행 없이 끝날 때 처리
	if inComment {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			result = append(result, Comment{
				Text:    text,
				Line:    commentLine,
				EndLine: line,
				IsBlock: false,
				Kind:    KindComment,
			})
		}
	}

	return result, nil
}
