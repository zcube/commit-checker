package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// v100TestConfig: v1.0.0 스키마 설정 예시 (required_language, no_coauthor 사용).
const v100TestConfig = `comment_language:
  enabled: true
  required_language: korean
  min_length: 5

commit_message:
  no_coauthor: true
  no_unicode_spaces: true
`

// setMigrateDryRun: migrateDryRun 플래그를 설정하고 테스트 종료 시 복원.
func setMigrateDryRun(t *testing.T, v bool) {
	t.Helper()
	orig := migrateDryRun
	migrateDryRun = v
	t.Cleanup(func() { migrateDryRun = orig })
}

// runMigrateCmd: migrateCmd 의 RunE 를 직접 호출하고 stdout 을 반환.
func runMigrateCmd(t *testing.T) (string, error) {
	t.Helper()
	var err error
	out := captureStdout(t, func() { err = migrateCmd.RunE(migrateCmd, nil) })
	return out, err
}

func TestMigrate_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	setConfigFile(t, filepath.Join(dir, "missing.yml"))
	setMigrateDryRun(t, false)

	if _, err := runMigrateCmd(t); err == nil {
		t.Error("없는 파일이면 에러가 나야 합니다")
	}
}

func TestMigrate_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	original := "comment_language:\n  enabled: true\n  locale: ko\n"
	writeTestFile(t, cfgPath, original)
	setConfigFile(t, cfgPath)
	setMigrateDryRun(t, false)

	if _, err := runMigrateCmd(t); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Error("최신 버전이면 파일이 변경되면 안 됩니다")
	}
}

func TestMigrate_V100_DryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, v100TestConfig)
	setConfigFile(t, cfgPath)
	setMigrateDryRun(t, true)

	out, err := runMigrateCmd(t)
	if err != nil {
		t.Fatalf("migrate --dry-run: %v", err)
	}
	if !strings.Contains(out, "locale") {
		t.Errorf("dry-run 출력에 변환 결과가 없습니다:\n%s", out)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "required_language") {
		t.Error("dry-run 인데 파일이 수정되었습니다")
	}
}

func TestMigrate_V100_WritesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, v100TestConfig)
	setConfigFile(t, cfgPath)
	setMigrateDryRun(t, false)

	if _, err := runMigrateCmd(t); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	migrated := string(data)
	if strings.Contains(migrated, "required_language") {
		t.Errorf("required_language 가 남아 있습니다:\n%s", migrated)
	}
	if !strings.Contains(migrated, "locale: korean") {
		t.Errorf("locale 로 변환되지 않았습니다:\n%s", migrated)
	}
}

func TestMigrate_UnknownFormat_Error(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	// 매핑이 아닌 YAML 리스트 → 어느 스키마로도 파싱 불가
	writeTestFile(t, cfgPath, "- 항목1\n- 항목2\n")
	setConfigFile(t, cfgPath)
	setMigrateDryRun(t, false)

	if _, err := runMigrateCmd(t); err == nil {
		t.Error("인식할 수 없는 형식이면 에러가 나야 합니다")
	}
}
