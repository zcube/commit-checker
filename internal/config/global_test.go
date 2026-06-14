package config_test

// 전역 설정 경로(GlobalConfigPath) 우선순위와 최상위 enabled 필드 동작 검증.
//
// 적용 정책: 프로젝트 설정(.commit-checker.yml)이 존재하면 전역 설정은 완전히 무시되고
// 프로젝트 설정(+프로젝트가 선언한 preset/include)만 사용된다.
// 전역 설정은 프로젝트 설정이 없을 때만 사용된다.
//
// 경로 우선순위:
//  1. $COMMIT_CHECKER_GLOBAL_CONFIG
//  2. $XDG_CONFIG_HOME/commit-checker/config.yaml, config.yml
//  3. os.UserConfigDir()/commit-checker/config.yaml, config.yml
//  4. $HOME/.config/commit-checker/config.yaml, config.yml
//  5. ~/.commit-checker.yml (legacy)

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

// writeXDGGlobal: XDG 경로($XDG_CONFIG_HOME/commit-checker/config.yaml)에 전역 설정을 기록.
func writeXDGGlobal(t *testing.T, content string) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "commit-checker", "config.yaml")
	writeFileAt(t, path, content)
	return path
}

// writeHomeConfigGlobal: HOME 기반 기본 경로($HOME/.config/commit-checker/config.yaml)에 전역 설정을 기록.
func writeHomeConfigGlobal(t *testing.T, tmpHome, content string) string {
	t.Helper()
	path := filepath.Join(tmpHome, ".config", "commit-checker", "config.yaml")
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
	ucdPath := filepath.Join(dir, "commit-checker", "config.yaml")
	writeFileAt(t, ucdPath, "comment_language:\n  min_length: 2\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != ucdPath {
		t.Errorf("UserConfigDir 경로를 기대: got (%q, %v), want (%q, true)", path, exists, ucdPath)
	}
}

func TestGlobalConfigPath_UserConfigDirYmlFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 의 UserConfigDir(%AppData%)는 HOME 으로 격리할 수 없어 건너뜀")
	}
	tmpHome := isolateGlobalPaths(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	ucdPath := filepath.Join(dir, "commit-checker", "config.yml")
	writeFileAt(t, ucdPath, "comment_language:\n  min_length: 2\n")
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != ucdPath {
		t.Errorf("UserConfigDir config.yml 경로를 기대: got (%q, %v), want (%q, true)", path, exists, ucdPath)
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

func TestGlobalConfigPath_HomeConfigPriorToLegacy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 의 HOME/.config 경로는 환경 격리가 일관되지 않아 건너뜀")
	}
	tmpHome := isolateGlobalPaths(t)
	homePath := writeHomeConfigGlobal(t, tmpHome, "comment_language:\n  min_length: 4\n")
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != homePath {
		t.Errorf("$HOME/.config 경로가 legacy 보다 우선이어야 함: got (%q, %v), want (%q, true)", path, exists, homePath)
	}
}

func TestGlobalConfigPath_HomeConfigYmlFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows 의 HOME/.config 경로는 환경 격리가 일관되지 않아 건너뜀")
	}
	tmpHome := isolateGlobalPaths(t)
	homeYaml := filepath.Join(tmpHome, ".config", "commit-checker", "config.yml")
	writeFileAt(t, homeYaml, "comment_language:\n  min_length: 5\n")
	writeLegacyGlobal(t, tmpHome, "comment_language:\n  min_length: 3\n")

	path, exists := config.GlobalConfigPath()
	if !exists || path != homeYaml {
		t.Errorf("$HOME/.config/config.yml 경로가 legacy 보다 우선이어야 함: got (%q, %v), want (%q, true)", path, exists, homeYaml)
	}
}

func TestLoad_프로젝트설정존재시_전역무시(t *testing.T) {
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
	if hasGlobal {
		t.Errorf("프로젝트 설정이 있으면 전역(XDG) 값은 무시되어야 함: %v", cfg.CommentLanguage.AllowedWords)
	}
	if !hasProject {
		t.Errorf("프로젝트 allowed_words 는 적용되어야 함: %v", cfg.CommentLanguage.AllowedWords)
	}
}

func TestLoad_프로젝트설정존재시_전역locale미반영(t *testing.T) {
	isolateGlobalPaths(t)
	// 전역(en) + 프로젝트(ko): 전역 값이 전혀 섞이지 않고 프로젝트만 적용되어야 함
	writeXDGGlobal(t, "comment_language:\n  locale: en\ncommit_message:\n  locale: en\n")

	path := writeConfig(t, "comment_language:\n  locale: ko\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CommentLanguage.GetLocale() != "korean" {
		t.Errorf("프로젝트 locale(ko)만 적용되어야 함: got %q", cfg.CommentLanguage.GetLocale())
	}
	// 프로젝트가 설정하지 않은 필드에도 전역 값이 채워지면 안 됨 (기본값 ko 유지)
	if cfg.CommitMessage.Locale != "ko" {
		t.Errorf("전역의 commit_message.locale(en)이 반영되면 안 됨: got %q", cfg.CommitMessage.Locale)
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

func TestLoad_HomeConfigYaml_프로젝트설정없을때적용(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	homeYaml := writeHomeConfigGlobal(t, tmpHome, `commit_message:
  no_ai_coauthor: true
  no_emoji: true
  locale: ko
  conventional_commit:
    enabled: true
    locale: en
`)

	cfg, err := config.Load(filepath.Join(t.TempDir(), ".commit-checker.yaml")) // 파일 없음
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CommitMessage.ConventionalCommit.IsEnabled() {
		t.Fatalf("home config yaml should be loaded: %s", homeYaml)
	}
	if got := cfg.CommitMessage.ConventionalCommit.Locale; got != "en" {
		t.Fatalf("home config yaml should set conventional locale en, got %q", got)
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

func TestLoad_Enabled_프로젝트존재시_전역무시(t *testing.T) {
	tmpHome := isolateGlobalPaths(t)
	writeLegacyGlobal(t, tmpHome, "enabled: false\n")

	// 전역 enabled: false + 프로젝트 enabled: true → 프로젝트만 사용 (전역 무시)
	path := writeConfig(t, "enabled: true\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsEnabled() {
		t.Error("프로젝트 설정이 있으면 전역의 enabled: false 는 무시되어야 함")
	}

	// 전역 enabled: false + 프로젝트 미설정 → 전역이 무시되므로 기본값(true) 유지
	path2 := writeConfig(t, "comment_language:\n  min_length: 3\n")
	cfg2, err := config.Load(path2)
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if !cfg2.IsEnabled() {
		t.Error("프로젝트 설정이 존재하면 전역의 enabled: false 도 무시되어야 함 (기본값 true)")
	}

	// 프로젝트 설정 부재 → 전역의 enabled: false 적용 (전역 모드)
	cfg3, err := config.Load(filepath.Join(t.TempDir(), ".commit-checker.yml")) // 파일 없음
	if err != nil {
		t.Fatalf("Load3: %v", err)
	}
	if cfg3.IsEnabled() {
		t.Error("프로젝트 설정이 없으면 전역의 enabled: false 가 적용되어야 함")
	}
}

func TestLoad_전역모드_전역preset동작(t *testing.T) {
	isolateGlobalPaths(t)
	// 전역 설정이 선언한 preset.url 은 전역 모드에서도 기본값으로 병합되어야 함
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("comment_language:\n  min_length: 9\n  locale: en\n"))
	}))
	defer srv.Close()

	writeXDGGlobal(t, fmt.Sprintf("preset:\n  url: %s\ncomment_language:\n  min_length: 2\n", srv.URL))

	cfg, err := config.Load(filepath.Join(t.TempDir(), ".commit-checker.yml")) // 프로젝트 설정 없음
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 전역 본문 값이 preset 보다 우선
	if cfg.CommentLanguage.MinLength != 2 {
		t.Errorf("전역 본문이 preset 보다 우선해야 함: got %d, want 2", cfg.CommentLanguage.MinLength)
	}
	// 전역 본문에 없는 값은 preset 에서 채워짐
	if cfg.CommentLanguage.GetLocale() != "english" {
		t.Errorf("전역 preset 의 locale 이 기본값으로 적용되어야 함: got %q", cfg.CommentLanguage.GetLocale())
	}
}
