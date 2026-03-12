package progress

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// stepResult: 단계 실행 결과.
type stepResult struct {
	name   string
	errs   int
	failed bool
}

// Step: 실행할 검사 단계 정보.
type Step struct {
	Name string
	Fn   func() ([]string, error)
}

// RunWithProgress: bubbletea 스피너와 함께 검사 단계를 순차 실행.
// TTY가 아닌 경우 단순 텍스트 출력으로 폴백.
func RunWithProgress(steps []Step) (allErrs []string, fatalErr error) {
	if !isTTY() {
		return runPlain(steps, os.Stderr)
	}
	return runTUI(steps)
}

func isTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runPlain: TTY가 아닐 때 단순 텍스트로 진행 표시.
func runPlain(steps []Step, w io.Writer) ([]string, error) {
	var allErrs []string
	for _, s := range steps {
		_, _ = fmt.Fprintf(w, "  %s ...\n", s.Name)
		errs, err := s.Fn()
		if err != nil {
			return nil, err
		}
		allErrs = append(allErrs, errs...)
	}
	return allErrs, nil
}

// --- bubbletea TUI 모델 ---

type stepDoneMsg struct {
	idx  int
	errs []string
	err  error
}

type model struct {
	steps    []Step
	spinner  spinner.Model
	results  []stepResult
	current  int
	done     bool
	allErrs  []string
	fatalErr error
}

func newModel(steps []Step) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		steps:   steps,
		spinner: s,
		results: make([]stepResult, len(steps)),
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

func runTUI(steps []Step) ([]string, error) {
	m := newModel(steps)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		// bubbletea 자체 에러 시 plain 으로 폴백
		return runPlain(steps, os.Stderr)
	}
	final := result.(model)
	if final.fatalErr != nil {
		return nil, final.fatalErr
	}
	return final.allErrs, nil
}
