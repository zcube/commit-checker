package comment

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// HCLParser 는 HCL(HashiCorp Configuration Language — Terraform 등)에서
// 주석과 문자열 리터럴을 추출하는 파서.
//
// hashicorp/hcl v2 hclsyntax 토크나이저 기반: hclsyntax.LexConfig 로 토큰화한 뒤
// 주석/문자열 토큰만 골라내고, 문자열 본문은 토큰 Range 의 바이트 오프셋으로
// 원본 소스에서 그대로 잘라낸다 (인터폴레이션 포함 원문 유지).
//
// 추출 규칙:
//   - 줄 주석(# / //)과 블록 주석(/* */)은 TokenComment 하나로 들어오며,
//     마커를 제거하고 정리한 텍스트를 KindComment 로 추출
//   - 따옴표 문자열은 TokenOQuote ~ TokenCQuote 사이의 원문을 KindString 하나로.
//     인터폴레이션(${ }) 내부의 중첩 따옴표는 깊이를 추적해 외부 문자열이
//     조기 종료되지 않도록 함
//   - heredoc 은 TokenOHeredoc ~ TokenCHeredoc 사이의 본문을 KindString 하나로.
//     Line 은 <<LABEL 이 있는 줄
//   - LexConfig 의 diagnostics 는 무시 — 기존 파서처럼 잘못된 입력에도
//     best-effort 로 동작하며 에러를 반환하지 않음
type HCLParser struct{}

func (p *HCLParser) SupportedExtensions() []string {
	return []string{".hcl", ".tf", ".tfvars"}
}

// unescapeTemplateLiteral 은 따옴표 문자열 안의 템플릿 리터럴 이스케이프
// ($${ → ${, %%{ → %{)를 풀어 기존 상태 기계 파서가 만들던 텍스트와 맞춘다.
func unescapeTemplateLiteral(s string) string {
	s = strings.ReplaceAll(s, "$${", "${")
	s = strings.ReplaceAll(s, "%%{", "%{")
	return s
}

// heredocLabelOf 는 TokenOHeredoc 바이트("<<LABEL\n" 또는 "<<-LABEL\n")에서
// 라벨을 추출한다.
func heredocLabelOf(b []byte) string {
	s := strings.TrimPrefix(string(b), "<<")
	s = strings.TrimPrefix(s, "-")
	return strings.TrimSpace(s)
}

func (p *HCLParser) ParseFile(content string) ([]Comment, error) {
	// diagnostics 는 의도적으로 무시 (best-effort 파싱)
	tokens, _ := hclsyntax.LexConfig([]byte(content), "", hcl.Pos{Line: 1, Column: 1})

	var result []Comment

	var (
		// 따옴표 문자열 추적: 인터폴레이션 안의 중첩 따옴표 때문에
		// OQuote/CQuote 가 중첩될 수 있어 깊이를 센다.
		quoteDepth int
		quoteStart hcl.Pos // 최상위 OQuote 의 끝 = 문자열 본문 시작
		quoteLine  int     // 문자열 시작 줄

		// heredoc 추적: 인터폴레이션 안에 또 다른 heredoc 이 올 수 있어 깊이를 센다.
		heredocDepth int
		heredocStart hcl.Pos // 최상위 OHeredoc 의 끝 = 본문 시작 (라벨 줄 개행 다음)
		heredocLine  int     // <<LABEL 이 있는 줄
		heredocLabel string  // EOF 까지 닫히지 않은 heredoc 처리용

		endLine = 1 // 마지막 토큰 줄 — EOF 까지 닫히지 않은 항목의 EndLine 용
	)

	emitString := func(text string, line, endLine int) {
		if text == "" {
			return
		}
		result = append(result, Comment{
			Text:    text,
			Line:    line,
			EndLine: endLine,
			IsBlock: false,
			Kind:    KindString,
		})
	}

	for _, tok := range tokens {
		if l := tok.Range.End.Line; l > endLine {
			endLine = l
		}

		switch tok.Type {
		case hclsyntax.TokenComment:
			// 문자열/heredoc 내부(인터폴레이션 표현식 등)의 주석은
			// 문자열 원문에 포함되므로 별도 주석으로 추출하지 않음
			if quoteDepth > 0 || heredocDepth > 0 {
				continue
			}
			raw := string(tok.Bytes)
			if strings.HasPrefix(raw, "/*") {
				// 블록 주석 — 마커 제거 후 각 줄 선행 별표 정리
				inner := strings.TrimSuffix(strings.TrimPrefix(raw, "/*"), "*/")
				result = append(result, Comment{
					Text:    cleanBlockComment(inner),
					Line:    tok.Range.Start.Line,
					EndLine: tok.Range.End.Line,
					IsBlock: true,
					Kind:    KindComment,
				})
				continue
			}
			// 줄 주석 — # 또는 // 마커 제거.
			// 토큰 바이트가 종결 개행까지 포함하므로 EndLine 은 시작 줄로 고정.
			var text string
			if strings.HasPrefix(raw, "//") {
				text = raw[2:]
			} else {
				text = strings.TrimPrefix(raw, "#")
			}
			if text = strings.TrimSpace(text); text != "" {
				result = append(result, Comment{
					Text:    text,
					Line:    tok.Range.Start.Line,
					EndLine: tok.Range.Start.Line,
					IsBlock: false,
					Kind:    KindComment,
				})
			}

		case hclsyntax.TokenOQuote:
			if heredocDepth > 0 {
				continue // heredoc 본문 원문에 포함됨
			}
			if quoteDepth == 0 {
				quoteStart = tok.Range.End
				quoteLine = tok.Range.Start.Line
			}
			quoteDepth++

		case hclsyntax.TokenCQuote:
			if heredocDepth > 0 || quoteDepth == 0 {
				continue
			}
			quoteDepth--
			if quoteDepth == 0 {
				text := unescapeTemplateLiteral(content[quoteStart.Byte:tok.Range.Start.Byte])
				emitString(text, quoteLine, tok.Range.Start.Line)
			}

		case hclsyntax.TokenQuotedNewline:
			// 따옴표 문자열 안의 개행은 비정상 입력 — 기존 파서처럼
			// 그 지점에서 문자열을 관대하게 종료 처리
			if quoteDepth == 1 && heredocDepth == 0 {
				text := unescapeTemplateLiteral(content[quoteStart.Byte:tok.Range.Start.Byte])
				emitString(text, quoteLine, tok.Range.Start.Line)
				quoteDepth = 0
			}

		case hclsyntax.TokenOHeredoc:
			if quoteDepth > 0 {
				continue
			}
			if heredocDepth == 0 {
				heredocStart = tok.Range.End
				heredocLine = tok.Range.Start.Line
				heredocLabel = heredocLabelOf(tok.Bytes)
			}
			heredocDepth++

		case hclsyntax.TokenCHeredoc:
			if quoteDepth > 0 || heredocDepth == 0 {
				continue
			}
			heredocDepth--
			if heredocDepth == 0 {
				// CHeredoc 토큰은 닫는 라벨 줄(선행 공백 포함)에서 시작하므로
				// 본문은 그 직전 개행까지 — 마지막 개행만 제거
				body := strings.TrimSuffix(content[heredocStart.Byte:tok.Range.Start.Byte], "\n")
				emitString(body, heredocLine, tok.Range.Start.Line)
			}
		}
	}

	// 파일 끝까지 닫히지 않은 문자열/heredoc 도 기존 파서처럼 best-effort 로 추출
	switch {
	case quoteDepth > 0:
		emitString(unescapeTemplateLiteral(content[quoteStart.Byte:]), quoteLine, endLine)
	case heredocDepth > 0:
		body := content[heredocStart.Byte:]
		// 개행 없이 끝나는 마지막 줄이 닫는 라벨이면 본문에서 제외
		if idx := strings.LastIndexByte(body, '\n'); idx >= 0 &&
			strings.TrimSpace(body[idx+1:]) == heredocLabel {
			body = body[:idx]
		} else {
			body = strings.TrimSuffix(body, "\n")
		}
		emitString(body, heredocLine, endLine)
	}

	return result, nil
}
