package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type stepState int

const (
	stepPending stepState = iota
	stepRunning
	stepDone
	stepFailed
)

// Step describes one unit of work in a progress sequence.
type Step struct {
	Label string
	Run   func() error
}

type stepDoneMsg struct {
	index int
	err   error
}

// ── Model ─────────────────────────────────────────────────────────────────────

type ProgressModel struct {
	title    string
	steps    []Step
	states   []stepState
	errs     []error
	spinner  spinner.Model
	done     bool
	finalErr error
}

func NewProgress(title string, steps []Step) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	states := make([]stepState, len(steps))
	states[0] = stepRunning

	return ProgressModel{
		title:   title,
		steps:   steps,
		states:  states,
		errs:    make([]error, len(steps)),
		spinner: s,
	}
}

// Err returns the final error after the program exits.
func (m ProgressModel) Err() error { return m.finalErr }

// ── Init ─────────────────────────────────────────────────────────────────────

func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runStep(0))
}

func (m ProgressModel) runStep(i int) tea.Cmd {
	return func() tea.Msg {
		return stepDoneMsg{index: i, err: m.steps[i].Run()}
	}
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case stepDoneMsg:
		if msg.err != nil {
			m.states[msg.index] = stepFailed
			m.errs[msg.index] = msg.err
			m.done = true
			m.finalErr = msg.err
			return m, tea.Quit
		}
		m.states[msg.index] = stepDone
		next := msg.index + 1
		if next >= len(m.steps) {
			m.done = true
			return m, tea.Quit
		}
		m.states[next] = stepRunning
		return m, m.runStep(next)
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m ProgressModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title) + "\n\n")

	for i, step := range m.steps {
		switch m.states[i] {
		case stepPending:
			b.WriteString(dimStyle.Render("  ○  "+step.Label) + "\n")
		case stepRunning:
			b.WriteString("  " + m.spinner.View() + "  " + step.Label + "\n")
		case stepDone:
			b.WriteString(okStyle.Render("  ✓  "+step.Label) + "\n")
		case stepFailed:
			b.WriteString(errStyle.Render("  ✗  "+step.Label) + "\n")
			if m.errs[i] != nil {
				for _, line := range strings.Split(m.errs[i].Error(), "\n") {
					if line = strings.TrimSpace(line); line != "" {
						b.WriteString(errStyle.Render("       "+line) + "\n")
					}
				}
			}
		}
	}

	return b.String()
}
