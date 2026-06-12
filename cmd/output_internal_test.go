package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	return progress.Step{
		Name:     name,
		Category: "test_check",
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
		err = runStepsAndReport(context.Background(), steps, "text")
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
		err = runStepsAndReport(context.Background(), steps, "text")
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
		err = runStepsAndReport(context.Background(), steps, "json")
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
		err = runStepsAndReport(context.Background(), steps, "json")
	})

	if err != nil {
		t.Errorf("위반이 없으면 nil 을 반환해야 합니다: %v", err)
	}
	if !strings.Contains(stdout, `"pass"`) {
		t.Errorf("JSON 결과가 stdout 에 출력되어야 합니다:\n%s", stdout)
	}
}
