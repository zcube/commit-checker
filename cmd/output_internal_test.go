package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zcube/commit-checker/internal/config"
	"github.com/zcube/commit-checker/internal/i18n"
	"github.com/zcube/commit-checker/internal/progress"
)

// setGlobalQuiet: globalQuiet 플래그를 설정하고 테스트 종료 시 복원.
func setGlobalQuiet(t *testing.T, v bool) {
	t.Helper()
	orig := globalQuiet
	globalQuiet = v
	t.Cleanup(func() { globalQuiet = orig })
}

// makeTestStep: 지정한 위반 목록을 반환하는 검사 step 을 생성.
func makeTestStep(name string, errs []string) progress.Step {
	return makeTestStepCat(name, "test_check", errs)
}

// makeTestStepCat: 카테고리를 지정해 위반 목록을 반환하는 검사 step 을 생성.
func makeTestStepCat(name, category string, errs []string) progress.Step {
	return progress.Step{
		Name:     name,
		Category: category,
		Fn: func(ctx context.Context) ([]string, error) {
			return errs, nil
		},
	}
}

func TestRunStepsAndReport_Text_Violations_ReturnsSentinel(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStep("테스트 검사", []string{"file.go:1: 위반 발견"})}

	var err error
	stderr := captureStderr(t, func() {
		err = runStepsAndReport(context.Background(), steps, "text", true)
	})

	if !errors.Is(err, errSilentExit) {
		t.Errorf("위반이 있으면 errSilentExit 를 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stderr, "file.go:1: 위반 발견") {
		t.Errorf("위반 메시지가 stderr 에 출력되어야 합니다:\n%s", stderr)
	}
}

func TestRunStepsAndReport_Text_NoViolations_ReturnsNil(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStep("테스트 검사", nil)}

	var err error
	stderr := captureStderr(t, func() {
		err = runStepsAndReport(context.Background(), steps, "text", true)
	})

	if err != nil {
		t.Errorf("위반이 없으면 nil 을 반환해야 합니다: %v", err)
	}
	if stderr != "" {
		t.Errorf("위반이 없으면 stderr 출력이 없어야 합니다:\n%s", stderr)
	}
}

func TestRunStepsAndReport_JSON_Violations_ReturnsSentinel(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStep("테스트 검사", []string{"file.go:1: 위반 발견"})}

	var err error
	stdout := captureStdout(t, func() {
		err = runStepsAndReport(context.Background(), steps, "json", true)
	})

	if !errors.Is(err, errSilentExit) {
		t.Errorf("JSON 출력에서도 위반이 있으면 errSilentExit 를 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stdout, `"fail"`) {
		t.Errorf("JSON 결과가 stdout 에 출력되어야 합니다:\n%s", stdout)
	}
}

func TestRunStepsAndReport_JSON_NoViolations_ReturnsNil(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStep("테스트 검사", nil)}

	var err error
	stdout := captureStdout(t, func() {
		err = runStepsAndReport(context.Background(), steps, "json", true)
	})

	if err != nil {
		t.Errorf("위반이 없으면 nil 을 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stdout, `"pass"`) {
		t.Errorf("JSON 결과가 stdout 에 출력되어야 합니다:\n%s", stdout)
	}
}

// --- 개선 가이드 출력 ---

func TestRunStepsAndReport_Text_GuideOn_PrintsFailedCategories(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{
		makeTestStepCat("바이너리 검사", "binary", []string{"a.bin: 바이너리 감지"}),
		makeTestStepCat("lint 검사", "lint", []string{"b.json:1: 구문 오류"}),
		makeTestStepCat("인코딩 검사", "encoding", nil), // 통과한 step 은 가이드 제외
	}

	stderr := captureStderr(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "text", true)
	})

	header := i18n.T("guide.header", nil)
	if !strings.Contains(stderr, header) {
		t.Errorf("가이드 헤더가 출력되어야 합니다:\n%s", stderr)
	}
	if !strings.Contains(stderr, "[binary] "+guideText("binary")) {
		t.Errorf("binary 가이드가 출력되어야 합니다:\n%s", stderr)
	}
	if !strings.Contains(stderr, "[lint] "+guideText("lint")) {
		t.Errorf("lint 가이드가 출력되어야 합니다:\n%s", stderr)
	}
	if strings.Contains(stderr, "[encoding]") {
		t.Errorf("통과한 카테고리의 가이드는 출력되면 안 됩니다:\n%s", stderr)
	}
	// 가이드는 위반 목록·요약 줄보다 뒤에 위치
	if idx := strings.Index(stderr, header); idx < strings.Index(stderr, "a.bin: 바이너리 감지") {
		t.Errorf("가이드는 위반 목록 뒤에 출력되어야 합니다:\n%s", stderr)
	}
}

func TestRunStepsAndReport_Text_GuideOff_NoGuide(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStepCat("바이너리 검사", "binary", []string{"a.bin: 바이너리 감지"})}

	stderr := captureStderr(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "text", false)
	})

	if strings.Contains(stderr, i18n.T("guide.header", nil)) {
		t.Errorf("가이드 비활성 시 가이드가 출력되면 안 됩니다:\n%s", stderr)
	}
}

func TestRunStepsAndReport_Text_Guide_NoDuplicates(t *testing.T) {
	setGlobalQuiet(t, true)
	// 같은 카테고리의 step 이 여러 번 실패해도 가이드는 1회만
	steps := []progress.Step{
		makeTestStepCat("lint 검사 1", "lint", []string{"a.json:1: 구문 오류"}),
		makeTestStepCat("lint 검사 2", "lint", []string{"b.json:2: 구문 오류"}),
	}

	stderr := captureStderr(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "text", true)
	})

	if n := strings.Count(stderr, "[lint] "); n != 1 {
		t.Errorf("같은 카테고리 가이드는 1회만 출력되어야 합니다 (출력 %d회):\n%s", n, stderr)
	}
}

func TestRunStepsAndReport_Text_Guide_UnknownCategory_Skipped(t *testing.T) {
	setGlobalQuiet(t, true)
	// guide.<category> i18n 키가 없는 카테고리는 가이드 전체를 생략
	steps := []progress.Step{makeTestStepCat("테스트 검사", "test_check", []string{"file.go:1: 위반"})}

	stderr := captureStderr(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "text", true)
	})

	if strings.Contains(stderr, i18n.T("guide.header", nil)) {
		t.Errorf("가이드 키가 없는 카테고리만 실패하면 가이드를 출력하지 않아야 합니다:\n%s", stderr)
	}
}

func TestRunStepsAndReport_JSON_GuideOn_IncludesGuides(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStepCat("바이너리 검사", "binary", []string{"a.bin: 바이너리 감지"})}

	stdout := captureStdout(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "json", true)
	})

	if !strings.Contains(stdout, `"guides"`) {
		t.Errorf("가이드 활성 시 JSON 에 guides 필드가 포함되어야 합니다:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"binary"`) {
		t.Errorf("guides 에 실패 카테고리 키가 포함되어야 합니다:\n%s", stdout)
	}
}

func TestRunStepsAndReport_JSON_GuideOff_OmitsGuides(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStepCat("바이너리 검사", "binary", []string{"a.bin: 바이너리 감지"})}

	stdout := captureStdout(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "json", false)
	})

	if strings.Contains(stdout, `"guides"`) {
		t.Errorf("가이드 비활성 시 JSON 에 guides 필드가 없어야 합니다:\n%s", stdout)
	}
}

func TestRunStepsAndReport_JSON_NoViolations_OmitsGuides(t *testing.T) {
	setGlobalQuiet(t, true)
	steps := []progress.Step{makeTestStepCat("바이너리 검사", "binary", nil)}

	stdout := captureStdout(t, func() {
		_ = runStepsAndReport(context.Background(), steps, "json", true)
	})

	if strings.Contains(stdout, `"guides"`) {
		t.Errorf("위반이 없으면 JSON 에 guides 필드가 없어야 합니다:\n%s", stdout)
	}
}

// --- failedGuides ---

func TestFailedGuides_OrderAndDedup(t *testing.T) {
	steps := []progress.StepResult{
		{Name: "lint", Category: "lint", Errors: []string{"e1"}},
		{Name: "binary", Category: "binary", Errors: []string{"e2"}},
		{Name: "lint2", Category: "lint", Errors: []string{"e3"}},
		{Name: "encoding", Category: "encoding"}, // 위반 없음
		{Name: "unknown", Category: "no_such_category", Errors: []string{"e4"}},
	}

	cats, guides := failedGuides(steps)

	want := []string{"lint", "binary"}
	if len(cats) != len(want) {
		t.Fatalf("카테고리 수 = %d, want %d (%v)", len(cats), len(want), cats)
	}
	for i, c := range want {
		if cats[i] != c {
			t.Errorf("cats[%d] = %q, want %q (step 순서 유지)", i, cats[i], c)
		}
		if guides[c] == "" {
			t.Errorf("guides[%q] 가 비어 있으면 안 됩니다", c)
		}
	}
}

func TestGuideEnabled_NoGuideFlag(t *testing.T) {
	cfg := &config.Config{} // 기본 설정 (guide.enabled 미지정 → true)

	if !guideEnabled(cfg) {
		t.Error("기본 설정에서는 가이드가 활성화되어야 합니다")
	}

	orig := globalNoGuide
	globalNoGuide = true
	t.Cleanup(func() { globalNoGuide = orig })

	if guideEnabled(cfg) {
		t.Error("--no-guide 플래그가 켜지면 설정과 무관하게 가이드가 비활성화되어야 합니다")
	}
}

func TestGuideEnabled_ConfigDisabled(t *testing.T) {
	disabled := false
	cfg := &config.Config{Guide: config.GuideConfig{Enabled: &disabled}}

	if guideEnabled(cfg) {
		t.Error("guide.enabled: false 설정이면 가이드가 비활성화되어야 합니다")
	}
}
