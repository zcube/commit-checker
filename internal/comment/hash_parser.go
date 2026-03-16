package comment

import "strings"

// HashStyleParser 는 # 줄 주석을 사용하는 언어(Ruby, Shell 등)에서 주석을 추출.
// Ruby 는 =begin/=end 멀티라인 블록 주석도 지원.
type HashStyleParser struct {
	exts       []string
	rubyBlocks bool // =begin/=end 블록 주석 지원 여부 (Ruby 전용)
}

// NewHashStyleParser 는 주어진 확장자를 처리하는 파서를 생성합니다.
// rubyBlocks=true 이면 =begin/=end 블록 주석을 처리합니다.
func NewHashStyleParser(exts []string, rubyBlocks bool) *HashStyleParser {
	return &HashStyleParser{exts: exts, rubyBlocks: rubyBlocks}
}

func (p *HashStyleParser) SupportedExtensions() []string {
	return p.exts
}

func (p *HashStyleParser) ParseFile(content string) ([]Comment, error) {
	var result []Comment
	lines := strings.Split(content, "\n")

	inBlock := false
	blockStart := 0
	var blockBuf strings.Builder

	for lineNum, rawLine := range lines {
		lineNo := lineNum + 1
		trimmed := strings.TrimSpace(rawLine)

		// =begin/=end 블록 주석 처리 (Ruby 전용)
		if p.rubyBlocks {
			if !inBlock && trimmed == "=begin" {
				inBlock = true
				blockStart = lineNo
				blockBuf.Reset()
				continue
			}
			if inBlock {
				if trimmed == "=end" {
					text := strings.TrimSpace(blockBuf.String())
					result = append(result, Comment{
						Text:    text,
						Line:    blockStart,
						EndLine: lineNo,
						IsBlock: true,
						Kind:    KindComment,
					})
					inBlock = false
					blockBuf.Reset()
				} else {
					if blockBuf.Len() > 0 {
						blockBuf.WriteRune('\n')
					}
					blockBuf.WriteString(rawLine)
				}
				continue
			}
		}

		// # 줄 주석 처리
		if strings.HasPrefix(trimmed, "#") {
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if text != "" {
				result = append(result, Comment{
					Text:    text,
					Line:    lineNo,
					EndLine: lineNo,
					IsBlock: false,
					Kind:    KindComment,
				})
			}
		}
	}

	// 파일이 =end 없이 끝난 경우 처리
	if inBlock && blockBuf.Len() > 0 {
		text := strings.TrimSpace(blockBuf.String())
		if text != "" {
			result = append(result, Comment{
				Text:    text,
				Line:    blockStart,
				EndLine: len(lines),
				IsBlock: true,
				Kind:    KindComment,
			})
		}
	}

	return result, nil
}
