package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func TestCheckMsgCustomRules_Forbidden(t *testing.T) {
	rules := []config.CustomRule{
		{Name: "no-wip", Pattern: `(?i)^WIP`, Message: "WIP 접두사를 제거하세요"},
	}

	t.Run("forbidden pattern found", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("WIP: 작업 중", rules)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}
	})

	t.Run("no match", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("feat: 기능 추가", rules)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d", len(errs))
		}
	})

	t.Run("empty pattern skipped", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("any message", []config.CustomRule{{Name: "empty"}})
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d", len(errs))
		}
	})

	t.Run("invalid regex skipped", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("any message", []config.CustomRule{{Name: "bad", Pattern: "[invalid"}})
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d", len(errs))
		}
	})
}

func TestCheckMsgCustomRules_Required(t *testing.T) {
	rules := []config.CustomRule{
		{Name: "need-ticket", Pattern: `\[PROJ-\d+\]`, Message: "티켓 ID가 필요합니다", Required: true},
	}

	t.Run("required pattern present", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("[PROJ-123] fix: 버그 수정", rules)
		if len(errs) != 0 {
			t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("required pattern missing", func(t *testing.T) {
		errs := checker.CheckMsgCustomRules("fix: 버그 수정", rules)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}
	})
}

func TestCheckMsgCustomRules_MultiLine(t *testing.T) {
	rules := []config.CustomRule{
		{Name: "no-todo", Pattern: `TODO`, Message: "TODO를 제거하세요"},
	}

	content := "feat: 기능 추가\n\n본문 내용\nTODO: 나중에 처리\n끝"
	errs := checker.CheckMsgCustomRules(content, rules)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for TODO on line 4, got %d: %v", len(errs), errs)
	}
}

func TestCheckMsgCustomRules_DefaultMessage(t *testing.T) {
	rules := []config.CustomRule{
		{Name: "no-wip", Pattern: `WIP`},
	}
	errs := checker.CheckMsgCustomRules("WIP commit", rules)
	if len(errs) == 0 {
		t.Fatal("expected error")
	}
}
