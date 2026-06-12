package cmd

// 최상위 enabled 필드(리포 단위 opt-out)와 --require-config 플래그(전역 opt-in)의
// 훅 진입 커맨드 동작 검증.

import (
	"errors"
	"path/filepath"
	"testing"
)

// setRequireConfig: 패키지 전역 globalRequireConfig 를 설정하고 테스트 종료 시 복원.
func setRequireConfig(t *testing.T, v bool) {
	t.Helper()
	orig := globalRequireConfig
	globalRequireConfig = v
	t.Cleanup(func() { globalRequireConfig = orig })
}

// badAICoauthorMsg: 기본 설정에서 위반(AI 공동 작성자)이 되는 커밋 메시지.
const badAICoauthorMsg = "feat: 기능 추가\n\nCo-authored-by: Claude <noreply@anthropic.com>\n"

func TestMsgCmd_EnabledFalse_검사없이성공종료(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "enabled: false\n")
	setConfigFile(t, cfgPath)
	setMsgFix(t, false)
	path := writeMsgTestFile(t, badAICoauthorMsg)

	stderr := captureStderr(t, func() {
		if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
			t.Errorf("enabled: false 이면 위반이 있어도 성공 종료해야 합니다: %v", err)
		}
	})
	if stderr != "" {
		t.Errorf("enabled: false 이면 아무 출력이 없어야 합니다: %q", stderr)
	}
}

func TestMsgCmd_EnabledFalse_전역설정있어도opt_out(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	// 전역 설정(COMMIT_CHECKER_GLOBAL_CONFIG)은 검사 활성 상태,
	// 프로젝트 설정이 enabled: false 한 줄로 opt-out 하는 시나리오
	globalPath := filepath.Join(dir, "global.yml")
	writeTestFile(t, globalPath, "commit_message:\n  no_ai_coauthor: true\n")
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", globalPath)

	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "enabled: false\n")
	setConfigFile(t, cfgPath)
	setMsgFix(t, false)
	path := writeMsgTestFile(t, badAICoauthorMsg)

	if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
		t.Errorf("opt-out 리포는 전역 설정이 있어도 성공 종료해야 합니다: %v", err)
	}
}

func TestPushCmd_EnabledFalse_검사없이성공종료(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "enabled: false\n")
	setConfigFile(t, cfgPath)

	if err := pushCmd.RunE(pushCmd, nil); err != nil {
		t.Errorf("enabled: false 이면 push 검사도 성공 종료해야 합니다: %v", err)
	}
}

func TestMsgCmd_RequireConfig_설정없으면무동작종료(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음
	setRequireConfig(t, true)
	setMsgFix(t, false)
	path := writeMsgTestFile(t, badAICoauthorMsg)

	stderr := captureStderr(t, func() {
		if err := msgCmd.RunE(msgCmd, []string{path}); err != nil {
			t.Errorf("--require-config + 설정 없음이면 무동작 성공 종료해야 합니다: %v", err)
		}
	})
	if stderr != "" {
		t.Errorf("--require-config 건너뜀 시 아무 출력이 없어야 합니다: %q", stderr)
	}
}

func TestMsgCmd_RequireConfig_설정있으면정상검사(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "commit_message:\n  no_ai_coauthor: true\n")
	setConfigFile(t, cfgPath)
	setRequireConfig(t, true)
	setMsgFix(t, false)
	path := writeMsgTestFile(t, badAICoauthorMsg)

	stderr := captureStderr(t, func() {
		err := msgCmd.RunE(msgCmd, []string{path})
		if !errors.Is(err, errSilentExit) {
			t.Errorf("설정 파일이 있으면 정상 검사로 위반을 보고해야 합니다: %v", err)
		}
	})
	if stderr == "" {
		t.Error("위반 보고가 stderr 에 출력되어야 합니다")
	}
}

func TestPushCmd_RequireConfig_설정없으면무동작종료(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음
	setRequireConfig(t, true)

	if err := pushCmd.RunE(pushCmd, nil); err != nil {
		t.Errorf("--require-config + 설정 없음이면 push 도 무동작 성공 종료해야 합니다: %v", err)
	}
}
