package cmd

import (
	"path/filepath"
	"testing"
)

// runValidateCmd: validateCmd 의 RunE 를 직접 호출 (stdout 은 가로채서 버림).
func runValidateCmd(t *testing.T) error {
	t.Helper()
	var err error
	captureStdout(t, func() { err = validateCmd.RunE(validateCmd, nil) })
	return err
}

func TestValidate_ValidConfig(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "comment_language:\n  enabled: true\n  locale: ko\n")
	setConfigFile(t, cfgPath)

	if err := runValidateCmd(t); err != nil {
		t.Errorf("유효한 설정인데 에러: %v", err)
	}
}

func TestValidate_MissingConfig_UsesDefaults(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, "nonexistent.yml"))

	// 설정 파일이 없으면 기본 설정으로 검증 통과해야 함
	if err := runValidateCmd(t); err != nil {
		t.Errorf("기본 설정은 경고가 없어야 합니다: %v", err)
	}
}

func TestValidate_InvalidGlobPattern_Warns(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	// "[" 는 잘못된 glob 패턴 → 검증 경고 발생
	writeTestFile(t, cfgPath, "comment_language:\n  ignore_files:\n    - \"[\"\n")
	setConfigFile(t, cfgPath)

	if err := runValidateCmd(t); err == nil {
		t.Error("잘못된 glob 패턴이면 경고 에러가 나야 합니다")
	}
}

func TestValidate_BrokenYAML_Error(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "comment_language: [broken\n")
	setConfigFile(t, cfgPath)

	if err := runValidateCmd(t); err == nil {
		t.Error("깨진 YAML 이면 로드 에러가 나야 합니다")
	}
}
