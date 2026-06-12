package cmd

// prepare-msg 커맨드(소스별 동작, 힌트 내용, 중복 방지) 테스트.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/i18n"
)

// writePrepareMsgFile: prepare-commit-msg 훅이 받는 커밋 메시지 파일을 생성.
func writePrepareMsgFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	writeTestFile(t, path, content)
	return path
}

// readFileString: 파일 내용을 문자열로 읽음.
func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPrepareMsgCmd_UserHasMessage_NoChange(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음 → 기본 설정

	// 사용자가 이미 메시지를 가진 source 는 파일을 건드리지 않고 종료 0
	for _, source := range []string{"message", "merge", "squash", "commit"} {
		orig := "feat: 이미 작성된 메시지\n"
		path := writePrepareMsgFile(t, orig)

		if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path, source, "HEAD"}); err != nil {
			t.Errorf("source=%s: 에러 없이 종료해야 합니다: %v", source, err)
		}
		if got := readFileString(t, path); got != orig {
			t.Errorf("source=%s: 파일이 변경되면 안 됩니다:\n%s", source, got)
		}
	}
}

func TestPrepareMsgCmd_EmptySource_AppendsHint(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 기본 설정 (no_ai_coauthor 활성)
	path := writePrepareMsgFile(t, "")

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatalf("prepare-msg 는 에러 없이 종료해야 합니다: %v", err)
	}

	got := readFileString(t, path)
	header := "# " + i18n.T("cmd.prepare_msg.hint_header", nil)
	if !strings.Contains(got, header) {
		t.Errorf("힌트 헤더가 추가되어야 합니다:\n%s", got)
	}
	// 기본 설정에서는 no_ai_coauthor 가 활성 → coauthor 안내가 포함되어야 함
	if !strings.Contains(got, "# "+i18n.T("cmd.prepare_msg.hint_coauthor", nil)) {
		t.Errorf("AI co-author 안내가 포함되어야 합니다:\n%s", got)
	}
	// 기본 설정에서는 conventional commit 이 비활성 → 형식 안내는 없어야 함
	if strings.Contains(got, "# "+i18n.T("cmd.prepare_msg.hint_format", nil)) {
		t.Errorf("conventional commit 비활성 시 형식 안내는 없어야 합니다:\n%s", got)
	}
	// 모든 힌트 줄은 # 주석이어야 함 (git 이 커밋 시 제거)
	for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			t.Errorf("힌트 줄은 모두 # 으로 시작해야 합니다: %q", line)
		}
	}
}

func TestPrepareMsgCmd_TemplateSource_AppendsHint(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	path := writePrepareMsgFile(t, "# 템플릿 내용\n")

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path, "template"}); err != nil {
		t.Fatalf("source=template 은 힌트를 추가해야 합니다: %v", err)
	}
	got := readFileString(t, path)
	if !strings.Contains(got, "# "+i18n.T("cmd.prepare_msg.hint_header", nil)) {
		t.Errorf("source=template 일 때 힌트가 추가되어야 합니다:\n%s", got)
	}
	if !strings.Contains(got, "# 템플릿 내용") {
		t.Errorf("기존 템플릿 내용이 유지되어야 합니다:\n%s", got)
	}
}

func TestPrepareMsgCmd_Duplicate_NotAppendedTwice(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	path := writePrepareMsgFile(t, "")

	// 두 번 실행해도 힌트 블록은 1회만 추가되어야 함
	for i := 0; i < 2; i++ {
		if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
			t.Fatalf("%d번째 실행 실패: %v", i+1, err)
		}
	}
	got := readFileString(t, path)
	header := "# " + i18n.T("cmd.prepare_msg.hint_header", nil)
	if n := strings.Count(got, header); n != 1 {
		t.Errorf("힌트 헤더는 1회만 추가되어야 합니다 (현재 %d회):\n%s", n, got)
	}
}

func TestPrepareMsgCmd_ConventionalAndLanguage_HintContent(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, `commit_message:
  conventional_commit:
    enabled: true
    types: [feat, fix, docs]
  language_check:
    enabled: true
    locale: ko
`)
	setConfigFile(t, cfgPath)
	path := writePrepareMsgFile(t, "")

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	got := readFileString(t, path)

	if !strings.Contains(got, "# "+i18n.T("cmd.prepare_msg.hint_format", nil)) {
		t.Errorf("conventional commit 형식 안내가 포함되어야 합니다:\n%s", got)
	}
	wantTypes := "# " + i18n.T("cmd.prepare_msg.hint_types", map[string]any{"Types": "feat, fix, docs"})
	if !strings.Contains(got, wantTypes) {
		t.Errorf("설정된 type 목록(feat, fix, docs)이 포함되어야 합니다:\n%s", got)
	}
	wantLang := "# " + i18n.T("cmd.prepare_msg.hint_language", map[string]any{
		"Language": i18n.T("lang.korean", nil),
	})
	if !strings.Contains(got, wantLang) {
		t.Errorf("언어 안내(한국어)가 포함되어야 합니다:\n%s", got)
	}
}

func TestPrepareMsgCmd_CommitMessageDisabled_NoHint(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "commit_message:\n  enabled: false\n")
	setConfigFile(t, cfgPath)
	orig := ""
	path := writePrepareMsgFile(t, orig)

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	if got := readFileString(t, path); got != orig {
		t.Errorf("commit_message 비활성 시 파일이 변경되면 안 됩니다:\n%s", got)
	}
}

func TestPrepareMsgCmd_AllPoliciesDisabled_NoHeaderOnlyBlock(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	// 모든 안내 대상 정책 비활성 → 헤더만 단독으로 추가되면 안 됨
	writeTestFile(t, cfgPath, `commit_message:
  no_ai_coauthor: false
`)
	setConfigFile(t, cfgPath)
	orig := ""
	path := writePrepareMsgFile(t, orig)

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	if got := readFileString(t, path); got != orig {
		t.Errorf("활성화된 정책이 없으면 힌트를 추가하지 않아야 합니다:\n%s", got)
	}
}

func TestPrepareMsgHint_LanguageAny_Skipped(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, `commit_message:
  no_ai_coauthor: false
  language_check:
    enabled: true
    locale: any
`)
	setConfigFile(t, cfgPath)
	orig := ""
	path := writePrepareMsgFile(t, orig)

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	// locale=any 는 특정 언어 요구가 아니므로 언어 안내를 생성하지 않음
	if got := readFileString(t, path); got != orig {
		t.Errorf("locale=any 면 언어 안내 없이 파일이 그대로여야 합니다:\n%s", got)
	}
}

func TestPrepareMsgCmd_ExistingContent_AppendedAfter(t *testing.T) {
	isolateHome(t)
	dir := chdirTemp(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml"))
	// 개행 없이 끝나는 기존 내용 뒤에도 안전하게 추가되어야 함
	path := writePrepareMsgFile(t, "feat: 작성 중인 메시지")

	if err := prepareMsgCmd.RunE(prepareMsgCmd, []string{path}); err != nil {
		t.Fatal(err)
	}
	got := readFileString(t, path)
	if !strings.HasPrefix(got, "feat: 작성 중인 메시지\n") {
		t.Errorf("기존 내용이 맨 앞에 유지되어야 합니다:\n%s", got)
	}
	if !strings.Contains(got, "# "+i18n.T("cmd.prepare_msg.hint_header", nil)) {
		t.Errorf("힌트가 기존 내용 뒤에 추가되어야 합니다:\n%s", got)
	}
}
