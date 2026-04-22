package comment

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// mdLinkCommentRe 는 [//]: # (text), [//]: # "text", [//]: # 'text' 형식의 마크다운 주석을 감지합니다.
var mdLinkCommentRe = regexp.MustCompile(`^\[//\]: # (?:\(([^)]+)\)|"([^"]+)"|'([^']+)')`)

// MarkdownParser 는 goldmark 기반 마크다운 파서입니다.
// 헤딩·단락·리스트 텍스트와 HTML/마크다운 주석을 KindComment 로 추출합니다.
// 펜스드 코드 블록, 인라인 코드 스팬, HTML 블록(주석 제외)은 건너뜁니다.
type MarkdownParser struct{}

func (p *MarkdownParser) SupportedExtensions() []string {
	return []string{".md", ".markdown"}
}

func (p *MarkdownParser) ParseFile(content string) ([]Comment, error) {
	src := []byte(content)
	md := goldmark.New()
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	var result []Comment

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindFencedCodeBlock, ast.KindCodeBlock:
			return ast.WalkSkipChildren, nil

		case ast.KindHTMLBlock:
			// HTML 주석(<!-- ... -->) 만 추출하고 나머지 HTML 블록은 건너뜁니다.
			raw := mdHTMLBlockText(n, src)
			if commentText, ok := mdExtractHTMLComment(raw); ok && strings.TrimSpace(commentText) != "" {
				lineNo := mdBlockLineNo(n, src)
				result = append(result, Comment{
					Text:    strings.TrimSpace(commentText),
					Line:    lineNo,
					EndLine: mdBlockEndLine(n, src),
					IsBlock: true,
					Kind:    KindComment,
				})
			}
			return ast.WalkSkipChildren, nil

		case ast.KindHeading, ast.KindParagraph:
			extracted := strings.TrimSpace(mdExtractText(n, src))
			if extracted == "" {
				return ast.WalkSkipChildren, nil
			}
			result = append(result, Comment{
				Text:    extracted,
				Line:    mdBlockLineNo(n, src),
				EndLine: mdBlockEndLine(n, src),
				IsBlock: false,
				Kind:    KindComment,
			})
			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	// [//]: # (comment) 형식의 마크다운 주석을 추출합니다.
	// goldmark 는 이를 링크 참조 정의로 소비하므로 소스에서 직접 파싱합니다.
	for i, line := range strings.Split(content, "\n") {
		m := mdLinkCommentRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		// m[1], m[2], m[3] 중 하나가 실제 텍스트입니다.
		commentText := strings.TrimSpace(m[1] + m[2] + m[3])
		if commentText == "" {
			continue
		}
		result = append(result, Comment{
			Text:    commentText,
			Line:    i + 1,
			EndLine: i + 1,
			IsBlock: false,
			Kind:    KindComment,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Line < result[j].Line
	})

	return result, nil
}

// mdExtractText 는 노드 하위에서 재귀적으로 텍스트를 추출합니다.
// 인라인 코드 스팬(KindCodeSpan)과 인라인 HTML(KindRawHTML)은 건너뜁니다.
func mdExtractText(n ast.Node, src []byte) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindCodeSpan, ast.KindRawHTML:
			// 인라인 코드와 인라인 HTML 건너뜀
		case ast.KindText:
			t := child.(*ast.Text)
			buf.Write(t.Segment.Value(src))
			if t.SoftLineBreak() || t.HardLineBreak() {
				buf.WriteRune(' ')
			}
		default:
			buf.WriteString(mdExtractText(child, src))
		}
	}
	return buf.String()
}

// mdHTMLBlockText 는 HTMLBlock 노드의 원본 텍스트를 반환합니다.
func mdHTMLBlockText(n ast.Node, src []byte) string {
	lines := n.Lines()
	if lines == nil {
		return ""
	}
	var buf strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(src))
	}
	return buf.String()
}

// mdExtractHTMLComment 는 <!-- ... --> 형식에서 주석 내용을 추출합니다.
func mdExtractHTMLComment(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "<!--") {
		return "", false
	}
	end := strings.Index(raw, "-->")
	if end < 4 {
		return "", false
	}
	return strings.TrimSpace(raw[4:end]), true
}

// mdBlockLineNo 는 블록 노드의 첫 번째 줄 번호를 반환합니다.
func mdBlockLineNo(n ast.Node, src []byte) int {
	lines := n.Lines()
	if lines != nil && lines.Len() > 0 {
		firstSeg := lines.At(0)
		return bytes.Count(src[:firstSeg.Start], []byte("\n")) + 1
	}
	return 1
}

// mdBlockEndLine 은 블록 노드의 마지막 줄 번호를 반환합니다.
func mdBlockEndLine(n ast.Node, src []byte) int {
	lines := n.Lines()
	if lines == nil || lines.Len() == 0 {
		return mdBlockLineNo(n, src)
	}
	lastSeg := lines.At(lines.Len() - 1)
	endPos := lastSeg.Stop
	if endPos > lastSeg.Start {
		endPos--
	}
	return bytes.Count(src[:endPos], []byte("\n")) + 1
}
