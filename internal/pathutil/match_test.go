package pathutil_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/pathutil"
)

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			path:     "main.go",
			patterns: []string{"main.go"},
			want:     true,
		},
		{
			name:     "wildcard extension",
			path:     "foo.pb.go",
			patterns: []string{"*.pb.go"},
			want:     true,
		},
		{
			name:     "vendor double-star",
			path:     "vendor/github.com/pkg/foo.go",
			patterns: []string{"vendor/**"},
			want:     true,
		},
		{
			name:     "nested double-star",
			path:     "internal/generated/foo.go",
			patterns: []string{"**/generated/**"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "internal/checker/diff.go",
			patterns: []string{"vendor/**", "*.pb.go"},
			want:     false,
		},
		{
			name:     "empty patterns",
			path:     "any/path.go",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "base name match",
			path:     "deep/dir/generated.go",
			patterns: []string{"generated.go"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.MatchesAny(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("MatchesAny(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}
