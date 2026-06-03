package comment

import (
	"strings"
	"unicode"
)

// PythonParser 는 Python 소스 코드에서 # 줄 주석과 문자열 리터럴을 추출.
// 단일/이중 따옴표 문자열과 삼중 따옴표 문자열(docstring 포함)을 모두 처리.
// PEP 723 인라인 스크립트 메타데이터 블록(# /// <type> ... # ///)은 KindImport로 표시되어
// 언어 검사에서 제외됩니다.
type PythonParser struct{}

func (p *PythonParser) SupportedExtensions() []string {
	return []string{".py"}
}

func (p *PythonParser) ParseFile(content string) ([]Comment, error) {
	const (
		stCode     = iota
		stLine     // # 주석 내부
		stDQ       // "..." 내부
		stSQ       // '...' 내부
		stTripleDQ // """...""" 내부
		stTripleSQ // '''...''' 내부
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

	peekN := func(i, offset int) rune {
		if i+offset < n {
			return runes[i+offset]
		}
		return 0
	}

	emitString := func(endLine int) {
		val := buf.String()
		if val != "" {
			result = append(result, Comment{
				Text:    strings.TrimSpace(val),
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
			case ch == '#':
				state = stLine
				commentLine = line
			case ch == '"' && peekN(i, 1) == '"' && peekN(i, 2) == '"':
				state = stTripleDQ
				strLine = line
				buf.Reset()
				i += 2
			case ch == '\'' && peekN(i, 1) == '\'' && peekN(i, 2) == '\'':
				state = stTripleSQ
				strLine = line
				buf.Reset()
				i += 2
			case ch == '"':
				state = stDQ
				strLine = line
				buf.Reset()
			case ch == '\'':
				state = stSQ
				strLine = line
				buf.Reset()
			}

		case stLine:
			if ch == '\n' {
				text := strings.TrimSpace(buf.String())
				// 셔뱅(#!) 라인은 주석으로 처리하지 않음
				if text != "" && (commentLine != 1 || !strings.HasPrefix(text, "!")) {
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

		case stDQ:
			if ch == '\n' {
				emitString(line)
				line++
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++
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

		case stTripleDQ:
			if ch == '\n' {
				line++
				buf.WriteRune(ch)
			} else if ch == '"' && peekN(i, 1) == '"' && peekN(i, 2) == '"' {
				emitString(line)
				state = stCode
				i += 2
			} else {
				buf.WriteRune(ch)
			}

		case stTripleSQ:
			if ch == '\n' {
				line++
				buf.WriteRune(ch)
			} else if ch == '\'' && peekN(i, 1) == '\'' && peekN(i, 2) == '\'' {
				emitString(line)
				state = stCode
				i += 2
			} else {
				buf.WriteRune(ch)
			}
		}
	}

	// 파일이 개행 없이 끝날 때 처리
	if state == stLine {
		text := strings.TrimSpace(buf.String())
		if text != "" && (commentLine != 1 || !strings.HasPrefix(text, "!")) {
			result = append(result, Comment{
				Text:    text,
				Line:    commentLine,
				EndLine: line,
				IsBlock: false,
				Kind:    KindComment,
			})
		}
	}
	if state == stDQ || state == stSQ || state == stTripleDQ || state == stTripleSQ {
		emitString(line)
	}

	markPEP723Blocks(result)
	return result, nil
}

// markPEP723Blocks 는 PEP 723 인라인 스크립트 메타데이터 블록을 감지하여 KindImport로 표시합니다.
// 패턴: # /// <type> (시작) ... # /// (종료), 시작/종료 줄 포함 블록 전체가 제외 대상입니다.
func markPEP723Blocks(comments []Comment) {
	inBlock := false
	for i := range comments {
		c := &comments[i]
		if c.Kind != KindComment || c.IsBlock {
			continue
		}
		if !inBlock {
			if isPEP723OpenTag(c.Text) {
				c.Kind = KindImport
				inBlock = true
			}
		} else {
			c.Kind = KindImport
			if c.Text == "///" {
				inBlock = false
			}
		}
	}
}

// isPEP723OpenTag 는 PEP 723 블록 시작 태그를 감지합니다.
// 텍스트는 `#` 제거 후 TrimSpace 된 값입니다.
// 예: "/// script", "/// tool.uv"
func isPEP723OpenTag(text string) bool {
	if !strings.HasPrefix(text, "/// ") {
		return false
	}
	typ := strings.TrimSpace(text[4:])
	if typ == "" {
		return false
	}
	for _, ch := range typ {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' && ch != '.' {
			return false
		}
	}
	return true
}
