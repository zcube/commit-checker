package comment

import (
	"go/parser"
	"go/token"
	"strings"
)

// GoParser extracts comments from Go source code using the official go/parser AST.
type GoParser struct{}

func (p *GoParser) SupportedExtensions() []string {
	return []string{".go"}
}

func (p *GoParser) ParseFile(content string) ([]Comment, error) {
	fset := token.NewFileSet()
	// ParseComments includes comment nodes; AllErrors continues past syntax errors.
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments|parser.AllErrors)
	if err != nil && f == nil {
		return nil, err
	}

	var comments []Comment
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

			comments = append(comments, Comment{
				Text:    text,
				Line:    pos.Line,
				EndLine: endPos.Line,
				IsBlock: isBlock,
			})
		}
	}
	return comments, nil
}
