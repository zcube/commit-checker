package comment

import "strings"

// HCLParser 는 HCL(HashiCorp Configuration Language — Terraform 등)에서
// 주석과 문자열 리터럴을 추출하는 상태 기계 기반 파서.
//
// 지원 문법 (hclsyntax 스펙 기준):
//   - 줄 주석: # 과 // 둘 다
//   - 블록 주석: /* */ (중첩 없음)
//   - 문자열: "..." (백슬래시 이스케이프 지원)
//   - 인터폴레이션: 문자열 안의 ${ ... } 와 %{ ... } — 내부의 중첩 따옴표 문자열과
//     중첩 중괄호를 추적해 외부 문자열이 조기 종료되지 않도록 함.
//     $${ 와 %%{ 는 리터럴 이스케이프로 인터폴레이션을 시작하지 않음.
//   - heredoc: <<LABEL / <<-LABEL 부터 LABEL 만 있는 줄까지 본문 전체를 KindString 하나로.
type HCLParser struct{}

func (p *HCLParser) SupportedExtensions() []string {
	return []string{".hcl", ".tf", ".tfvars"}
}

// interpFrame 은 문자열 내부 인터폴레이션 추적 스택의 프레임 종류.
type interpFrame uint8

const (
	frameBrace  interpFrame = iota // ${ ... } / %{ ... } 표현식 또는 중첩 { } 내부
	frameString                    // 인터폴레이션 표현식 안의 중첩 "..." 문자열 내부
)

// isHeredocLabelRune 은 heredoc 라벨에 쓸 수 있는 문자인지 판단합니다.
func isHeredocLabelRune(ch rune) bool {
	return ch == '_' || ch == '-' ||
		(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func (p *HCLParser) ParseFile(content string) ([]Comment, error) {
	const (
		stCode    = iota
		stLine    // # 또는 // 주석 내부
		stBlock   // /* 주석 내부
		stString  // "..." 문자열 내부 (인터폴레이션 포함)
		stHeredoc // heredoc 본문 내부
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

		// 문자열 인터폴레이션 추적 스택: 비어 있으면 일반 문자열 본문.
		interpStack []interpFrame

		// heredoc 상태: pendingLabel 은 <<LABEL 발견 후 줄 끝까지 대기 중인 라벨.
		heredocPendingLabel string
		heredocPendingTrim  bool // <<- 형식 여부 (닫는 라벨 앞 공백 허용)
		heredocLabel        string
		heredocTrim         bool
		heredocLineBuf      strings.Builder // heredoc 의 현재 줄 버퍼
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

	emitLineComment := func(endLine int) {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			result = append(result, Comment{
				Text:    text,
				Line:    commentLine,
				EndLine: endLine,
				IsBlock: false,
				Kind:    KindComment,
			})
		}
		buf.Reset()
	}

	// tryHeredocStart 는 i 위치의 '<' 에서 <<LABEL / <<-LABEL 을 인식합니다.
	// 인식되면 라벨을 대기 상태로 등록하고 라벨 끝 직전 인덱스를 반환합니다.
	tryHeredocStart := func(i int) (int, bool) {
		if peek(i) != '<' {
			return i, false
		}
		j := i + 2
		trim := false
		if j < n && runes[j] == '-' {
			trim = true
			j++
		}
		start := j
		for j < n && isHeredocLabelRune(runes[j]) {
			j++
		}
		if j == start {
			return i, false // 라벨 없음 — heredoc 아님
		}
		heredocPendingLabel = string(runes[start:j])
		heredocPendingTrim = trim
		return j - 1, true
	}

	// closesHeredoc 은 heredoc 의 한 줄이 닫는 라벨인지 판단합니다.
	closesHeredoc := func(lineStr string) bool {
		candidate := strings.TrimRight(lineStr, " \t\r")
		if heredocTrim {
			// <<- 형식은 닫는 라벨 앞 공백을 허용
			candidate = strings.TrimLeft(candidate, " \t")
		}
		return candidate == heredocLabel
	}

	for i := 0; i < n; i++ {
		ch := runes[i]

		switch state {
		case stCode:
			switch {
			case ch == '\n':
				line++
				// <<LABEL 발견 후 줄이 끝나면 heredoc 본문 시작
				if heredocPendingLabel != "" {
					heredocLabel = heredocPendingLabel
					heredocTrim = heredocPendingTrim
					heredocPendingLabel = ""
					heredocLineBuf.Reset()
					buf.Reset()
					state = stHeredoc
				}
			case ch == '#':
				state = stLine
				commentLine = line
				buf.Reset()
			case ch == '/' && peek(i) == '/':
				state = stLine
				commentLine = line
				buf.Reset()
				i++ // 두 번째 '/' 소비
			case ch == '/' && peek(i) == '*':
				state = stBlock
				commentLine = line
				buf.Reset()
				i++ // '*' 소비
			case ch == '"':
				state = stString
				strLine = line
				interpStack = interpStack[:0]
				buf.Reset()
			case ch == '<' && heredocPendingLabel == "":
				if next, ok := tryHeredocStart(i); ok {
					strLine = line // heredoc 시작 줄 = <<LABEL 이 있는 줄
					i = next
				}
			}

		case stLine:
			if ch == '\n' {
				emitLineComment(line)
				state = stCode
				line++
				// <<EOF 뒤에 줄 주석이 붙은 경우에도 heredoc 본문 시작
				if heredocPendingLabel != "" {
					heredocLabel = heredocPendingLabel
					heredocTrim = heredocPendingTrim
					heredocPendingLabel = ""
					heredocLineBuf.Reset()
					buf.Reset()
					state = stHeredoc
				}
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

		case stString:
			if len(interpStack) == 0 {
				// 일반 문자열 본문
				switch {
				case ch == '\n':
					// HCL 쿼트 문자열은 한 줄 — 비정상 입력은 관대하게 종료 처리
					emitString(line)
					line++
					state = stCode
				case ch == '\\' && i+1 < n:
					i++ // 이스케이프 시퀀스는 언어 감지에 불필요하므로 건너뜀
				case (ch == '$' || ch == '%') && peek(i) == ch && i+2 < n && runes[i+2] == '{':
					// $${ / %%{ 리터럴 이스케이프 — 인터폴레이션 아님
					buf.WriteRune(ch)
					buf.WriteRune('{')
					i += 2
				case (ch == '$' || ch == '%') && peek(i) == '{':
					// ${ ... } / %{ ... } 인터폴레이션 시작
					interpStack = append(interpStack, frameBrace)
					buf.WriteRune(ch)
					buf.WriteRune('{')
					i++
				case ch == '"':
					emitString(line)
					state = stCode
				default:
					buf.WriteRune(ch)
				}
				continue
			}

			// 인터폴레이션 내부
			top := interpStack[len(interpStack)-1]
			if top == frameBrace {
				// ${ ... } 표현식 내부 — 중첩 중괄호와 중첩 문자열 추적
				switch ch {
				case '\n':
					line++
					buf.WriteRune(ch)
				case '{':
					interpStack = append(interpStack, frameBrace)
					buf.WriteRune(ch)
				case '}':
					interpStack = interpStack[:len(interpStack)-1]
					buf.WriteRune(ch)
				case '"':
					interpStack = append(interpStack, frameString)
					buf.WriteRune(ch)
				default:
					buf.WriteRune(ch)
				}
			} else {
				// 인터폴레이션 표현식 안의 중첩 문자열 내부
				switch {
				case ch == '\n':
					line++
					buf.WriteRune(ch)
				case ch == '\\' && i+1 < n:
					i++
				case (ch == '$' || ch == '%') && peek(i) == ch && i+2 < n && runes[i+2] == '{':
					buf.WriteRune(ch)
					buf.WriteRune('{')
					i += 2
				case (ch == '$' || ch == '%') && peek(i) == '{':
					interpStack = append(interpStack, frameBrace)
					buf.WriteRune(ch)
					buf.WriteRune('{')
					i++
				case ch == '"':
					interpStack = interpStack[:len(interpStack)-1]
					buf.WriteRune(ch)
				default:
					buf.WriteRune(ch)
				}
			}

		case stHeredoc:
			if ch == '\n' {
				lineStr := heredocLineBuf.String()
				heredocLineBuf.Reset()
				if closesHeredoc(lineStr) {
					// 닫는 라벨 줄은 본문에 포함하지 않음
					emitString(line)
					state = stCode
				} else {
					if buf.Len() > 0 {
						buf.WriteRune('\n')
					}
					buf.WriteString(lineStr)
				}
				line++
			} else {
				heredocLineBuf.WriteRune(ch)
			}
		}
	}

	// 파일이 개행 없이 끝날 때 처리
	switch state {
	case stLine:
		emitLineComment(line)
	case stString:
		emitString(line)
	case stHeredoc:
		lineStr := heredocLineBuf.String()
		if !closesHeredoc(lineStr) && lineStr != "" {
			if buf.Len() > 0 {
				buf.WriteRune('\n')
			}
			buf.WriteString(lineStr)
		}
		emitString(line)
	}

	return result, nil
}
