package comment

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// GoParser 는 공식 go/parser AST를 사용하여 Go 소스 코드에서 주석과 문자열 리터럴을 추출.
type GoParser struct{}

func (p *GoParser) SupportedExtensions() []string {
	return []string{".go"}
}

func (p *GoParser) ParseFile(content string) ([]Comment, error) {
	fset := token.NewFileSet()
	// ParseComments: 주석 노드 포함; AllErrors: 구문 오류 무시하고 계속 파싱.
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments|parser.AllErrors)
	if err != nil && f == nil {
		return nil, err
	}

	var result []Comment

	// 주석 추출
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			pos := fset.Position(c.Pos())
			endPos := fset.Position(c.End())
			text := c.Text
			isBlock := strings.HasPrefix(text, "/*")

			if isBlock {
				text = strings.TrimPrefix(text, "/*")
				text = strings.TrimSuffix(text, "*/")
				text = cleanBlockComment(text)
			} else {
				text = strings.TrimPrefix(text, "//")
				text = strings.TrimSpace(text)
			}

			result = append(result, Comment{
				Text:    text,
				Line:    pos.Line,
				EndLine: endPos.Line,
				IsBlock: isBlock,
				Kind:    KindComment,
			})
		}
	}

	// import 경로 위치 수집 (언어 검사 제외 대상)
	importPositions := make(map[token.Pos]bool)
	for _, imp := range f.Imports {
		importPositions[imp.Path.Pos()] = true
	}

	// 문자열 리터럴 추출 (AST 순회)
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		pos := fset.Position(lit.Pos())
		endPos := fset.Position(lit.End())

		val := unquoteGoString(lit.Value)
		if val == "" {
			return true
		}

		kind := KindString
		if importPositions[lit.Pos()] {
			kind = KindImportString
		}

		result = append(result, Comment{
			Text:    val,
			Line:    pos.Line,
			EndLine: endPos.Line,
			IsBlock: false,
			Kind:    kind,
		})
		return true
	})

	// 주석과 문자열을 라인 순으로 정렬 (directive 분석 정확도)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Line < result[j].Line
	})

	return result, nil
}

// unquoteGoString 은 Go 문자열 리터럴에서 실제 값을 추출.
// 해석 문자열("..."): strconv.Unquote 사용.
// 원시 문자열(``...``): 백틱 제거.
func unquoteGoString(raw string) string {
	if strings.HasPrefix(raw, "`") {
		return raw[1 : len(raw)-1]
	}
	if v, err := strconv.Unquote(raw); err == nil {
		return v
	}
	// 파싱 실패 시 따옴표만 제거
	if len(raw) >= 2 {
		return raw[1 : len(raw)-1]
	}
	return ""
}
