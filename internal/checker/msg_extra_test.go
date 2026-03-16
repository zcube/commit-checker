package checker_test

import (
	"testing"

	"github.com/zcube/commit-checker/internal/checker"
	"github.com/zcube/commit-checker/internal/config"
)

func TestCheckMsg_SubjectLimit(t *testing.T) {
	enabled := true
	cfg := &config.Config{}
	cfg.CommitMessage.SubjectLimit.Enabled = &enabled
	cfg.CommitMessage.SubjectLimit.MaxLength = 20

	t.Run("subject within limit", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "짧은 제목")
		for _, e := range errs {
			if containsString(e, "너무 깁니다") || containsString(e, "too long") {
				t.Errorf("unexpected subject length error: %s", e)
			}
		}
	})

	t.Run("subject exceeds limit", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "이것은 20자를 초과하는 매우 긴 제목입니다")
		found := false
		for _, e := range errs {
			if containsString(e, "너무 깁니다") || containsString(e, "too long") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subject too long error, got: %v", errs)
		}
	})
}

func TestCheckMsg_BodyLineLimit(t *testing.T) {
	enabled := true
	cfg := &config.Config{}
	cfg.CommitMessage.BodyLineLimit.Enabled = &enabled
	cfg.CommitMessage.BodyLineLimit.MaxLength = 30

	t.Run("body within limit", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "제목\n\n짧은 본문")
		for _, e := range errs {
			if containsString(e, "너무 깁니다") || containsString(e, "too long") {
				t.Errorf("unexpected body line length error: %s", e)
			}
		}
	})

	t.Run("body line exceeds limit", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "제목\n\n이것은 30자를 훨씬 초과하는 매우 긴 본문 줄입니다 abcdefghijklmnopqrstuvwxyz")
		found := false
		for _, e := range errs {
			if containsString(e, "너무 깁니다") || containsString(e, "too long") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected body line too long error, got: %v", errs)
		}
	})
}

func TestCheckMsg_NoEmoji(t *testing.T) {
	noEmoji := true
	cfg := &config.Config{}
	cfg.CommitMessage.NoEmoji = &noEmoji

	t.Run("emoji in subject triggers error", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "feat: 새 기능 추가 🎉")
		found := false
		for _, e := range errs {
			if containsString(e, "이모지") || containsString(e, "emoji") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected emoji error, got: %v", errs)
		}
	})

	t.Run("no emoji passes", func(t *testing.T) {
		errs := checker.CheckMsg(cfg, "feat: 새 기능 추가")
		for _, e := range errs {
			if containsString(e, "이모지") || containsString(e, "emoji") {
				t.Errorf("unexpected emoji error: %s", e)
			}
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
