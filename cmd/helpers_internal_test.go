package cmd

// 내부 단위 테스트 공용 헬퍼 모음.
// 패키지 전역 플래그 변수를 사용하는 테스트는 여기 헬퍼로 저장/복원합니다.

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// isolateHome: HOME/XDG_CONFIG_HOME/COMMIT_CHECKER_GLOBAL_CONFIG 를 임시 값으로 바꿔
// 전역 설정(legacy ~/.commit-checker.yml, XDG 경로, 환경 변수 경로)의 영향을 차단.
func isolateHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg-config"))
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", "")
}

// setConfigFile: 패키지 전역 configFile 을 설정하고 테스트 종료 시 복원.
func setConfigFile(t *testing.T, path string) {
	t.Helper()
	orig := configFile
	configFile = path
	t.Cleanup(func() { configFile = orig })
}

// chdirTemp: git 저장소가 아닌 임시 디렉터리로 작업 디렉터리를 변경하고 종료 시 복원.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// writeTestFile: 경로에 내용을 기록 (필요한 상위 디렉터리 자동 생성).
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// gitRun: dir 에서 git 명령을 실행 (실패 시 테스트 중단).
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	// 전역 설정의 GPG 서명 요구를 무력화
	c.Env = append(os.Environ(), "GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=commit.gpgsign",
		"GIT_CONFIG_VALUE_0=false")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// captureStdout: fn 실행 동안 표준 출력을 가로채 문자열로 반환.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	_ = w.Close()
	out := <-done
	os.Stdout = orig
	return out
}

// captureStderr: fn 실행 동안 표준 에러를 가로채 문자열로 반환.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	_ = w.Close()
	out := <-done
	os.Stderr = orig
	return out
}
