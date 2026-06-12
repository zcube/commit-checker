package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/i18n"
)

// setMsgFix: msgFix 플래그를 설정하고 테스트 종료 시 복원.
func setMsgFix(t *testing.T, v bool) {
	t.Helper()
	orig := msgFix
	msgFix = v
	t.Cleanup(func() { msgFix = orig })
}

// writeMsgTestFile: 커밋 메시지 파일을 임시 경로에 생성.
func writeMsgTestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	writeTestFile(t, path, content)
	return path
}

func TestMsgCmd_CleanMessage(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음 → 기본 설정
	setMsgFix(t, false)
	path := writeMsgTestFile(t, "feat: 새 기능 추가\n\n정상적인 커밋 메시지입니다.\n")

	if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
		t.Errorf("정상 메시지는 통과해야 합니다: %v", err)
	}
}

func TestMsgCmd_MissingFile_Error(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, false)

	err := msgCmd.RunE(msgCmd, []string{filepath.Join(dir, "no-such-file")})
	if err == nil {
		t.Fatal("없는 메시지 파일이면 에러가 나야 합니다")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("파일 읽기 에러를 기대했습니다: %v", err)
	}
}

func TestMsgCmd_BrokenConfig_Error(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "commit_message: [broken\n")
	setConfigFile(t, cfgPath)
	setMsgFix(t, false)
	path := writeMsgTestFile(t, "feat: 기능 추가\n")

	err := msgCmd.RunE(msgCmd, []string{path})
	if err == nil {
		t.Fatal("깨진 설정 파일이면 에러가 나야 합니다")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Errorf("설정 로드 에러를 기대했습니다: %v", err)
	}
}

func TestMsgCmd_Fix_RemovesAICoauthor(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, true)
	path := writeMsgTestFile(t, "feat: 기능 추가\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")

	// --fix 가 AI 공동 작성자 트레일러를 제거하면 검사도 통과해야 함
	if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
		t.Fatalf("--fix 후에는 통과해야 합니다: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Co-authored-by") {
		t.Errorf("AI 공동 작성자 트레일러가 제거되지 않았습니다:\n%s", data)
	}
}

func TestMsgCmd_Fix_RemovesInvisibleChar(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, true)
	// U+00A0 NBSP 는 자동 수정 대상
	path := writeMsgTestFile(t, "feat: 공백\u00a0수정\n")

	if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
		t.Fatalf("--fix 후에는 통과해야 합니다: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\u00a0") {
		t.Errorf("비가시 문자가 제거되지 않았습니다: %q", data)
	}
}

func TestMsgCmd_Violations_ReturnsSentinel(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, false)
	path := writeMsgTestFile(t, "feat: 기능 추가\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")

	var err error
	stderr := captureStderr(t, func() {
		err = msgCmd.RunE(msgCmd, []string{path})
	})
	if !errors.Is(err, errSilentExit) {
		t.Errorf("검사 실패 시 errSilentExit 를 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stderr, "Co-authored-by") {
		t.Errorf("위반 메시지가 stderr 에 출력되어야 합니다:\n%s", stderr)
	}
}

// setGlobalNoGuide: globalNoGuide 플래그를 설정하고 테스트 종료 시 복원.
func setGlobalNoGuide(t *testing.T, v bool) {
	t.Helper()
	orig := globalNoGuide
	globalNoGuide = v
	t.Cleanup(func() { globalNoGuide = orig })
}

func TestMsgCmd_Violations_PrintsGuideOnce(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, false)
	setGlobalNoGuide(t, false)
	// AI co-author + 비가시 문자: 위반이 여러 건이어도 가이드는 1회만
	path := writeMsgTestFile(t, "feat: 공백 수정\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")

	stderr := captureStderr(t, func() {
		_ = msgCmd.RunE(msgCmd, []string{path})
	})

	header := i18n.T("guide.header", nil)
	if n := strings.Count(stderr, header); n != 1 {
		t.Errorf("가이드 헤더는 1회만 출력되어야 합니다 (출력 %d회):\n%s", n, stderr)
	}
	if n := strings.Count(stderr, "[commit_message] "); n != 1 {
		t.Errorf("commit_message 가이드는 1회만 출력되어야 합니다 (출력 %d회):\n%s", n, stderr)
	}
}

func TestMsgCmd_Violations_NoGuideFlag_SuppressesGuide(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	setMsgFix(t, false)
	setGlobalNoGuide(t, true)
	path := writeMsgTestFile(t, "feat: 기능 추가\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")

	stderr := captureStderr(t, func() {
		_ = msgCmd.RunE(msgCmd, []string{path})
	})

	if strings.Contains(stderr, i18n.T("guide.header", nil)) {
		t.Errorf("--no-guide 가 켜지면 가이드가 출력되면 안 됩니다:\n%s", stderr)
	}
}

func TestMsgCmd_Violations_ConfigDisabled_SuppressesGuide(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "guide:\n  enabled: false\n")
	setConfigFile(t, cfgPath)
	setMsgFix(t, false)
	setGlobalNoGuide(t, false)
	path := writeMsgTestFile(t, "feat: 기능 추가\n\nCo-authored-by: Claude <noreply@anthropic.com>\n")

	stderr := captureStderr(t, func() {
		_ = msgCmd.RunE(msgCmd, []string{path})
	})

	if strings.Contains(stderr, i18n.T("guide.header", nil)) {
		t.Errorf("guide.enabled: false 설정이면 가이드가 출력되면 안 됩니다:\n%s", stderr)
	}
}
