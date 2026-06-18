package progress

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// makeStep: 지정한 위반 목록/에러를 반환하는 단계를 생성하고 실행 여부를 ran 에 기록.
func makeStep(name, category string, errs []string, err error, ran *bool) Step {
	return Step{
		Name:     name,
		Category: category,
		Fn: func(_ context.Context) ([]string, error) {
			if ran != nil {
				*ran = true
			}
			return errs, err
		},
	}
}

// --- runPlain / RunWithProgress (비TTY 경로) ---

func TestRunPlain_정상실행(t *testing.T) {
	var buf bytes.Buffer
	steps := []Step{
		makeStep("인코딩 검사", "encoding", nil, nil, nil),
		makeStep("이모지 검사", "emoji", []string{"위반1", "위반2"}, nil, nil),
	}

	result, err := runPlain(context.Background(), steps, &buf)
	if err != nil {
		t.Fatalf("runPlain() error = %v, want nil", err)
	}
	if len(result.AllErrors) != 2 {
		t.Errorf("AllErrors 길이 = %d, want 2", len(result.AllErrors))
	}
	if len(result.Steps) != 2 {
		t.Fatalf("Steps 길이 = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Name != "인코딩 검사" || result.Steps[0].Category != "encoding" {
		t.Errorf("Steps[0] = %+v, want Name=인코딩 검사 Category=encoding", result.Steps[0])
	}
	if len(result.Steps[1].Errors) != 2 {
		t.Errorf("Steps[1].Errors 길이 = %d, want 2", len(result.Steps[1].Errors))
	}
	for _, s := range result.Steps {
		if s.Failed {
			t.Errorf("Steps[%s].Failed = true, want false", s.Name)
		}
	}
	// 진행 표시가 출력되는지 확인
	out := buf.String()
	if !strings.Contains(out, "인코딩 검사") || !strings.Contains(out, "이모지 검사") {
		t.Errorf("출력에 단계 이름이 없음: %q", out)
	}
}

func TestRunPlain_치명적오류시중단(t *testing.T) {
	fatalErr := errors.New("치명적 오류")
	var thirdRan bool
	steps := []Step{
		makeStep("단계1", "step1", []string{"위반1"}, nil, nil),
		makeStep("단계2", "step2", nil, fatalErr, nil),
		makeStep("단계3", "step3", nil, nil, &thirdRan),
	}

	result, err := runPlain(context.Background(), steps, io.Discard)
	if !errors.Is(err, fatalErr) {
		t.Fatalf("runPlain() error = %v, want %v", err, fatalErr)
	}
	if thirdRan {
		t.Error("치명적 오류 후 단계3이 실행됨, 중단되어야 함")
	}
	if !result.Steps[1].Failed {
		t.Error("Steps[1].Failed = false, want true")
	}
	// 실패 전까지의 위반만 수집되어야 함
	if len(result.AllErrors) != 1 {
		t.Errorf("AllErrors 길이 = %d, want 1", len(result.AllErrors))
	}
}

func TestRunPlain_컨텍스트취소시중단(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var secondRan bool
	steps := []Step{
		{
			Name:     "단계1",
			Category: "step1",
			Fn: func(_ context.Context) ([]string, error) {
				// 단계 실행 중 취소가 발생한 상황을 재현
				cancel()
				return nil, nil
			},
		},
		makeStep("단계2", "step2", nil, nil, &secondRan),
	}

	_, err := runPlain(ctx, steps, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runPlain() error = %v, want context.Canceled", err)
	}
	if secondRan {
		t.Error("취소 후 단계2가 실행됨, 건너뛰어야 함")
	}
}

func TestRunPlain_사전취소시즉시중단(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 실행 전에 이미 취소된 상태

	var ran bool
	steps := []Step{makeStep("단계1", "step1", nil, nil, &ran)}

	_, err := runPlain(ctx, steps, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runPlain() error = %v, want context.Canceled", err)
	}
	if ran {
		t.Error("취소된 컨텍스트에서 단계가 실행됨")
	}
}

func TestRunWithProgress_Quiet(t *testing.T) {
	steps := []Step{
		makeStep("단계1", "step1", []string{"위반1"}, nil, nil),
		makeStep("단계2", "step2", nil, nil, nil),
	}

	result, err := RunWithProgress(context.Background(), steps, Options{Quiet: true})
	if err != nil {
		t.Fatalf("RunWithProgress() error = %v, want nil", err)
	}
	if len(result.AllErrors) != 1 {
		t.Errorf("AllErrors 길이 = %d, want 1", len(result.AllErrors))
	}
	if len(result.Steps) != 2 {
		t.Errorf("Steps 길이 = %d, want 2", len(result.Steps))
	}
}

func TestRunWithProgress_비TTY(t *testing.T) {
	// go test 는 stderr 를 파이프로 연결하므로 비TTY(plain) 경로를 탄다
	steps := []Step{
		makeStep("단계1", "step1", []string{"위반1", "위반2"}, nil, nil),
	}

	result, err := RunWithProgress(context.Background(), steps, Options{})
	if err != nil {
		t.Fatalf("RunWithProgress() error = %v, want nil", err)
	}
	if len(result.AllErrors) != 2 {
		t.Errorf("AllErrors 길이 = %d, want 2", len(result.AllErrors))
	}
}

func TestRunWithProgress_빈단계목록(t *testing.T) {
	result, err := RunWithProgress(context.Background(), nil, Options{Quiet: true})
	if err != nil {
		t.Fatalf("RunWithProgress() error = %v, want nil", err)
	}
	if len(result.AllErrors) != 0 || len(result.Steps) != 0 {
		t.Errorf("빈 단계 목록 결과 = %+v, want 빈 결과", result)
	}
}

// --- Summary ---

func TestSummary_위반없음(t *testing.T) {
	steps := []StepResult{
		{Name: "단계1", Category: "step1"},
		{Name: "단계2", Category: "step2"},
	}
	total, checks := Summary(steps)
	if total != 0 || checks != "" {
		t.Errorf("Summary() = (%d, %q), want (0, 빈 문자열)", total, checks)
	}
}

func TestSummary_위반다건(t *testing.T) {
	steps := []StepResult{
		{Name: "인코딩 검사", Category: "encoding", Errors: []string{"위반1", "위반2"}},
		{Name: "이모지 검사", Category: "emoji", Errors: []string{"위반3"}},
		{Name: "통과 단계", Category: "pass"},
	}
	total, checks := Summary(steps)
	if total != 3 || checks != "encoding(2), emoji(1)" {
		t.Errorf("Summary() = (%d, %q), want (3, %q)", total, checks, "encoding(2), emoji(1)")
	}
}

func TestSummary_카테고리없으면이름사용(t *testing.T) {
	steps := []StepResult{
		{Name: "이름만 있는 단계", Errors: []string{"위반1"}},
	}
	_, checks := Summary(steps)
	if !strings.Contains(checks, "이름만 있는 단계(1)") {
		t.Errorf("Summary() checks = %q, want 단계 이름 포함", checks)
	}
}

// --- FormatJSON ---

func TestFormatJSON_위반없음(t *testing.T) {
	result := RunResult{
		Steps: []StepResult{{Name: "단계1", Category: "step1"}},
	}
	data, err := FormatJSON(result, nil)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	var out struct {
		Status     string                   `json:"status"`
		Violations []map[string]interface{} `json:"violations"`
		Summary    struct {
			Total   int            `json:"total"`
			ByCheck map[string]int `json:"by_check"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}
	if out.Status != "pass" {
		t.Errorf("status = %q, want pass", out.Status)
	}
	if out.Violations == nil {
		t.Error("violations 가 null, want 빈 배열")
	}
	if out.Summary.Total != 0 {
		t.Errorf("summary.total = %d, want 0", out.Summary.Total)
	}
}

func TestFormatJSON_위반있음(t *testing.T) {
	result := RunResult{
		Steps: []StepResult{
			{Name: "인코딩 검사", Category: "encoding", Errors: []string{"위반1", "위반2"}},
			{Name: "이모지 검사", Category: "emoji", Errors: []string{"위반3"}},
		},
	}
	data, err := FormatJSON(result, nil)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	var out struct {
		Status     string `json:"status"`
		Violations []struct {
			Message string `json:"message"`
			Check   string `json:"check"`
		} `json:"violations"`
		Summary struct {
			Total   int            `json:"total"`
			ByCheck map[string]int `json:"by_check"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}
	if out.Status != "fail" {
		t.Errorf("status = %q, want fail", out.Status)
	}
	if len(out.Violations) != 3 {
		t.Fatalf("violations 길이 = %d, want 3", len(out.Violations))
	}
	if out.Violations[0].Check != "encoding" || out.Violations[0].Message != "위반1" {
		t.Errorf("violations[0] = %+v, want check=encoding message=위반1", out.Violations[0])
	}
	if out.Summary.Total != 3 {
		t.Errorf("summary.total = %d, want 3", out.Summary.Total)
	}
	if out.Summary.ByCheck["encoding"] != 2 || out.Summary.ByCheck["emoji"] != 1 {
		t.Errorf("summary.by_check = %v, want encoding:2 emoji:1", out.Summary.ByCheck)
	}
}

// --- TUI model (bubbletea Program 없이 Update/View 직접 호출) ---

// newTestModel: 테스트용 model 생성 헬퍼.
func newTestModel(t *testing.T, steps []Step, opts Options) (model, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return newModel(ctx, cancel, steps, opts), cancel
}

func TestModel_단계완료후다음단계실행(t *testing.T) {
	steps := []Step{
		makeStep("단계1", "step1", []string{"위반1"}, nil, nil),
		makeStep("단계2", "step2", nil, nil, nil),
	}
	m, _ := newTestModel(t, steps, Options{NoColor: true})

	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init() = nil, want 첫 단계 실행 커맨드")
	}

	// 단계1 완료 메시지 처리
	next, cmd := m.Update(stepDoneMsg{idx: 0, errs: []string{"위반1"}})
	m = next.(model)
	if m.current != 1 {
		t.Errorf("current = %d, want 1", m.current)
	}
	if len(m.allErrs) != 1 {
		t.Errorf("allErrs 길이 = %d, want 1", len(m.allErrs))
	}
	if m.done {
		t.Error("done = true, want false (단계2 남음)")
	}
	if cmd == nil {
		t.Fatal("다음 단계 실행 커맨드가 nil")
	}

	// 반환된 커맨드를 직접 실행하면 단계2의 완료 메시지가 나와야 함
	msg, ok := cmd().(stepDoneMsg)
	if !ok || msg.idx != 1 {
		t.Fatalf("cmd() = %+v, want stepDoneMsg{idx:1}", msg)
	}

	// 마지막 단계 완료 처리
	next, _ = m.Update(msg)
	m = next.(model)
	if !m.done {
		t.Error("done = false, want true (모든 단계 완료)")
	}
	if m.fatalErr != nil {
		t.Errorf("fatalErr = %v, want nil", m.fatalErr)
	}
}

func TestModel_치명적오류시종료(t *testing.T) {
	fatalErr := errors.New("치명적 오류")
	steps := []Step{
		makeStep("단계1", "step1", nil, fatalErr, nil),
		makeStep("단계2", "step2", nil, nil, nil),
	}
	m, _ := newTestModel(t, steps, Options{NoColor: true})

	next, _ := m.Update(stepDoneMsg{idx: 0, err: fatalErr})
	m = next.(model)
	if !errors.Is(m.fatalErr, fatalErr) {
		t.Errorf("fatalErr = %v, want %v", m.fatalErr, fatalErr)
	}
	if !m.done {
		t.Error("done = false, want true")
	}
	if !m.results[0].failed {
		t.Error("results[0].failed = false, want true")
	}
}

func TestModel_CtrlC시컨텍스트취소(t *testing.T) {
	steps := []Step{makeStep("단계1", "step1", nil, nil, nil)}
	m, _ := newTestModel(t, steps, Options{NoColor: true})

	next, _ := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'c'})
	m = next.(model)
	if m.fatalErr == nil {
		t.Error("fatalErr = nil, want interrupted 오류")
	}
	if m.ctx.Err() == nil {
		t.Error("Ctrl-C 후 컨텍스트가 취소되지 않음")
	}
}

func TestModel_취소후다음단계건너뜀(t *testing.T) {
	steps := []Step{
		makeStep("단계1", "step1", nil, nil, nil),
		makeStep("단계2", "step2", nil, nil, nil),
	}
	m, cancel := newTestModel(t, steps, Options{NoColor: true})

	// 단계1 완료 직전에 컨텍스트 취소
	cancel()
	next, _ := m.Update(stepDoneMsg{idx: 0})
	m = next.(model)
	if !errors.Is(m.fatalErr, context.Canceled) {
		t.Errorf("fatalErr = %v, want context.Canceled", m.fatalErr)
	}
	if !m.done {
		t.Error("done = false, want true")
	}
}

func TestModel_View(t *testing.T) {
	steps := []Step{
		makeStep("통과 단계", "pass", nil, nil, nil),
		makeStep("위반 단계", "warn", []string{"위반1"}, nil, nil),
		makeStep("실패 단계", "fail", nil, fmt.Errorf("오류"), nil),
	}
	m, _ := newTestModel(t, steps, Options{NoColor: true})

	// 실행 전: 현재 단계가 스피너와 함께 표시
	view := m.View().Content
	if !strings.Contains(view, "통과 단계") {
		t.Errorf("View() = %q, want 현재 단계 이름 포함", view)
	}

	// 단계 결과를 채우고 완료 상태로 만든 뒤 출력 확인
	next, _ := m.Update(stepDoneMsg{idx: 0})
	m = next.(model)
	next, _ = m.Update(stepDoneMsg{idx: 1, errs: []string{"위반1"}})
	m = next.(model)
	next, _ = m.Update(stepDoneMsg{idx: 2, err: fmt.Errorf("오류")})
	m = next.(model)

	view = m.View().Content
	for _, name := range []string{"통과 단계", "위반 단계", "실패 단계"} {
		if !strings.Contains(view, name) {
			t.Errorf("View() 에 %q 이(가) 없음: %q", name, view)
		}
	}
	if !strings.Contains(view, "(1 issues)") {
		t.Errorf("View() 에 위반 건수 표시가 없음: %q", view)
	}
}

func TestFormatJSON_가이드포함(t *testing.T) {
	result := RunResult{
		Steps: []StepResult{
			{Name: "바이너리 검사", Category: "binary", Errors: []string{"위반1"}},
		},
	}
	guides := map[string]string{"binary": "git rm --cached 로 제거하세요"}
	data, err := FormatJSON(result, guides)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	var out struct {
		Guides map[string]string `json:"guides"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("JSON 파싱 실패: %v", err)
	}
	if out.Guides["binary"] != guides["binary"] {
		t.Errorf("guides[binary] = %q, want %q", out.Guides["binary"], guides["binary"])
	}
}

func TestFormatJSON_가이드없으면필드생략(t *testing.T) {
	result := RunResult{
		Steps: []StepResult{
			{Name: "바이너리 검사", Category: "binary", Errors: []string{"위반1"}},
		},
	}
	data, err := FormatJSON(result, nil)
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}
	if strings.Contains(string(data), `"guides"`) {
		t.Errorf("guides 가 nil 이면 필드를 생략해야 합니다 (기존 소비자 호환):\n%s", data)
	}
}
