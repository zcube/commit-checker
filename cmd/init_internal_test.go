package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
)

// setInitFlags: init 커맨드 플래그를 설정하고 테스트 종료 시 복원.
func setInitFlags(t *testing.T, force bool, lang string) {
	t.Helper()
	origForce, origLang := initForce, initLang
	initForce, initLang = force, lang
	t.Cleanup(func() { initForce, initLang = origForce, origLang })
}

// runInitCmd: initCmd 의 RunE 를 직접 호출 (stdout 은 가로채서 버림).
func runInitCmd(t *testing.T) error {
	t.Helper()
	var err error
	captureStdout(t, func() { err = initCmd.RunE(initCmd, nil) })
	return err
}

// ---- getDefaultConfig -----------------------------------------------------------

func TestGetDefaultConfig_Locales(t *testing.T) {
	tests := []struct {
		lang        string
		wantLocale  string
		wantExample string
	}{
		{"ko", "locale: ko", "기능"},
		{"ja", "locale: ja", "機能"},
		{"zh", "locale: zh", "功能"},
		{"en", "locale: en", "feat"},
		{"fr", "locale: ko", "기능"}, // 미지원 로케일은 ko 로 대체
	}
	for _, tt := range tests {
		got := getDefaultConfig(tt.lang)
		if !strings.Contains(got, tt.wantLocale) {
			t.Errorf("getDefaultConfig(%q): %q 가 없습니다", tt.lang, tt.wantLocale)
		}
		if !strings.Contains(got, tt.wantExample) {
			t.Errorf("getDefaultConfig(%q): 타입 별칭 예시 %q 가 없습니다", tt.lang, tt.wantExample)
		}
	}
}

func TestGetDefaultConfig_EmptyLang_DetectsLocale(t *testing.T) {
	// 빈 문자열이면 환경 로케일을 감지 — 어떤 로케일이든 유효한 설정이 나와야 함
	got := getDefaultConfig("")
	if !strings.Contains(got, "comment_language:") {
		t.Errorf("기본 설정에 comment_language 섹션이 없습니다:\n%s", got)
	}
}

func TestLocalizedExample(t *testing.T) {
	tests := []struct {
		locale string
		want   string
	}{
		{"ko", "기능"},
		{"ja", "機能"},
		{"zh", "功能"},
		{"en", "feat"},
		{"unknown", "feat"},
	}
	for _, tt := range tests {
		if got := localizedExample(tt.locale); got != tt.want {
			t.Errorf("localizedExample(%q) = %q, want %q", tt.locale, got, tt.want)
		}
	}
}

// ---- init 커맨드 ------------------------------------------------------------------

func TestInitCmd_CreatesValidConfig(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	setConfigFile(t, cfgPath)
	setInitFlags(t, false, "ko")

	if err := runInitCmd(t); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("설정 파일이 생성되지 않았습니다: %v", err)
	}

	// 생성된 설정이 실제로 로드/검증 가능한지 확인
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("생성된 설정 로드 실패: %v", err)
	}
	if warns := config.Validate(cfg, cfgPath); len(warns) != 0 {
		t.Errorf("생성된 설정에 경고가 있습니다: %v", warns)
	}
}

func TestInitCmd_AlreadyExists_Error(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	original := "comment_language:\n  locale: ko\n"
	writeTestFile(t, cfgPath, original)
	setConfigFile(t, cfgPath)
	setInitFlags(t, false, "ko")

	if err := runInitCmd(t); err == nil {
		t.Error("파일이 이미 존재하면 에러가 나야 합니다")
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Error("force 없이 기존 파일이 덮어써졌습니다")
	}
}

func TestInitCmd_ForceOverwrites(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "# 기존 내용\n")
	setConfigFile(t, cfgPath)
	setInitFlags(t, true, "ja")

	if err := runInitCmd(t); err != nil {
		t.Fatalf("init --force: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "locale: ja") {
		t.Errorf("--force 로 ja 설정이 생성되지 않았습니다:\n%s", data)
	}
}
