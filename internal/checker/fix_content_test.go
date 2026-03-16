package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func boolPtr(b bool) *bool { return &b }

func TestFixFileContent_NoIssues(t *testing.T) {
	cfg := &config.Config{}
	result := checker.FixFileContent(cfg, "정상적인 UTF-8 내용입니다.")
	if result.NeedsFixing() {
		t.Errorf("expected no fixes needed, got changes: %v", result.Changes)
	}
}

func TestFixFileContent_BadRune(t *testing.T) {
	cfg := &config.Config{}
	// 잘못된 UTF-8 바이트 포함
	content := "good" + string([]byte{0xFF, 0xFE}) + "text"
	result := checker.FixFileContent(cfg, content)
	if !result.NeedsFixing() {
		t.Error("expected fix needed for bad rune")
	}
	if result.Fixed == content {
		t.Error("expected content to be changed")
	}
}

func TestFixFileContent_InvisibleChars_Enabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.NoInvisibleChars = boolPtr(true)
	// ZERO WIDTH SPACE (U+200B)
	content := "텍스트\u200B포함"
	result := checker.FixFileContent(cfg, content)
	if !result.NeedsFixing() {
		t.Error("expected fix needed for invisible char")
	}
}

func TestFixFileContent_InvisibleChars_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.NoInvisibleChars = boolPtr(false)
	content := "텍스트\u200B포함"
	result := checker.FixFileContent(cfg, content)
	if result.NeedsFixing() {
		t.Error("expected no fix when invisible chars disabled")
	}
}

func TestFixFileContent_AmbiguousChars_Enabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Encoding.NoAmbiguousChars = boolPtr(true)
	cfg.CommitMessage.Locale = "ko"
	// 키릴 А (U+0410) — 라틴 A처럼 보임
	content := "test \u0410 value"
	result := checker.FixFileContent(cfg, content)
	if !result.NeedsFixing() {
		t.Error("expected fix needed for ambiguous char")
	}
}

func TestFixFileContent_Original_Preserved(t *testing.T) {
	cfg := &config.Config{}
	content := "원본 내용"
	result := checker.FixFileContent(cfg, content)
	if result.Original != content {
		t.Errorf("Original should be unchanged, got %q", result.Original)
	}
}
