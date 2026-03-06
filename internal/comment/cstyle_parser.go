package comment

import "strings"

// CStyleParser 는 C 스타일 언어(TypeScript, JavaScript, Java, Kotlin, C, C++, C#, Swift, Rust 등)에서
// 주석과 문자열 리터럴을 추출하는 상태 기계 기반 파서.
type CStyleParser struct {
	extensions  []string
	hasTemplate bool // JS/TS have template literals (backticks)
}

// NewCStyleParser creates a parser for the given extensions.
// Set hasTemplate=true for JS/TS to handle backtick template literals.
func NewCStyleParser(extensions []string, hasTemplate bool) *CStyleParser {
	return &CStyleParser{extensions: extensions, hasTemplate: hasTemplate}
}

func (p *CStyleParser) SupportedExtensions() []string {
	return p.extensions
}

func (p *CStyleParser) ParseFile(content string) ([]Comment, error) {
	const (
		stCode     = iota
		stLine     // inside // comment
		stBlock    // inside /* comment
		stDQ       // inside "..." string
		stSQ       // inside '...' string (also char literal)
		stTemplate // inside `...` template literal
	)

	var (
		result      []Comment
		runes       = []rune(content)
		n           = len(runes)
		state       = stCode
		buf         strings.Builder
		commentLine int
		strLine     int
		line        = 1
	)

	peek := func(i int) rune {
		if i+1 < n {
			return runes[i+1]
		}
		return 0
	}

	emitString := func(endLine int) {
		val := buf.String()
		if val != "" {
			result = append(result, Comment{
				Text:    val,
				Line:    strLine,
				EndLine: endLine,
				IsBlock: false,
				Kind:    KindString,
			})
		}
		buf.Reset()
	}

	for i := 0; i < n; i++ {
		ch := runes[i]

		switch state {
		case stCode:
			switch {
			case ch == '\n':
				line++
			case ch == '/' && peek(i) == '/':
				state = stLine
				commentLine = line
				i++ // consume second '/'
			case ch == '/' && peek(i) == '*':
				state = stBlock
				commentLine = line
				i++ // consume '*'
			case ch == '"':
				state = stDQ
				strLine = line
				buf.Reset()
			case ch == '\'':
				state = stSQ
				strLine = line
				buf.Reset()
			case p.hasTemplate && ch == '`':
				state = stTemplate
				strLine = line
				buf.Reset()
			}

		case stLine:
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
				state = stCode
				line++
			} else {
				buf.WriteRune(ch)
			}

		case stBlock:
			if ch == '*' && peek(i) == '/' {
				text := cleanBlockComment(buf.String())
				result = append(result, Comment{
					Text:    text,
					Line:    commentLine,
					EndLine: line,
					IsBlock: true,
					Kind:    KindComment,
				})
				buf.Reset()
				state = stCode
				i++ // consume '/'
			} else {
				if ch == '\n' {
					line++
				}
				buf.WriteRune(ch)
			}

		case stDQ:
			if ch == '\n' {
				emitString(line)
				line++
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++ // 이스케이프 시퀀스는 언어 감지에 불필요하므로 건너뜀
			} else if ch == '"' {
				emitString(line)
				state = stCode
			} else {
				buf.WriteRune(ch)
			}

		case stSQ:
			if ch == '\n' {
				emitString(line)
				line++
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++
			} else if ch == '\'' {
				emitString(line)
				state = stCode
			} else {
				buf.WriteRune(ch)
			}

		case stTemplate:
			if ch == '\n' {
				line++
				buf.WriteRune(ch)
			} else if ch == '\\' && i+1 < n {
				i++
			} else if ch == '`' {
				emitString(line)
				state = stCode
			} else {
				buf.WriteRune(ch)
			}
			// ${...} 표현식은 단순화를 위해 건너뜀.
		}
	}

	// 파일이 개행 없이 끝날 때 line comment 처리
	if state == stLine {
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

	// 파일이 개행 없이 끝날 때 string 처리
	if state == stDQ || state == stSQ || state == stTemplate {
		emitString(line)
	}

	return result, nil
}
