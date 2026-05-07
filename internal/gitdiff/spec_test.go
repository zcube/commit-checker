package gitdiff

import (
	"testing"
)

// TestSpec_IsDefault: 빈 spec 만 default 로 인식.
func TestSpec_IsDefault(t *testing.T) {
	cases := []struct {
		s    Spec
		want bool
	}{
		{Spec{}, true},
		{Spec{From: "HEAD"}, false},
		{Spec{To: "worktree"}, false},
		{Spec{From: "A", To: "B"}, false},
	}
	for _, c := range cases {
		if got := c.s.IsDefault(); got != c.want {
			t.Errorf("IsDefault(%+v) = %v, want %v", c.s, got, c.want)
		}
	}
}

// TestSpec_IsWorktree: worktree alias 들 인식.
func TestSpec_IsWorktree(t *testing.T) {
	cases := []struct {
		to   string
		want bool
	}{
		{"worktree", true},
		{"working-tree", true},
		{"wt", true},
		{"", false},
		{"HEAD", false},
		{"main", false},
	}
	for _, c := range cases {
		s := Spec{To: c.to}
		if got := s.IsWorktree(); got != c.want {
			t.Errorf("IsWorktree(to=%q) = %v, want %v", c.to, got, c.want)
		}
	}
}

// TestBuildDiffArgs: 기본은 --staged.
func TestBuildDiffArgs(t *testing.T) {
	cases := []struct {
		name string
		s    Spec
		want []string
	}{
		{"default-staged", Spec{}, []string{"diff", "--staged"}},
		{"head-vs-worktree", Spec{From: "HEAD", To: "worktree"}, []string{"diff", "HEAD"}},
		{"index-vs-worktree", Spec{To: "worktree"}, []string{"diff"}},
		{"from-only", Spec{From: "origin/main"}, []string{"diff", "origin/main", "HEAD"}},
		{"explicit-range", Spec{From: "A", To: "B"}, []string{"diff", "A", "B"}},
	}
	for _, c := range cases {
		got := buildDiffArgs(c.s)
		if !equalStrings(got, c.want) {
			t.Errorf("[%s] buildDiffArgs(%+v) = %v, want %v", c.name, c.s, got, c.want)
		}
	}
}

// TestParseRange: ".." 와 "..." 양쪽 처리.
func TestParseRange(t *testing.T) {
	// 일반 range
	from, to, ok := ParseRange("origin/main..HEAD")
	if !ok || from != "origin/main" || to != "HEAD" {
		t.Errorf("ParseRange(\"origin/main..HEAD\") = %q, %q, %v", from, to, ok)
	}
	// range 가 아닌 경우
	_, _, ok = ParseRange("HEAD")
	if ok {
		t.Error("ParseRange(\"HEAD\") should not be a range")
	}
	_, _, ok = ParseRange("")
	if ok {
		t.Error("ParseRange(\"\") should not be a range")
	}
}

// TestSpecFromArgs: 위치 인자 → Spec 변환.
func TestSpecFromArgs(t *testing.T) {
	// 0 args → default
	s, err := SpecFromArgs(nil, false)
	if err != nil || !s.IsDefault() {
		t.Errorf("0 args: got %+v err %v", s, err)
	}

	// 1 arg single ref → ref vs worktree
	s, err = SpecFromArgs([]string{"HEAD"}, false)
	if err != nil || s.From != "HEAD" || s.To != RefWorktree {
		t.Errorf("1 ref: got %+v err %v", s, err)
	}

	// 1 arg range → split
	s, err = SpecFromArgs([]string{"main..feature"}, false)
	if err != nil || s.From != "main" || s.To != "feature" {
		t.Errorf("range: got %+v err %v", s, err)
	}

	// 2 args → A vs B
	s, err = SpecFromArgs([]string{"A", "B"}, false)
	if err != nil || s.From != "A" || s.To != "B" {
		t.Errorf("2 args: got %+v err %v", s, err)
	}

	// --staged + 0 args → default (HEAD↔index)
	s, err = SpecFromArgs(nil, true)
	if err != nil || !s.IsDefault() {
		t.Errorf("--staged 0: got %+v err %v", s, err)
	}

	// --staged + 1 arg → ref vs index
	s, err = SpecFromArgs([]string{"origin/main"}, true)
	if err != nil || s.From != "origin/main" || s.To != "" {
		t.Errorf("--staged ref: got %+v err %v", s, err)
	}

	// --staged + 2 args → error
	_, err = SpecFromArgs([]string{"A", "B"}, true)
	if err == nil {
		t.Error("--staged with 2 args should error")
	}

	// 3+ args → error
	_, err = SpecFromArgs([]string{"A", "B", "C"}, false)
	if err == nil {
		t.Error("3 args should error")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
