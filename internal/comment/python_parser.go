package comment

import "strings"

// PythonParser extracts # line comments from Python source code.
//
// It handles single-quoted, double-quoted, and triple-quoted string literals
// to avoid treating # inside strings as comments.
type PythonParser struct{}

func (p *PythonParser) SupportedExtensions() []string {
	return []string{".py"}
}

func (p *PythonParser) ParseFile(content string) ([]Comment, error) {
	const (
		stCode    = iota
		stLine    // inside # comment
		stDQ      // inside "..."
		stSQ      // inside '...'
		stTripleDQ // inside """..."""
		stTripleSQ // inside '''...'''
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

	peekN := func(i, offset int) rune {
		if i+offset < n {
			return runes[i+offset]
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
			case ch == '#':
				state = stLine
				commentLine = line
			case ch == '"' && peekN(i, 1) == '"' && peekN(i, 2) == '"':
				state = stTripleDQ
				i += 2
			case ch == '\'' && peekN(i, 1) == '\'' && peekN(i, 2) == '\'':
				state = stTripleSQ
				i += 2
			case ch == '"':
				state = stDQ
			case ch == '\'':
				state = stSQ
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

		case stDQ:
			if ch == '\n' {
				line++
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++
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

		case stTripleDQ:
			if ch == '\n' {
				line++
			} else if ch == '"' && peekN(i, 1) == '"' && peekN(i, 2) == '"' {
				state = stCode
				i += 2
			}

		case stTripleSQ:
			if ch == '\n' {
				line++
			} else if ch == '\'' && peekN(i, 1) == '\'' && peekN(i, 2) == '\'' {
				state = stCode
				i += 2
			}
		}
	}

	// Handle file ending without trailing newline while in a # comment.
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
