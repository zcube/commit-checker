package checker

import (
	"strings"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/directive"
)

// commentUnit 은 언어 검사의 기본 단위입니다.
// 연속된 줄 주석은 하나의 단위로 묶여 합쳐진 텍스트로 검사됩니다.
type commentUnit struct {
	text    string       // 검사할 텍스트 (연속 줄 주석은 "\n"으로 연결)
	line    int          // 첫 번째 줄 (오류 보고에 사용)
	endLine int          // 마지막 줄
	lang    string       // 적용 언어
	kind    comment.Kind // 주석 종류
}

// buildCommentUnits 는 주석 목록과 지시자 상태를 바탕으로 언어 검사 단위를 생성합니다.
//
// 규칙:
//   - 블록 주석(/* */), 문자열 리터럴: 개별 단위
//   - 줄 주석(//): 인접한 줄(EndLine+1 == 다음 Line)에 같은 언어의 주석이 이어지면 하나의 단위로 묶음
//   - Skip=true 인 지시자 주석, KindImport: 제외
//   - checkStrings=false 이면 KindString 제외
func buildCommentUnits(
	comments []comment.Comment,
	states []directive.CommentState,
	checkStrings bool,
) []commentUnit {
	n := len(comments)
	var units []commentUnit
	i := 0
	for i < n {
		c := comments[i]
		s := states[i]

		if s.Skip || c.Kind == comment.KindImport {
			i++
			continue
		}
		if c.Kind == comment.KindString && !checkStrings {
			i++
			continue
		}

		// 블록 주석 또는 문자열 리터럴: 개별 단위
		if c.IsBlock || c.Kind == comment.KindString {
			units = append(units, commentUnit{
				text:    strings.TrimSpace(c.Text),
				line:    c.Line,
				endLine: c.EndLine,
				lang:    s.Language,
				kind:    c.Kind,
			})
			i++
			continue
		}

		// 줄 주석: 연속된 것들을 하나의 단위로 묶습니다.
		startLine := c.Line
		prevEndLine := c.EndLine
		lang := s.Language
		var texts []string

		for i < n {
			ci := comments[i]
			si := states[i]

			// 묶음을 끊는 조건
			if si.Skip || ci.Kind == comment.KindImport {
				break
			}
			if ci.IsBlock || ci.Kind == comment.KindString {
				break
			}
			if si.Language != lang {
				break
			}
			// 줄 연속성 확인: 이전 주석 바로 다음 줄이어야 합니다.
			if len(texts) > 0 && ci.Line > prevEndLine+1 {
				break
			}

			t := strings.TrimSpace(ci.Text)
			if t != "" {
				texts = append(texts, t)
			}
			prevEndLine = ci.EndLine
			i++
		}

		if len(texts) > 0 {
			units = append(units, commentUnit{
				text:    strings.Join(texts, "\n"),
				line:    startLine,
				endLine: prevEndLine,
				lang:    lang,
				kind:    comment.KindComment,
			})
		}
	}
	return units
}
