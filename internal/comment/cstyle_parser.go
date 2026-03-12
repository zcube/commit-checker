package comment

import "strings"

// CStyleParser 는 C 스타일 언어(TypeScript, JavaScript, Java, Kotlin, C, C++, C#, Swift, Rust 등)에서
// 주석과 문자열 리터럴을 추출하는 상태 기계 기반 파서.
type CStyleParser struct {
	extensions  []string
	hasTemplate bool // JS/TS 는 백틱 템플릿 리터럴을 지원합니다.
}

// NewCStyleParser 는 주어진 확장자를 처리하는 파서를 생성합니다.
// JS/TS 의 백틱 템플릿 리터럴 처리를 위해 hasTemplate=true 로 설정하세요.
func NewCStyleParser(extensions []string, hasTemplate bool) *CStyleParser {
	return &CStyleParser{extensions: extensions, hasTemplate: hasTemplate}
}

func (p *CStyleParser) SupportedExtensions() []string {
	return p.extensions
}

// kindForImport 는 import 컨텍스트 여부에 따라 KindImport 또는 KindString을 반환.
func kindForImport(isImport bool) Kind {
	if isImport {
		return KindImport
	}
	return KindString
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

	var (
		result      []Comment
		runes       = []rune(content)
		n           = len(runes)
		state       = stCode
		buf         strings.Builder
		linePre     strings.Builder // stCode 상태에서 현재 줄의 앞쪽 내용 추적
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

	// isImportContext 는 현재 줄 앞부분이 import/include 경로 컨텍스트인지 판단합니다.
	isImportContext := func() bool {
		pre := strings.TrimSpace(linePre.String())
		// C/C++: #include "file.h" 패턴
		if strings.HasPrefix(pre, "#include") {
			return true
		}
		// JS/TS: import ... from "module" 또는 import ... from 'module'
		if pre == "from" || strings.HasSuffix(pre, " from") {
			return true
		}
		// ESM 단독 import: import "polyfill" 또는 import 'polyfill'
		if pre == "import" {
			return true
		}
		return false
	}

	emitString := func(endLine int, kind Kind) {
		val := buf.String()
		if val != "" {
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
				linePre.Reset()
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
			default:
				linePre.WriteRune(ch)
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
				linePre.Reset()
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
				emitString(line, kindForImport(isImportContext()))
				line++
				linePre.Reset()
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++ // 이스케이프 시퀀스는 언어 감지에 불필요하므로 건너뜀
			} else if ch == '"' {
				emitString(line, kindForImport(isImportContext()))
				state = stCode
			} else {
				buf.WriteRune(ch)
			}

		case stSQ:
			if ch == '\n' {
				emitString(line, kindForImport(isImportContext()))
				line++
				linePre.Reset()
				state = stCode
			} else if ch == '\\' && i+1 < n {
				i++
			} else if ch == '\'' {
				emitString(line, kindForImport(isImportContext()))
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
				emitString(line, KindString)
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
		emitString(line, kindForImport(isImportContext()))
	}

	return result, nil
}
