package checker_test

import (
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func conventionalConfig() *config.Config {
	t := true
	cfg := &config.Config{}
	cfg.CommitMessage.ConventionalCommit.Enabled = &t
	return cfg
}

// --- valid formats ---

func TestConventional_ValidFeat(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat: add new login page\n")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConventional_ValidFix(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "fix: correct null pointer in auth\n")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConventional_ValidWithScope(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat(auth): add OAuth2 support\n")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConventional_ValidBreakingChange(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat!: remove deprecated API\n")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConventional_ValidBreakingChangeWithScope(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat(api)!: remove v1 endpoints\n")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestConventional_ValidAllDefaultTypes(t *testing.T) {
	for _, typ := range config.DefaultConventionalTypes {
		msg := typ + ": some description\n"
		errs := checker.CheckMsg(conventionalConfig(), msg)
		if len(errs) != 0 {
			t.Errorf("type %q should be valid, got: %v", typ, errs)
		}
	}
}

func TestConventional_ValidWithBody(t *testing.T) {
	msg := "feat: add something\n\nThis is the body.\n\nBREAKING CHANGE: old API removed\n"
	errs := checker.CheckMsg(conventionalConfig(), msg)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

// --- invalid formats ---

func TestConventional_MissingColon(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat add something\n")
	if len(errs) == 0 {
		t.Error("expected format error, got none")
	}
}

func TestConventional_MissingDescription(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat: \n")
	if len(errs) == 0 {
		t.Error("expected format error for empty description, got none")
	}
}

func TestConventional_NoSpaceAfterColon(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "feat:add something\n")
	if len(errs) == 0 {
		t.Error("expected format error for missing space after colon, got none")
	}
}

func TestConventional_InvalidType(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "unknown: add something\n")
	if len(errs) == 0 {
		t.Error("expected type error, got none")
	}
	if !strings.Contains(errs[0], "unknown") {
		t.Errorf("error should mention the invalid type, got: %s", errs[0])
	}
}

func TestConventional_PlainMessage(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "just a plain commit message\n")
	if len(errs) == 0 {
		t.Error("expected format error for plain message, got none")
	}
}

// --- skip rules ---

func TestConventional_SkipMergeCommit(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "Merge branch 'feature' into main\n")
	if len(errs) != 0 {
		t.Errorf("merge commit should be skipped, got: %v", errs)
	}
}

func TestConventional_SkipRevertCommit(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "Revert \"feat: add something\"\n")
	if len(errs) != 0 {
		t.Errorf("revert commit should be skipped, got: %v", errs)
	}
}

func TestConventional_SkipFixupCommit(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "fixup! feat: add something\n")
	if len(errs) != 0 {
		t.Errorf("fixup! commit should be skipped, got: %v", errs)
	}
}

func TestConventional_SkipSquashCommit(t *testing.T) {
	errs := checker.CheckMsg(conventionalConfig(), "squash! feat: add something\n")
	if len(errs) != 0 {
		t.Errorf("squash! commit should be skipped, got: %v", errs)
	}
}

// --- disabled ---

func TestConventional_DisabledByDefault(t *testing.T) {
	// Without enabling, plain messages should pass.
	cfg := &config.Config{}
	errs := checker.CheckMsg(cfg, "just a plain commit message\n")
	if len(errs) != 0 {
		t.Errorf("conventional check disabled by default, got: %v", errs)
	}
}

// --- custom types ---

func TestConventional_CustomTypes(t *testing.T) {
	t2 := true
	cfg := &config.Config{}
	cfg.CommitMessage.ConventionalCommit.Enabled = &t2
	cfg.CommitMessage.ConventionalCommit.Types = []string{"task", "hotfix"}

	errs := checker.CheckMsg(cfg, "task: do something\n")
	if len(errs) != 0 {
		t.Errorf("custom type 'task' should be valid, got: %v", errs)
	}

	errs = checker.CheckMsg(cfg, "feat: standard type should now fail\n")
	if len(errs) == 0 {
		t.Error("'feat' should be invalid when custom types are set, got none")
	}
}

// --- require scope ---

func TestConventional_RequireScope_Missing(t *testing.T) {
	t2 := true
	cfg := &config.Config{}
	cfg.CommitMessage.ConventionalCommit.Enabled = &t2
	cfg.CommitMessage.ConventionalCommit.RequireScope = &t2

	errs := checker.CheckMsg(cfg, "feat: add something\n")
	if len(errs) == 0 {
		t.Error("expected scope-required error, got none")
	}
}

func TestConventional_RequireScope_Present(t *testing.T) {
	t2 := true
	cfg := &config.Config{}
	cfg.CommitMessage.ConventionalCommit.Enabled = &t2
	cfg.CommitMessage.ConventionalCommit.RequireScope = &t2

	errs := checker.CheckMsg(cfg, "feat(auth): add something\n")
	if len(errs) != 0 {
		t.Errorf("scope provided, expected no errors, got: %v", errs)
	}
}
