package progress

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Step: 실행할 검사 단계 정보.
type Step struct {
	Name     string
	Category string // 기계가 읽을 수 있는 카테고리 (JSON 출력 등)
	Fn       func() ([]string, error)
}

// StepResult: 단계 실행 결과.
type StepResult struct {
	Name     string
	Category string
	Errors   []string
	Failed   bool // 치명적 오류 발생 여부
}

// RunResult: RunWithProgress 반환값.
type RunResult struct {
	AllErrors []string
	Steps     []StepResult
}

// Options: RunWithProgress 옵션.
type Options struct {
	Quiet   bool
	NoColor bool
}

// RunWithProgress: 검사 단계를 순차 실행하고 결과를 반환.
func RunWithProgress(steps []Step, opts Options) (RunResult, error) {
	if opts.Quiet {
		return runPlainSilent(steps)
	}
	if !isTTY() {
		return runPlain(steps, os.Stderr)
	}
	return runTUI(steps, opts)
}

func isTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runPlainSilent: 출력 없이 조용히 실행.
func runPlainSilent(steps []Step) (RunResult, error) {
	return runPlain(steps, io.Discard)
}

// runPlain: TTY가 아닐 때 단순 텍스트로 진행 표시.
func runPlain(steps []Step, w io.Writer) (RunResult, error) {
	result := RunResult{
		Steps: make([]StepResult, len(steps)),
	}
	for i, s := range steps {
		_, _ = fmt.Fprintf(w, "  %s ...\n", s.Name)
		errs, err := s.Fn()
		result.Steps[i] = StepResult{
			Name:     s.Name,
			Category: s.Category,
			Errors:   errs,
		}
		if err != nil {
			result.Steps[i].Failed = true
			return result, err
		}
		result.AllErrors = append(result.AllErrors, errs...)
	}
	return result, nil
}

// SummaryLine: 위반 건수 요약 줄 반환.
func SummaryLine(steps []StepResult) string {
	total := 0
	var parts []string
	for _, s := range steps {
		n := len(s.Errors)
		if n > 0 {
			total += n
			cat := s.Category
			if cat == "" {
				cat = s.Name
			}
			parts = append(parts, fmt.Sprintf("%s(%d)", cat, n))
		}
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("✗ %d건 위반: %s", total, strings.Join(parts, ", "))
}

// jsonViolation: JSON 출력 위반 항목.
type jsonViolation struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
	Check   string `json:"check"`
}

// jsonSummary: JSON 출력 요약.
type jsonSummary struct {
	Total   int            `json:"total"`
	ByCheck map[string]int `json:"by_check"`
}

// jsonOutput: JSON 출력 최상위 구조체.
type jsonOutput struct {
	Status     string          `json:"status"`
	Violations []jsonViolation `json:"violations"`
	Summary    jsonSummary     `json:"summary"`
}

// FormatJSON: 결과를 JSON으로 직렬화.
func FormatJSON(result RunResult) ([]byte, error) {
	out := jsonOutput{
		Status:     "pass",
		Violations: []jsonViolation{},
		Summary:    jsonSummary{ByCheck: map[string]int{}},
	}
	for _, s := range result.Steps {
		for _, msg := range s.Errors {
			out.Violations = append(out.Violations, jsonViolation{
				Message: msg,
				Check:   s.Category,
			})
			out.Summary.ByCheck[s.Category]++
			out.Summary.Total++
		}
	}
	if out.Summary.Total > 0 {
		out.Status = "fail"
	}
	return json.MarshalIndent(out, "", "  ")
}

// --- bubbletea TUI 구현 ---

type stepDoneMsg struct {
	idx  int
	errs []string
	err  error
}

type stepResult struct {
	name   string
	errs   int
	failed bool
}

type model struct {
	steps    []Step
	spinner  spinner.Model
	results  []stepResult
	current  int
	done     bool
	allErrs  []string
	fatalErr error
	stepErrs [][]string // 단계별 오류 메시지
}

func newModel(steps []Step, opts Options) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	if !opts.NoColor {
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	}
	return model{
		steps:    steps,
		spinner:  s,
		results:  make([]stepResult, len(steps)),
		stepErrs: make([][]string, len(steps)),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runStep(0))
}

func (m model) runStep(idx int) tea.Cmd {
	return func() tea.Msg {
		errs, err := m.steps[idx].Fn()
		return stepDoneMsg{idx: idx, errs: errs, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.fatalErr = fmt.Errorf("interrupted")
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stepDoneMsg:
		m.stepErrs[msg.idx] = msg.errs
		m.results[msg.idx] = stepResult{
			name:   m.steps[msg.idx].Name,
			errs:   len(msg.errs),
			failed: msg.err != nil,
		}
		m.allErrs = append(m.allErrs, msg.errs...)
		if msg.err != nil {
			m.fatalErr = msg.err
			m.done = true
			return m, tea.Quit
		}
		next := msg.idx + 1
		m.current = next
		if next >= len(m.steps) {
			m.done = true
			return m, tea.Quit
		}
		return m, m.runStep(next)
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	checkMark := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	crossMark := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
	warnMark := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("!")

	for i, r := range m.results {
		if i >= m.current && !m.done {
			break
		}
		icon := checkMark
		if r.failed {
			icon = crossMark
		} else if r.errs > 0 {
			icon = warnMark
		}
		suffix := ""
		if r.errs > 0 {
			suffix = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).
				Render(fmt.Sprintf(" (%d issues)", r.errs))
		}
		fmt.Fprintf(&b, "  %s %s%s\n", icon, r.name, suffix)
	}

	if !m.done && m.current < len(m.steps) {
		fmt.Fprintf(&b, "  %s %s\n", m.spinner.View(), m.steps[m.current].Name)
	}

	return b.String()
}

func runTUI(steps []Step, opts Options) (RunResult, error) {
	m := newModel(steps, opts)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		// bubbletea 자체 에러 시 plain 으로 폴백
		return runPlain(steps, os.Stderr)
	}
	final := finalModel.(model)
	if final.fatalErr != nil {
		return RunResult{}, final.fatalErr
	}
	result := RunResult{AllErrors: final.allErrs}
	result.Steps = make([]StepResult, len(steps))
	for i := range steps {
		result.Steps[i] = StepResult{
			Name:     steps[i].Name,
			Category: steps[i].Category,
			Errors:   final.stepErrs[i],
			Failed:   final.results[i].failed,
		}
	}
	return result, nil
}
