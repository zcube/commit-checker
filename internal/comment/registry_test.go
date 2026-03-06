package comment_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
)

func TestExtensionsForLanguages(t *testing.T) {
	tests := []struct {
		langs []string
		want  []string // must all be present (order-independent)
	}{
		{
			langs: []string{"go"},
			want:  []string{".go"},
		},
		{
			langs: []string{"typescript"},
			want:  []string{".ts", ".tsx"},
		},
		{
			langs: []string{"go", "python"},
			want:  []string{".go", ".py"},
		},
		{
			langs: []string{"unknown_lang"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		got := comment.ExtensionsForLanguages(tt.langs)
		for _, want := range tt.want {
			found := false
			for _, g := range got {
				if g == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ExtensionsForLanguages(%v): missing %q in result %v", tt.langs, want, got)
			}
		}
	}
}

func TestGetParser_ByLanguageExtension(t *testing.T) {
	exts := comment.ExtensionsForLanguages([]string{"go", "typescript", "python"})
	for _, ext := range exts {
		p := comment.GetParser("test" + ext)
		if p == nil {
			t.Errorf("GetParser(test%s): expected parser, got nil", ext)
		}
	}
}
