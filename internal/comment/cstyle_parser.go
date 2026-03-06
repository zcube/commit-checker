package comment

import "strings"

// CStyleParser extracts comments from C-style languages (TypeScript, JavaScript,
// Java, Kotlin, C, C++, C#, Swift, Rust, etc.) using a state machine.
//
// It correctly handles:
//   - // line comments
//   - /* block comments */ (including /** JavaDoc / JSDoc */)
//   - "..." and '...' string literals with escape sequences
//   - ` ` template literals (JS/TS) with escape sequences
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
		comments    []Comment
		runes       = []rune(content)
		n           = len(runes)
		state       = stCode
		buf         strings.Builder
		commentLine int
		line        = 1
	)

	peek := func(i int) rune {
		if i+1 < n {
			return runes[i+1]
		}
		return 0
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
			case ch == '\'':
				state = stSQ
			case p.hasTemplate && ch == '`':
				state = stTemplate
			}

		case stLine:
			if ch == '\n' {
				text := strings.TrimSpace(buf.String())
				if text != "" {
					comments = append(comments, Comment{
						Text:    text,
						Line:    commentLine,
						EndLine: line,
						IsBlock: false,
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
				comments = append(comments, Comment{
					Text:    text,
					Line:    commentLine,
					EndLine: line,
					IsBlock: true,
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
				line++
				state = stCode // treat as unterminated
			} else if ch == '\\' && i+1 < n {
				i++ // skip escaped character
			} else if ch == '"' {
				state = stCode
			}

		case stSQ:
			if ch == '\n' {
				line++
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++
			} else if ch == '\'' {
				state = stCode
			}

		case stTemplate:
			if ch == '\n' {
				line++
			} else if ch == '\\' && i+1 < n {
				i++
			} else if ch == '`' {
				state = stCode
			}
			// ${...} expressions are skipped for simplicity; comments inside
			// template expressions won't be caught, which is an acceptable trade-off.
		}
	}

	// Handle file that ends without a trailing newline while in a line comment.
	if state == stLine {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			comments = append(comments, Comment{
				Text:    text,
				Line:    commentLine,
				EndLine: line,
				IsBlock: false,
			})
		}
	}

	return comments, nil
}
