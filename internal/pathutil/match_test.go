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

// TestMatchPath: 절대 경로 매칭용 MatchPath 검증.
// MatchesAny 와 달리 base name 단독 매칭이 없어야 한다.
func TestMatchPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{
			name:    "디렉터리 하위 전체 매칭",
			path:    "/home/user/work/repo",
			pattern: "/home/user/work/**",
			want:    true,
		},
		{
			name:    "디렉터리 자체도 매칭 (** 는 0개 세그먼트 허용)",
			path:    "/home/user/work",
			pattern: "/home/user/work/**",
			want:    true,
		},
		{
			name:    "깊은 하위 디렉터리 매칭",
			path:    "/home/user/work/team/sub/repo",
			pattern: "/home/user/work/**",
			want:    true,
		},
		{
			name:    "다른 디렉터리는 비매칭",
			path:    "/home/user/personal/repo",
			pattern: "/home/user/work/**",
			want:    false,
		},
		{
			name:    "base name 단독 매칭은 하지 않음",
			path:    "/home/user/deep/repo",
			pattern: "repo",
			want:    false,
		},
		{
			name:    "정확한 경로 매칭",
			path:    "/home/user/work",
			pattern: "/home/user/work",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathutil.MatchPath(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("MatchPath(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}
