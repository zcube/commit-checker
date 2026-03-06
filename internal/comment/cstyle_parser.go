package comment

import "strings"

// CStyleParser 는 C 스타일 언어(TypeScript, JavaScript, Java, Kotlin, C, C++, C#, Swift, Rust 등)에서
// 주석과 문자열 리터럴을 추출하는 상태 기계 기반 파서.
type CStyleParser struct {
	extensions  []string
	hasTemplate bool // JS/TS에서 템플릿 리터럴(백틱) 사용
}

// NewCStyleParser: 주어진 확장자에 대한 파서를 생성.
// JS/TS의 백틱 템플릿 리터럴을 처리하려면 hasTemplate=true로 설정.
func NewCStyleParser(extensions []string, hasTemplate bool) *CStyleParser {
	return &CStyleParser{extensions: extensions, hasTemplate: hasTemplate}
}

func (p *CStyleParser) SupportedExtensions() []string {
	return p.extensions
}

// findImportLines 는 #include, import, require, export...from 등 import 구문이 포함된 줄 번호를 반환.
func findImportLines(content string) map[int]bool {
	lines := strings.Split(content, "\n")
	importLines := make(map[int]bool)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1
		// C/C++ 전처리기 포함
		if strings.HasPrefix(trimmed, "#include") {
			importLines[lineNum] = true
			continue
		}
		// JS/TS/Java/Kotlin 임포트 구문
		if strings.HasPrefix(trimmed, "import") {
			importLines[lineNum] = true
			continue
		}
		// JS 모듈 요청 구문
		if strings.Contains(trimmed, "require(") {
			importLines[lineNum] = true
			continue
		}
		// JS/TS 재내보내기 구문
		if strings.HasPrefix(trimmed, "export") && strings.Contains(trimmed, " from ") {
			importLines[lineNum] = true
			continue
		}
		// Rust 파일 포함 매크로
		if strings.Contains(trimmed, "include!(") ||
			strings.Contains(trimmed, "include_str!(") ||
			strings.Contains(trimmed, "include_bytes!(") {
			importLines[lineNum] = true
			continue
		}
		// C# using (일반적으로 문자열 리터럴 없음)
		// Swift import (일반적으로 문자열 리터럴 없음)
	}
	return importLines
}

func (p *CStyleParser) ParseFile(content string) ([]Comment, error) {
	const (
		stCode     = iota
		stLine     // // 주석 내부
		stBlock    // /* 주석 내부
		stDQ       // "..." 문자열 내부
		stSQ       // '...' 문자열 내부 (문자 리터럴 포함)
		stTemplate // `...` 템플릿 리터럴 내부
	)

	importLines := findImportLines(content)

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
			kind := KindString
			if importLines[strLine] {
				kind = KindImportString
			}
			result = append(result, Comment{
				Text:    val,
				Line:    strLine,
				EndLine: endLine,
				IsBlock: false,
				Kind:    kind,
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
				i++ // 두 번째 '/' 소비
			case ch == '/' && peek(i) == '*':
				state = stBlock
				commentLine = line
				i++ // '*' 소비
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
				i++ // '/' 소비
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
