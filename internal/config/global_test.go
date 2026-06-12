package config_test

// 전역 설정 경로(GlobalConfigPath) 우선순위와 최상위 enabled 필드 동작 검증.
//
// 경로 우선순위:
//  1. $COMMIT_CHECKER_GLOBAL_CONFIG
//  2. $XDG_CONFIG_HOME/commit-checker/config.yml
//  3. os.UserConfigDir()/commit-checker/config.yml
//  4. ~/.commit-checker.yml (legacy)

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

// isolateGlobalPaths: 전역 설정 관련 환경 변수를 모두 빈 임시 디렉터리로 격리하고
// 격리된 HOME 경로를 반환한다.
func isolateGlobalPaths(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, "xdg-empty"))
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", "")
	return tmpHome
}

// writeFileAt: 상위 디렉터리를 만들며 파일을 기록.
func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeLegacyGlobal: legacy 경로(~/.commit-checker.yml)에 전역 설정을 기록.
func writeLegacyGlobal(t *testing.T, tmpHome, content string) string {
	t.Helper()
	path := filepath.Join(tmpHome, ".commit-checker.yml")
	writeFileAt(t, path, content)
	return path
}

// writeXDGGlobal: XDG 경로($XDG_CONFIG_HOME/commit-checker/config.yml)에 전역 설정을 기록.
func writeXDGGlobal(t *testing.T, content string) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "commit-checker", "config.yml")
	writeFileAt(t, path, content)
	return path
}

func TestGlobalConfigPath_None(t *testing.T) {
	isolateGlobalPaths(t)

	path, exists := config.GlobalConfigPath()
	if exists || path != "" {
		t.Errorf("전역 설정이 없으면 (\"\", false)를 기대: got (%q, %v)", path, exists)
	}
}

func TestGlobalConfigPath_EnvVar_최우선(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	// 하위 우선순위 경로(XDG, legacy)에도 파일을 두고 env 가 이기는지 확인
	writeXDGGlobal(t, "comment_language:\n  min_length: 2\n")
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")

	envPath := filepath.Join(t.TempDir(), "custom-global.yml")
	writeFileAt(t, envPath, "comment_language:\n  min_length: 1\n")
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", envPath)

	path, exists := config.GlobalConfigPath()
	if !exists || path != envPath {
		t.Errorf("env 경로가 최우선이어야 함: got (%q, %v), want (%q, true)", path, exists, envPath)
	}
}

func TestGlobalConfigPath_EnvVar_파일없으면전역설정없음(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	// env 가 가리키는 파일이 없으면 다른 경로(legacy)가 있어도 폴백하지 않음
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")
	t.Setenv("COMMIT_CHECKER_GLOBAL_CONFIG", filepath.Join(tmpHome, "no-such-file.yml"))

	path, exists := config.GlobalConfigPath()
	if exists || path != "" {
		t.Errorf("env 파일이 없으면 전역 설정 없음 처리: got (%q, %v)", path, exists)
	}
}

func TestGlobalConfigPath_XDG_Legacy보다우선(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")
	xdgPath := writeXDGGlobal(t, "comment_language:\n  min_length: 2\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != xdgPath {
		t.Errorf("XDG 경로가 legacy 보다 우선이어야 함: got (%q, %v), want (%q, true)", path, exists, xdgPath)
	}
}

func TestGlobalConfigPath_UserConfigDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 의 UserConfigDir(%AppData%)는 HOME 으로 격리할 수 없어 건너뜀")
	}
	isolateGlobalPaths(t)
	// XDG_CONFIG_HOME 을 비워 os.UserConfigDir() 폴백 경로를 사용
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	ucdPath := filepath.Join(dir, "commit-checker", "config.yml")
	writeFileAt(t, ucdPath, "comment_language:\n  min_length: 2\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != ucdPath {
		t.Errorf("UserConfigDir 경로를 기대: got (%q, %v), want (%q, true)", path, exists, ucdPath)
	}
}

func TestGlobalConfigPath_Legacy폴백(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	legacyPath := writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != legacyPath {
		t.Errorf("legacy 경로 폴백을 기대: got (%q, %v), want (%q, true)", path, exists, legacyPath)
	}
}

func TestLoad_XDG전역설정_프로젝트와병합(t *testing.T) {
	isolateGlobalPaths(t)
	writeXDGGlobal(t, "comment_language:\n  allowed_words:\n    - GlobalWord\n")

	path := writeConfig(t, `
comment_language:
  allowed_words:
    - ProjectWord
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var hasGlobal, hasProject bool
	for _, w := range cfg.CommentLanguage.AllowedWords {
		if w == "GlobalWord" {
			hasGlobal = true
		}
		if w == "ProjectWord" {
			hasProject = true
		}
	}
	if !hasGlobal || !hasProject {
		t.Errorf("전역(XDG)+프로젝트 allowed_words 병합 실패: %v", cfg.CommentLanguage.AllowedWords)
	}
}

func TestLoad_XDG전역설정_프로젝트설정없을때적용(t *testing.T) {
	isolateGlobalPaths(t)
	writeXDGGlobal(t, "comment_language:\n  locale: en\n")

	cfg, err := config.Load(filepath.Join(t.TempDir(), ".commit-checker.yml")) // 파일 없음
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.GetLocale() != "english" {
		t.Errorf("프로젝트 설정이 없으면 XDG 전역 설정이 적용되어야 함: got %q", cfg.CommentLanguage.GetLocale())
	}
}

func TestLoad_전역설정_구버전스키마자동마이그레이션(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	// v1.0.x 의 no_coauthor 필드: 마이그레이션 없이는 무시되어 기본값(true)이 됨.
	// 마이그레이션이 적용되면 no_ai_coauthor: false 로 변환되어 false 가 됨.
	writeLegacyGlobal(t, tmpHome, "commit_message:\n  no_coauthor: false\n")

	cfg, err := config.Load(filepath.Join(t.TempDir(), ".commit-checker.yml")) // 파일 없음
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommitMessage.IsNoAICoauthor() {
		t.Error("전역 설정의 no_coauthor: false 가 no_ai_coauthor 로 마이그레이션되어야 함")
	}
}

func TestConfig_IsEnabled_기본값true(t *testing.T) {
	cfg := &config.Config{}
	if !cfg.IsEnabled() {
		t.Error("enabled 미설정 시 기본값은 true 여야 함")
	}
}

func TestLoad_Enabled_프로젝트opt_out(t *testing.T) {
	isolateGlobalPaths(t)

	path := writeConfig(t, "enabled: false\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IsEnabled() {
		t.Error("프로젝트 설정의 enabled: false 가 반영되어야 함")
	}
}

func TestLoad_Enabled_병합_프로젝트우선(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	writeLegacyGlobal(t, tmpHome, "enabled: false\n")

	// 전역 enabled: false + 프로젝트 enabled: true → 프로젝트 우선
	path := writeConfig(t, "enabled: true\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsEnabled() {
		t.Error("프로젝트의 enabled: true 가 전역보다 우선해야 함")
	}

	// 전역 enabled: false + 프로젝트 미설정 → 전역 값 사용
	path2 := writeConfig(t, "comment_language:\n  min_length: 3\n")
	cfg2, err := config.Load(path2)
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if cfg2.IsEnabled() {
		t.Error("프로젝트가 enabled 를 설정하지 않으면 전역의 false 가 적용되어야 함")
	}
}
