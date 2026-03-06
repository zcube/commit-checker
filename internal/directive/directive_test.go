package directive_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/comment"
	"github.com/zcube/commit-checker/internal/directive"
)

func comments(texts ...string) []comment.Comment {
	cs := make([]comment.Comment, len(texts))
	for i, t := range texts {
		cs[i] = comment.Comment{Text: t, Line: i + 1, EndLine: i + 1}
	}
	return cs
}

func TestAnalyze_NoDirectives_AllChecked(t *testing.T) {
	cs := comments("한국어 주석", "또 다른 주석")
	states := directive.Analyze(cs, "korean")
	for i, s := range states {
		if s.Skip {
			t.Errorf("[%d] unexpected Skip=true", i)
		}
		if s.Language != "korean" {
			t.Errorf("[%d] expected language=korean, got %q", i, s.Language)
		}
	}
}

// commit-checker:ignore skips the immediately following comment.
func TestAnalyze_Ignore_SkipsNextComment(t *testing.T) {
	cs := comments(
		"commit-checker:ignore",
		"This English comment should be skipped",
		"한국어 주석은 체크됨",
	)
	states := directive.Analyze(cs, "korean")

	if !states[0].Skip {
		t.Error("[0] directive itself should be skipped")
	}
	if !states[1].Skip {
		t.Error("[1] comment after :ignore should be skipped")
	}
	if states[2].Skip {
		t.Error("[2] second comment should NOT be skipped")
	}
	if states[2].Language != "korean" {
		t.Errorf("[2] expected language=korean, got %q", states[2].Language)
	}
}

// commit-checker:ignore only skips the immediately next comment; the one after resumes.
func TestAnalyze_Ignore_OnlyOneComment(t *testing.T) {
	cs := comments(
		"commit-checker:ignore",
		"skipped",
		"also checked",
		"also checked",
	)
	states := directive.Analyze(cs, "korean")
	if !states[1].Skip {
		t.Error("[1] should be skipped")
	}
	if states[2].Skip {
		t.Error("[2] should not be skipped")
	}
	if states[3].Skip {
		t.Error("[3] should not be skipped")
	}
}

// commit-checker:disable / :enable region.
func TestAnalyze_Disable_Enable(t *testing.T) {
	cs := comments(
		"한국어",                   // 0 checked
		"commit-checker:disable", // 1 directive
		"English comment",        // 2 skipped (disabled)
		"another English",        // 3 skipped (disabled)
		"commit-checker:enable",  // 4 directive
		"한국어 재개",               // 5 checked again
	)
	states := directive.Analyze(cs, "korean")

	if states[0].Skip || states[0].Language != "korean" {
		t.Error("[0] should be checked with korean")
	}
	if !states[1].Skip {
		t.Error("[1] directive should be skipped")
	}
	if !states[2].Skip {
		t.Error("[2] should be skipped (disabled)")
	}
	if !states[3].Skip {
		t.Error("[3] should be skipped (disabled)")
	}
	if !states[4].Skip {
		t.Error("[4] directive should be skipped")
	}
	if states[5].Skip || states[5].Language != "korean" {
		t.Errorf("[5] should be checked with korean after enable, got skip=%v lang=%q", states[5].Skip, states[5].Language)
	}
}

// commit-checker:disable:lang=english allows English comments inside the region.
func TestAnalyze_Disable_WithLang(t *testing.T) {
	cs := comments(
		"commit-checker:disable:lang=english",
		"This English comment is allowed",
		"Another English comment",
		"commit-checker:enable",
		"한국어 재개",
	)
	states := directive.Analyze(cs, "korean")

	if !states[0].Skip {
		t.Error("[0] directive should be skipped")
	}
	if states[1].Skip {
		t.Error("[1] should NOT be skipped (lang override)")
	}
	if states[1].Language != "english" {
		t.Errorf("[1] expected language=english, got %q", states[1].Language)
	}
	if states[2].Language != "english" {
		t.Errorf("[2] expected language=english, got %q", states[2].Language)
	}
	if states[4].Language != "korean" {
		t.Errorf("[4] expected language=korean after enable, got %q", states[4].Language)
	}
}

// commit-checker:lang= switches language from that point forward.
func TestAnalyze_Lang_Switch(t *testing.T) {
	cs := comments(
		"한국어 주석",
		"commit-checker:lang=english",
		"This English is now required",
		"Another English comment",
	)
	states := directive.Analyze(cs, "korean")

	if states[0].Language != "korean" {
		t.Errorf("[0] expected korean, got %q", states[0].Language)
	}
	if !states[1].Skip {
		t.Error("[1] directive should be skipped")
	}
	if states[2].Language != "english" {
		t.Errorf("[2] expected english, got %q", states[2].Language)
	}
	if states[3].Language != "english" {
		t.Errorf("[3] expected english, got %q", states[3].Language)
	}
}

// commit-checker:lang= accepts locale codes.
func TestAnalyze_Lang_LocaleCode(t *testing.T) {
	cs := comments(
		"commit-checker:lang=ja",
		"これは日本語のコメントです",
	)
	states := directive.Analyze(cs, "korean")

	if states[1].Language != "japanese" {
		t.Errorf("expected japanese (from locale ja), got %q", states[1].Language)
	}
}

// commit-checker:file-lang= overrides language for the entire file.
func TestAnalyze_FileLang(t *testing.T) {
	cs := comments(
		"commit-checker:file-lang=english",
		"This English comment should pass",
		"Another English line",
	)
	states := directive.Analyze(cs, "korean")

	if !states[0].Skip {
		t.Error("[0] directive should be skipped")
	}
	for i := 1; i < len(states); i++ {
		if states[i].Skip {
			t.Errorf("[%d] should not be skipped", i)
		}
		if states[i].Language != "english" {
			t.Errorf("[%d] expected english, got %q", i, states[i].Language)
		}
	}
}

// commit-checker:file-lang= also accepts locale codes.
func TestAnalyze_FileLang_LocaleCode(t *testing.T) {
	cs := comments(
		"commit-checker:file-lang=zh",
		"这是一个中文注释内容示例",
	)
	states := directive.Analyze(cs, "korean")
	if states[1].Language != "chinese" {
		t.Errorf("expected chinese, got %q", states[1].Language)
	}
}

// file-lang + disable region: disable inside file-lang scope still works.
func TestAnalyze_FileLang_WithDisable(t *testing.T) {
	cs := comments(
		"commit-checker:file-lang=english",
		"English comment",
		"commit-checker:disable",
		"skipped comment",
		"commit-checker:enable",
		"English resumed",
	)
	states := directive.Analyze(cs, "korean")

	if states[1].Language != "english" {
		t.Errorf("[1] expected english, got %q", states[1].Language)
	}
	if !states[3].Skip {
		t.Error("[3] should be skipped (disabled)")
	}
	if states[5].Language != "english" {
		t.Errorf("[5] expected english after enable, got %q", states[5].Language)
	}
}

// Directive matching is case-insensitive.
func TestAnalyze_CaseInsensitive(t *testing.T) {
	cs := comments(
		"Commit-Checker:Ignore",
		"This should be skipped",
		"COMMIT-CHECKER:DISABLE",
		"Also skipped",
		"commit-checker:enable",
		"Checked",
	)
	states := directive.Analyze(cs, "korean")
	if !states[1].Skip {
		t.Error("[1] :Ignore should work case-insensitively")
	}
	if !states[3].Skip {
		t.Error("[3] :DISABLE should work case-insensitively")
	}
	if states[5].Skip {
		t.Error("[5] should be checked after enable")
	}
}

// IsDirective helper.
func TestIsDirective(t *testing.T) {
	yes := []string{
		"commit-checker:ignore",
		"commit-checker:disable",
		"commit-checker:enable",
		"commit-checker:lang=english",
		"commit-checker:file-lang=ko",
		"COMMIT-CHECKER:DISABLE",
		"  commit-checker:ignore  ",
	}
	no := []string{
		"한국어 주석입니다",
		"This is a comment",
		"// nolint",
		"commit",
		"checker:ignore",
	}
	for _, s := range yes {
		if !directive.IsDirective(s) {
			t.Errorf("IsDirective(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if directive.IsDirective(s) {
			t.Errorf("IsDirective(%q) = true, want false", s)
		}
	}
}
