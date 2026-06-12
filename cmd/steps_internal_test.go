package cmd

// --only 플래그(stepDefsFor / cfgWithOnlyEnabled)와 run/diff 커맨드 통합 동작 테스트.

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zcube/commit-checker/internal/config"
)

// runCmdE: 커맨드에 context 를 설정한 뒤 RunE 를 직접 호출.
// (Execute 를 거치지 않으면 cmd.Context() 가 nil 이므로 명시적으로 주입)
func runCmdE(c *cobra.Command, args []string) error {
	c.SetContext(context.Background())
	return c.RunE(c, args)
}

// setRunOnly: runOnly 플래그를 설정하고 테스트 종료 시 복원.
func setRunOnly(t *testing.T, v []string) {
	t.Helper()
	orig := runOnly
	runOnly = v
	t.Cleanup(func() { runOnly = orig })
}

// setDiffOnly: diffOnly 플래그를 설정하고 테스트 종료 시 복원.
func setDiffOnly(t *testing.T, v []string) {
	t.Helper()
	orig := diffOnly
	diffOnly = v
	t.Cleanup(func() { diffOnly = orig })
}

// ---- stepDefsFor -------------------------------------------------------------

func TestStepDefsFor_NoOnly_ReturnsAll(t *testing.T) {
	runDefs, err := stepDefsFor(false, nil)
	if err != nil {
		t.Fatal(err)
	}
	diffDefs, err := stepDefsFor(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	// run 은 runFn 이 있는 step 만, diff 는 diffFn 이 있는 step 만 포함해야 함
	for _, d := range runDefs {
		if d.runFn == nil {
			t.Errorf("run 정의에 runFn 이 없는 step 포함: %s", d.category)
		}
	}
	for _, d := range diffDefs {
		if d.diffFn == nil {
			t.Errorf("diff 정의에 diffFn 이 없는 step 포함: %s", d.category)
		}
	}
	// diff 전용 step (custom_rules 등) 때문에 diff 쪽이 더 많아야 함
	if len(diffDefs) <= len(runDefs) {
		t.Errorf("diff step(%d)이 run step(%d)보다 많아야 합니다", len(diffDefs), len(runDefs))
	}
}

func TestStepDefsFor_OnlySingleCategory(t *testing.T) {
	defs, err := stepDefsFor(true, []string{"comment_language"})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].category != "comment_language" {
		t.Errorf("comment_language step 1개만 남아야 합니다: %+v", defs)
	}
}

func TestStepDefsFor_OnlyMultipleCategories_KeepsRegistryOrder(t *testing.T) {
	// 입력 순서와 무관하게 레지스트리 순서(binary → lint)를 유지해야 함
	defs, err := stepDefsFor(false, []string{"lint", "binary"})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 || defs[0].category != "binary" || defs[1].category != "lint" {
		var cats []string
		for _, d := range defs {
			cats = append(cats, d.category)
		}
		t.Errorf("binary, lint 순서를 기대했습니다: %v", cats)
	}
}

func TestStepDefsFor_InvalidCategory_Error(t *testing.T) {
	_, err := stepDefsFor(true, []string{"no_such_check"})
	if err == nil {
		t.Fatal("잘못된 카테고리는 에러가 나야 합니다")
	}
	if !strings.Contains(err.Error(), "no_such_check") {
		t.Errorf("에러에 잘못된 카테고리 이름이 포함되어야 합니다: %v", err)
	}
	// 유효 카테고리 목록이 포함되어야 함
	if !strings.Contains(err.Error(), "comment_language") || !strings.Contains(err.Error(), "binary") {
		t.Errorf("에러에 유효 카테고리 목록이 포함되어야 합니다: %v", err)
	}
}

func TestStepDefsFor_DiffOnlyCategory_OnRun_Error(t *testing.T) {
	// custom_rules 등은 diff 전용 (runFn == nil) → run --only 에서는 에러
	for _, cat := range []string{"custom_rules", "protected_paths", "append_only"} {
		_, err := stepDefsFor(false, []string{cat})
		if err == nil {
			t.Errorf("diff 전용 카테고리 %s 는 run 에서 에러가 나야 합니다", cat)
			continue
		}
		if !strings.Contains(err.Error(), cat) {
			t.Errorf("에러에 카테고리 이름 %s 이 포함되어야 합니다: %v", cat, err)
		}
	}
	// run 의 유효 목록에는 diff 전용 카테고리가 포함되면 안 됨
	_, err := stepDefsFor(false, []string{"bogus"})
	if err == nil {
		t.Fatal("잘못된 카테고리는 에러가 나야 합니다")
	}
	if strings.Contains(err.Error(), "custom_rules") {
		t.Errorf("run 유효 목록에 diff 전용 카테고리가 포함되면 안 됩니다: %v", err)
	}
	// diff 에서는 같은 카테고리가 유효해야 함
	if _, err := stepDefsFor(true, []string{"custom_rules"}); err != nil {
		t.Errorf("custom_rules 는 diff 에서 유효해야 합니다: %v", err)
	}
}

func TestStepDefsFor_TrimsSpaces(t *testing.T) {
	defs, err := stepDefsFor(true, []string{" comment_language ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].category != "comment_language" {
		t.Errorf("공백을 제거한 카테고리가 적용되어야 합니다: %+v", defs)
	}
}

// ---- cfgWithOnlyEnabled --------------------------------------------------------

func TestCfgWithOnlyEnabled_ForcesDisabledCheckOn(t *testing.T) {
	off := false
	cfg := &config.Config{}
	cfg.CommentLanguage.Enabled = &off
	cfg.BinaryFile.Enabled = &off

	defs, err := stepDefsFor(true, []string{"comment_language"})
	if err != nil {
		t.Fatal(err)
	}
	overridden := cfgWithOnlyEnabled(cfg, defs)

	if !overridden.CommentLanguage.IsEnabled() {
		t.Error("--only 로 선택된 comment_language 는 enabled=false 여도 강제로 켜져야 합니다")
	}
	if overridden.BinaryFile.IsEnabled() {
		t.Error("선택되지 않은 binary 검사의 enabled=false 설정은 유지되어야 합니다")
	}
	// 원본 cfg 는 변경되지 않아야 함
	if cfg.CommentLanguage.IsEnabled() {
		t.Error("원본 cfg 가 수정되면 안 됩니다")
	}
}

func TestCfgWithOnlyEnabled_BoolFields(t *testing.T) {
	cfg := &config.Config{}
	cfg.ProtectedPaths.Enabled = false
	cfg.ProtectedPaths.Paths = []string{"legacy/**"}
	cfg.AppendOnly.Enabled = false
	cfg.AppendOnly.Paths = []string{"migrations/**"}

	defs, err := stepDefsFor(true, []string{"protected_paths", "append_only"})
	if err != nil {
		t.Fatal(err)
	}
	overridden := cfgWithOnlyEnabled(cfg, defs)

	if !overridden.ProtectedPaths.IsEnabled() {
		t.Error("protected_paths 가 강제로 켜져야 합니다")
	}
	if !overridden.AppendOnly.IsEnabled() {
		t.Error("append_only 가 강제로 켜져야 합니다")
	}
	if cfg.ProtectedPaths.Enabled || cfg.AppendOnly.Enabled {
		t.Error("원본 cfg 가 수정되면 안 됩니다")
	}
}

func TestCfgWithOnlyEnabled_Encoding_ForcesRequireUTF8(t *testing.T) {
	off := false
	cfg := &config.Config{}
	cfg.Encoding.Enabled = &off
	cfg.Encoding.RequireUTF8 = &off

	defs, err := stepDefsFor(false, []string{"encoding"})
	if err != nil {
		t.Fatal(err)
	}
	overridden := cfgWithOnlyEnabled(cfg, defs)

	if !overridden.Encoding.IsEnabled() || !overridden.Encoding.IsRequireUTF8() {
		t.Error("--only encoding 은 enabled 와 require_utf8 둘 다 강제로 켜야 합니다")
	}
}

// ---- run/diff 커맨드 통합 ------------------------------------------------------

// stageFile: 파일을 기록하고 git add 로 스테이지.
func stageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, name), content)
	c := exec.Command("git", "add", name)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git add %s: %v\n%s", name, err, out)
	}
}

func TestDiffCmd_Only_OverridesDisabledConfig(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	// 설정에서 comment_language 비활성화
	writeTestFile(t, cfgPath, "comment_language:\n  enabled: false\n")
	setConfigFile(t, cfgPath)
	setGlobalQuiet(t, true)

	// 영어 주석이 포함된 go 파일을 스테이지 (기본 required locale: korean → 위반)
	stageFile(t, dir, "main.go",
		"package main\n\n// this comment is definitely written in english language\nfunc main() {}\n")

	// --only 없이: 설정대로 비활성화 → 통과
	setDiffOnly(t, nil)
	if err := runCmdE(diffCmd, nil); err != nil {
		t.Fatalf("comment_language 비활성화 상태에서는 통과해야 합니다: %v", err)
	}

	// --only comment_language: 설정의 enabled=false 를 덮어쓰고 강제 실행 → 위반
	setDiffOnly(t, []string{"comment_language"})
	var err error
	stderr := captureStderr(t, func() {
		err = runCmdE(diffCmd, nil)
	})
	if !errors.Is(err, errSilentExit) {
		t.Errorf("--only comment_language 는 위반을 보고해야 합니다: %v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "main.go") {
		t.Errorf("위반 메시지에 파일 이름이 포함되어야 합니다:\n%s", stderr)
	}
}

func TestDiffCmd_Only_RunsOnlySelectedStep(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	setConfigFile(t, filepath.Join(dir, ".commit-checker.yml")) // 파일 없음 → 기본 설정
	setGlobalQuiet(t, true)

	// 한국어 주석은 통과하지만, lint 위반이 있는 JSON 도 함께 스테이지
	stageFile(t, dir, "ok.go", "package ok\n\n// 한국어 주석은 통과합니다\n")
	stageFile(t, dir, "broken.json", "{ broken json !!\n")

	// --only comment_language: JSON lint 는 실행되지 않으므로 통과해야 함
	setDiffOnly(t, []string{"comment_language"})
	if err := runCmdE(diffCmd, nil); err != nil {
		t.Errorf("comment_language 만 실행하면 JSON lint 위반은 무시되어야 합니다: %v", err)
	}

	// --only lint: JSON lint 위반이 보고되어야 함
	setDiffOnly(t, []string{"lint"})
	var err error
	stderr := captureStderr(t, func() {
		err = runCmdE(diffCmd, nil)
	})
	if !errors.Is(err, errSilentExit) {
		t.Errorf("--only lint 는 JSON 위반을 보고해야 합니다: %v", err)
	}
	if !strings.Contains(stderr, "broken.json") {
		t.Errorf("위반 메시지에 broken.json 이 포함되어야 합니다:\n%s", stderr)
	}
}

func TestDiffCmd_Only_InvalidCategory_Error(t *testing.T) {
	isolateHome(t)
	setGlobalQuiet(t, true)
	setDiffOnly(t, []string{"bogus"})

	err := runCmdE(diffCmd, nil)
	if err == nil {
		t.Fatal("잘못된 --only 카테고리는 에러가 나야 합니다")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("에러에 잘못된 카테고리 이름이 포함되어야 합니다: %v", err)
	}
}

func TestRunCmd_Only_DiffOnlyCategory_Error(t *testing.T) {
	isolateHome(t)
	setGlobalQuiet(t, true)
	setRunOnly(t, []string{"custom_rules"})

	err := runCmdE(runCmd, nil)
	if err == nil {
		t.Fatal("diff 전용 카테고리를 run --only 에 주면 에러가 나야 합니다")
	}
	if !strings.Contains(err.Error(), "custom_rules") {
		t.Errorf("에러에 카테고리 이름이 포함되어야 합니다: %v", err)
	}
}

func TestRunCmd_Only_OverridesDisabledConfig(t *testing.T) {
	isolateHome(t)
	dir := newTestGitRepo(t)
	cfgPath := filepath.Join(dir, ".commit-checker.yml")
	writeTestFile(t, cfgPath, "lint:\n  enabled: false\n")
	setConfigFile(t, cfgPath)
	setGlobalQuiet(t, true)

	// lint 위반 JSON 을 커밋하여 추적 파일로 만듦
	stageFile(t, dir, "broken.json", "{ broken json !!\n")
	gitRun(t, dir, "commit", "-m", "chore: broken json 추가")

	// --only 없이: lint 비활성화 → 통과
	setRunOnly(t, nil)
	if err := runCmdE(runCmd, nil); err != nil {
		t.Fatalf("lint 비활성화 상태에서는 통과해야 합니다: %v", err)
	}

	// --only lint: 강제 실행 → 위반
	setRunOnly(t, []string{"lint"})
	var err error
	stderr := captureStderr(t, func() {
		err = runCmdE(runCmd, nil)
	})
	if !errors.Is(err, errSilentExit) {
		t.Errorf("--only lint 는 위반을 보고해야 합니다: %v", err)
	}
	if !strings.Contains(stderr, "broken.json") {
		t.Errorf("위반 메시지에 broken.json 이 포함되어야 합니다:\n%s", stderr)
	}
}
