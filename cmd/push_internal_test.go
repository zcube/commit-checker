package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/i18n"
)

func TestIsPushZeroSHA(t *testing.T) {
	tests := []struct {
		sha  string
		want bool
	}{
		{"0000000000000000000000000000000000000000", true},
		{"abc1234567890123456789012345678901234567", false},
		{"", false},
		{"00000000000000000000000000000000000000", false}, // 38 zeros
		{"000000000000000000000000000000000000000x", false},
	}
	for _, tt := range tests {
		got := isPushZeroSHA(tt.sha)
		if got != tt.want {
			t.Errorf("isPushZeroSHA(%q) = %v, want %v", tt.sha, got, tt.want)
		}
	}
}

func TestParsePushRanges_Empty(t *testing.T) {
	ranges := parsePushRanges(strings.NewReader(""))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges, got %d", len(ranges))
	}
}

func TestParsePushRanges_ValidRange(t *testing.T) {
	input := "refs/heads/main abc1234567890123456789012345678901234567 refs/heads/main def1234567890123456789012345678901234567\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	expected := "def1234567890123456789012345678901234567..abc1234567890123456789012345678901234567"
	if ranges[0] != expected {
		t.Errorf("range = %q, want %q", ranges[0], expected)
	}
}

func TestParsePushRanges_DeleteBranch(t *testing.T) {
	// local SHA is all zeros → branch deletion, skip
	input := "refs/heads/feat 0000000000000000000000000000000000000000 refs/heads/feat abc1234567890123456789012345678901234567\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for deletion, got %d: %v", len(ranges), ranges)
	}
}

func TestParsePushRanges_NewBranch_NoRemoteBase(t *testing.T) {
	// remote SHA is all zeros → new branch, but no upstream found → skip
	input := "refs/heads/new-feature abc1234567890123456789012345678901234567 refs/heads/new-feature 0000000000000000000000000000000000000000\n"
	// findPushRemoteBase will likely fail in test environment since there's no real remote
	// Result may be 0 or 1 depending on git state — just verify it doesn't crash
	_ = parsePushRanges(strings.NewReader(input))
}

func TestParsePushRanges_InvalidLine(t *testing.T) {
	input := "invalid line with only three parts\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for invalid line, got %d", len(ranges))
	}
}

func TestParsePushRanges_MultipleRefs(t *testing.T) {
	input := strings.Join([]string{
		"refs/heads/main abc1234567890123456789012345678901234567 refs/heads/main def1234567890123456789012345678901234567",
		"refs/heads/feature ghi1234567890123456789012345678901234567 refs/heads/feature jkl1234567890123456789012345678901234567",
	}, "\n") + "\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 2 {
		t.Errorf("expected 2 ranges, got %d", len(ranges))
	}
}

func TestParsePushRanges_BlankLines(t *testing.T) {
	input := "\n\n\n"
	ranges := parsePushRanges(strings.NewReader(input))
	if len(ranges) != 0 {
		t.Errorf("expected 0 ranges for blank lines, got %d", len(ranges))
	}
}

// setPushRange: pushRange 플래그를 설정하고 테스트 종료 시 복원.
func setPushRange(t *testing.T, v string) {
	t.Helper()
	orig := pushRange
	pushRange = v
	t.Cleanup(func() { pushRange = orig })
}

func TestPushCmd_Violations_ReturnsSentinel(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음 → 기본 설정

	// AI 공동 작성자 트레일러가 포함된 위반 커밋 생성
	gitRun(t, dir, "commit", "--allow-empty",
		"-m", "feat: 기능 추가",
		"-m", "Co-authored-by: Claude <noreply@anthropic.com>")
	setPushRange(t, "HEAD~1..HEAD")

	var err error
	stderr := captureStderr(t, func() {
		err = pushCmd.RunE(pushCmd, nil)
	})
	if !errors.Is(err, errSilentExit) {
		t.Errorf("위반 발견 시 errSilentExit 를 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stderr, "Co-authored-by") {
		t.Errorf("위반 메시지가 stderr 에 출력되어야 합니다:\n%s", stderr)
	}
}

func TestPushCmd_NoViolations_ReturnsNil(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))

	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: 정상 커밋")
	setPushRange(t, "HEAD~1..HEAD")

	if err := pushCmd.RunE(pushCmd, nil); err != nil {
		t.Errorf("위반이 없으면 nil 을 반환해야 합니다: %v", err)
	}
}

func TestPushCmd_Violations_PrintsGuideOnce(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setGlobalNoGuide(t, false)

	// 위반 커밋 2개: 여러 커밋이 실패해도 가이드는 1회만
	gitRun(t, dir, "commit", "--allow-empty",
		"-m", "feat: 기능 1",
		"-m", "Co-authored-by: Claude <noreply@anthropic.com>")
	gitRun(t, dir, "commit", "--allow-empty",
		"-m", "feat: 기능 2",
		"-m", "Co-authored-by: Claude <noreply@anthropic.com>")
	setPushRange(t, "HEAD~2..HEAD")

	stderr := captureStderr(t, func() {
		_ = pushCmd.RunE(pushCmd, nil)
	})

	header := i18n.T("guide.header", nil)
	if n := strings.Count(stderr, header); n != 1 {
		t.Errorf("여러 커밋이 실패해도 가이드 헤더는 1회만 출력되어야 합니다 (출력 %d회):\n%s", n, stderr)
	}
	if n := strings.Count(stderr, "[commit_message] "); n != 1 {
		t.Errorf("commit_message 가이드는 1회만 출력되어야 합니다 (출력 %d회):\n%s", n, stderr)
	}
}

func TestPushCmd_Violations_NoGuideFlag_SuppressesGuide(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setGlobalNoGuide(t, true)

	gitRun(t, dir, "commit", "--allow-empty",
		"-m", "feat: 기능 추가",
		"-m", "Co-authored-by: Claude <noreply@anthropic.com>")
	setPushRange(t, "HEAD~1..HEAD")

	stderr := captureStderr(t, func() {
		_ = pushCmd.RunE(pushCmd, nil)
	})

	if strings.Contains(stderr, i18n.T("guide.header", nil)) {
		t.Errorf("--no-guide 가 켜지면 가이드가 출력되면 안 됩니다:\n%s", stderr)
	}
}
