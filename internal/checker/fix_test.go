package checker_test

import (
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func TestFixMsg_RemoveCoauthor_AI(t *testing.T) {
	cfg := allChecksConfig()
	msg := "feat: add feature\n\nSome body.\n\nCo-authored-by: Claude <noreply@anthropic.com>\n"
	result := checker.FixMsg(cfg, msg)
	if !result.NeedsFixing() {
		t.Fatal("expected fix needed for AI co-author")
	}
	if strings.Contains(result.Fixed, "Co-authored-by") {
		t.Errorf("AI co-author trailer should be removed, got:\n%s", result.Fixed)
	}
	if len(result.Changes) == 0 {
		t.Error("expected at least one change description")
	}
}

func TestFixMsg_KeepHumanCoauthor(t *testing.T) {
	cfg := allChecksConfig()
	msg := "feat: add feature\n\nCo-authored-by: Alice <alice@myteam.com>\n"
	result := checker.FixMsg(cfg, msg)
	if result.NeedsFixing() {
		t.Errorf("human co-author should not be removed, got changes: %v", result.Changes)
	}
	if !strings.Contains(result.Fixed, "Co-authored-by: Alice") {
		t.Error("human co-author should be preserved")
	}
}

func TestFixMsg_ReplaceAmbiguousChar(t *testing.T) {
	cfg := allChecksConfig()
	// U+0410 Cyrillic А → should be replaced with Latin A
	msg := "feat: \u0410dd feature"
	result := checker.FixMsg(cfg, msg)
	if !result.NeedsFixing() {
		t.Fatal("expected fix needed for ambiguous char")
	}
	if strings.ContainsRune(result.Fixed, '\u0410') {
		t.Error("Cyrillic А should have been replaced with Latin A")
	}
	if !strings.Contains(result.Fixed, "A") {
		t.Error("expected Latin A in fixed message")
	}
}

func TestFixMsg_ReplaceInvisibleSpace(t *testing.T) {
	cfg := allChecksConfig()
	// U+00A0 NO-BREAK SPACE → should become regular space
	msg := "feat: hello\u00A0world"
	result := checker.FixMsg(cfg, msg)
	if !result.NeedsFixing() {
		t.Fatal("expected fix needed for NBSP")
	}
	if strings.ContainsRune(result.Fixed, '\u00A0') {
		t.Error("NBSP should have been replaced")
	}
	if !strings.Contains(result.Fixed, "hello world") {
		t.Errorf("expected 'hello world' after fix, got: %q", result.Fixed)
	}
}

func TestFixMsg_RemoveBadRune(t *testing.T) {
	cfg := allChecksConfig()
	msg := "feat: bad\x80rune"
	result := checker.FixMsg(cfg, msg)
	if !result.NeedsFixing() {
		t.Fatal("expected fix needed for bad UTF-8 rune")
	}
	if strings.Contains(result.Fixed, "\x80") {
		t.Error("invalid UTF-8 byte should have been removed")
	}
}

func TestFixMsg_Clean_NoChanges(t *testing.T) {
	cfg := allChecksConfig()
	msg := "feat: normal commit\n\nBody text here.\n"
	result := checker.FixMsg(cfg, msg)
	if result.NeedsFixing() {
		t.Errorf("clean message should need no fixing, got changes: %v", result.Changes)
	}
	if result.Fixed != result.Original {
		t.Error("fixed should equal original for clean message")
	}
}

func TestFixMsg_MultipleIssues(t *testing.T) {
	cfg := allChecksConfig()
	// Ambiguous char + AI co-author
	msg := "feat: \u0410dd\n\nCo-authored-by: Copilot <github-copilot[bot]@users.noreply.github.com>\n"
	result := checker.FixMsg(cfg, msg)
	if !result.NeedsFixing() {
		t.Fatal("expected multiple fixes")
	}
	if len(result.Changes) < 2 {
		t.Errorf("expected at least 2 changes, got %d: %v", len(result.Changes), result.Changes)
	}
	if strings.Contains(result.Fixed, "Co-authored-by") {
		t.Error("co-author should be removed")
	}
	if strings.ContainsRune(result.Fixed, '\u0410') {
		t.Error("Cyrillic А should be replaced")
	}
}

func TestFixMsg_DisabledChecks(t *testing.T) {
	f := false
	cfg := &config.Config{}
	cfg.CommitMessage.NoAICoauthor = &f
	cfg.CommitMessage.NoUnicodeSpaces = &f
	cfg.CommitMessage.NoAmbiguousChars = &f
	cfg.CommitMessage.NoBadRunes = &f

	msg := "feat: \u0410dd\n\nCo-authored-by: Bot <x@y.com>\n"
	result := checker.FixMsg(cfg, msg)
	if result.NeedsFixing() {
		t.Errorf("all checks disabled: expected no fixes, got: %v", result.Changes)
	}
}
