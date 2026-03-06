package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func msgLangConfig(lang string) *config.Config {
	t := true
	f := false
	cfg := &config.Config{}
	cfg.CommitMessage.NoAICoauthor = &f
	cfg.CommitMessage.NoUnicodeSpaces = &f
	cfg.CommitMessage.NoAmbiguousChars = &f
	cfg.CommitMessage.NoBadRunes = &f
	cfg.CommitMessage.LanguageCheck.Enabled = &t
	cfg.CommitMessage.LanguageCheck.RequiredLanguage = lang
	cfg.CommitMessage.LanguageCheck.MinLength = 5
	cfg.CommitMessage.LanguageCheck.SkipPrefixes = []string{"Merge", "Revert", "fixup!", "squash!"}
	return cfg
}

func TestCheckMsgLanguage_Korean_Pass(t *testing.T) {
	cfg := msgLangConfig("korean")
	msg := "새로운 기능 추가\n\n데이터베이스 연결 로직을 개선했습니다.\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("expected no errors for Korean message, got: %v", errs)
	}
}

func TestCheckMsgLanguage_Korean_Fail_English(t *testing.T) {
	cfg := msgLangConfig("korean")
	msg := "add new feature for user authentication\n\nThis improves the login flow.\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) == 0 {
		t.Error("expected language error for English message with Korean required, got none")
	}
}

func TestCheckMsgLanguage_English_Pass(t *testing.T) {
	cfg := msgLangConfig("english")
	msg := "add new authentication feature\n\nThis improves the login flow significantly.\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("expected no errors for English message, got: %v", errs)
	}
}

func TestCheckMsgLanguage_SkipPrefix_Merge(t *testing.T) {
	cfg := msgLangConfig("korean")
	msg := "Merge branch 'feature/auth' into main\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("Merge commits should be skipped, got: %v", errs)
	}
}

func TestCheckMsgLanguage_SkipPrefix_Revert(t *testing.T) {
	cfg := msgLangConfig("korean")
	msg := "Revert \"add broken feature\"\n\nThis reverts commit abc123.\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("Revert commits should be skipped, got: %v", errs)
	}
}

func TestCheckMsgLanguage_ShortLine_Skipped(t *testing.T) {
	cfg := msgLangConfig("korean")
	// Subject is too short to trigger language check (< min_length letters)
	msg := "fix\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("short commit message should be skipped, got: %v", errs)
	}
}

func TestCheckMsgLanguage_Disabled(t *testing.T) {
	cfg := msgLangConfig("korean")
	f := false
	cfg.CommitMessage.LanguageCheck.Enabled = &f
	msg := "add new feature for user authentication\n"
	errs := checker.CheckMsg(cfg, msg)
	if len(errs) != 0 {
		t.Errorf("disabled language check should produce no errors, got: %v", errs)
	}
}
