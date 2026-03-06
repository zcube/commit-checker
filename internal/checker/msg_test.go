package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func allChecksConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.CommitMessage.NoCoauthor = &t
	cfg.CommitMessage.NoUnicodeSpaces = &t
	cfg.CommitMessage.NoAmbiguousChars = &t
	cfg.CommitMessage.NoBadRunes = &t
	cfg.CommitMessage.Locale = "ko"
	return cfg
}

// --- co-author ---

func TestCheckMsg_CoAuthor(t *testing.T) {
	msg := "feat: add new feature\n\nCo-authored-by: Bot <bot@example.com>\n"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected co-author error, got none")
	}
}

func TestCheckMsg_CoAuthor_CaseInsensitive(t *testing.T) {
	msg := "fix: bug\n\nco-authored-by: Someone <x@y.com>\n"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected co-author error (case-insensitive), got none")
	}
}

func TestCheckMsg_Clean(t *testing.T) {
	msg := "feat: normal commit message\n\nBody text here.\n"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

// --- invisible characters ---

func TestCheckMsg_UnicodeSpaces_NBSP(t *testing.T) {
	// U+00A0 NO-BREAK SPACE
	msg := "feat: hello\u00A0world"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected NBSP error, got none")
	}
}

func TestCheckMsg_UnicodeSpaces_ZWSP(t *testing.T) {
	// U+200B ZERO WIDTH SPACE
	msg := "feat: hello\u200Bworld"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected zero-width space error, got none")
	}
}

func TestCheckMsg_BOM_Allowed(t *testing.T) {
	// U+FEFF BOM — must be allowed
	msg := "\uFEFFfeat: starts with BOM"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	// BOM should not trigger an invisible-char error
	for _, e := range errs {
		if containsSubstr(e, "invisible") {
			t.Errorf("BOM should be allowed but got invisible error: %s", e)
		}
	}
}

func TestCheckMsg_BiDiControl(t *testing.T) {
	// U+202E RIGHT-TO-LEFT OVERRIDE (used in Trojan source attacks)
	msg := "feat: hello\u202Eworld"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected BiDi control character error, got none")
	}
}

// --- ambiguous characters ---

func TestCheckMsg_AmbiguousChar_CyrillicA(t *testing.T) {
	// U+0410 Cyrillic CAPITAL LETTER A — looks like Latin A
	msg := "feat: \u0410mbiguous"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected ambiguous character error for Cyrillic А, got none")
	}
}

func TestCheckMsg_AmbiguousChar_CyrillicO(t *testing.T) {
	// U+043E Cyrillic SMALL LETTER O — looks like Latin o
	msg := "feat: c\u043Emmit"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected ambiguous character error for Cyrillic о, got none")
	}
}

func TestCheckMsg_NoAmbiguousInNormalKorean(t *testing.T) {
	// Korean hangul should not trigger ambiguous char errors
	msg := "feat: 새로운 기능 추가"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) != 0 {
		t.Errorf("Korean text should not trigger ambiguous errors, got: %v", errs)
	}
}

// --- bad UTF-8 ---

func TestCheckMsg_BadRune(t *testing.T) {
	// Insert an invalid UTF-8 byte directly
	msg := "feat: bad\x80rune"
	errs := checker.CheckMsg(allChecksConfig(), msg)
	if len(errs) == 0 {
		t.Error("expected bad UTF-8 rune error, got none")
	}
}

// --- disabled checks ---

func TestCheckMsg_DisabledCoAuthor(t *testing.T) {
	f := false
	cfg := allChecksConfig()
	cfg.CommitMessage.NoCoauthor = &f
	msg := "feat: ok\n\nCo-authored-by: Bot <bot@example.com>\n"
	errs := checker.CheckMsg(cfg, msg)
	for _, e := range errs {
		if containsSubstr(e, "co-author") {
			t.Errorf("co-author check should be disabled but got: %s", e)
		}
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstr(s, sub))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
