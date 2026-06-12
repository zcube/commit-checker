package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
