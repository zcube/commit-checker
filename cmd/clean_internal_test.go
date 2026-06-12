package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// setCleanYes: cleanYes 플래그를 설정하고 테스트 종료 시 복원.
func setCleanYes(t *testing.T, v bool) {
	t.Helper()
	orig := cleanYes
	cleanYes = v
	t.Cleanup(func() { cleanYes = orig })
}

// runCleanCmd: cleanCmd 의 RunE 를 직접 호출 (stdout 은 가로채서 버림).
// 스캔 진행 메시지(stderr) 억제를 위해 quiet 모드로 실행합니다.
func runCleanCmd(t *testing.T) error {
	t.Helper()
	origQuiet := globalQuiet
	globalQuiet = true
	t.Cleanup(func() { globalQuiet = origQuiet })

	var err error
	captureStdout(t, func() { err = cleanCmd.RunE(cleanCmd, nil) })
	return err
}

func TestCleanCmd_NoCacheDirs(t *testing.T) {
	newTestGitRepo(t)
	setCleanYes(t, false)

	if err := runCleanCmd(t); err != nil {
		t.Fatalf("캐시 디렉터리가 없으면 정상 종료해야 합니다: %v", err)
	}
}

func TestCleanCmd_DryRun_KeepsFiles(t *testing.T) {
	dir := newTestGitRepo(t)
	setCleanYes(t, false)

	// node_modules 는 부모에 package.json 등이 있어야 캐시 디렉터리로 인식됨
	writeTestFile(t, filepath.Join(dir, "package.json"), "{}\n")
	writeTestFile(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), "module.exports = {};\n")

	if err := runCleanCmd(t); err != nil {
		t.Fatalf("clean dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "pkg", "index.js")); err != nil {
		t.Error("dry-run 인데 파일이 삭제되었습니다")
	}
}

func TestCleanCmd_Yes_RemovesUntracked(t *testing.T) {
	dir := newTestGitRepo(t)
	setCleanYes(t, true)

	writeTestFile(t, filepath.Join(dir, "package.json"), "{}\n")
	writeTestFile(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), "module.exports = {};\n")

	if err := runCleanCmd(t); err != nil {
		t.Fatalf("clean --yes: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); !os.IsNotExist(err) {
		t.Error("--yes 인데 미추적 node_modules 가 삭제되지 않았습니다")
	}
}
