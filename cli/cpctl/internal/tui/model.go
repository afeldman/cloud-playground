package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ──────────────────────────────────────────────────────────────────

var (
	border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	labelStyle = lipgloss.NewStyle().
			Width(14).
			Foreground(lipgloss.Color("245"))

	okStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Bold(true)

	serviceOk  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	serviceErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	keysStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// ── Messages ─────────────────────────────────────────────────────────────────

type statusMsg PlaygroundStatus
type tickMsg time.Time

// ── Model ────────────────────────────────────────────────────────────────────

type model struct {
	status  PlaygroundStatus
	spinner spinner.Model
	loading bool
	width   int
	height  int
	root    string
	name    string
}

func New(root, name string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	return model{
		spinner: s,
		loading: true,
		root:    root,
		name:    name,
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.doCheck(),
	)
}

func (m model) doCheck() tea.Cmd {
	return func() tea.Msg {
		return statusMsg(CheckAll(m.root, m.name))
	}
}

func tickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case statusMsg:
		m.status = PlaygroundStatus(msg)
		m.loading = false
		return m, tickAfter(10 * time.Second)

	case tickMsg:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.doCheck())

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.doCheck())
		case "t":
			tfDir := filepath.Join(m.root, "terraform", "localstack")
			return m, tea.ExecProcess(
				terraformPlanCmd(tfDir),
				func(err error) tea.Msg { return statusMsg(m.status) },
			)
		}
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("birdy-playground") + "\n\n")

	// Kind
	b.WriteString(m.renderKind())
	b.WriteString("\n")

	// LocalStack
	b.WriteString(m.renderLocalStack())
	b.WriteString("\n")

	// Terraform
	b.WriteString(m.renderTerraform())
	b.WriteString("\n")

	// Services section
	if m.status.LocalStack.Running && len(m.status.LocalStack.Services) > 0 {
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("─── Services " + strings.Repeat("─", 30)))
		b.WriteString("\n")
		b.WriteString(m.renderServices())
		b.WriteString("\n")
	}

	// Last refreshed
	b.WriteString("\n")
	if m.loading {
		b.WriteString(dimStyle.Render(m.spinner.View() + " refreshing…"))
	} else {
		age := time.Since(m.status.CheckedAt).Round(time.Second)
		b.WriteString(dimStyle.Render(fmt.Sprintf("last refreshed %s ago", age)))
	}

	b.WriteString("\n\n")
	b.WriteString(keysStyle.Render("q quit  r refresh  t terraform plan"))

	return border.Render(b.String())
}

func (m model) renderKind() string {
	k := m.status.Kind
	label := labelStyle.Render("Kind cluster")
	if !k.Running {
		return label + errStyle.Render("✗ not running") + " " + dimStyle.Render(k.Name)
	}
	detail := fmt.Sprintf("(%d/%d nodes ready)", k.Ready, k.Nodes)
	return label + okStyle.Render("● "+k.Name) + "  " + dimStyle.Render(detail)
}

func (m model) renderLocalStack() string {
	ls := m.status.LocalStack
	label := labelStyle.Render("LocalStack")
	if !ls.Running {
		return label + errStyle.Render("✗ not running")
	}
	return label + okStyle.Render("● http://localhost:4566")
}

func (m model) renderTerraform() string {
	tf := m.status.Terraform
	label := labelStyle.Render("Terraform")
	if !tf.Applied {
		return label + errStyle.Render("✗ no state")
	}
	return label + okStyle.Render(fmt.Sprintf("● %d resources applied", tf.Resources))
}

func (m model) renderServices() string {
	services := m.status.LocalStack.Services

	// Sort service names for stable output
	names := make([]string, 0, len(services))
	for n := range services {
		names = append(names, n)
	}
	sort.Strings(names)

	// Render in two-column layout
	var rows []string
	for i := 0; i < len(names); i += 2 {
		left := renderService(names[i], services[names[i]])
		if i+1 < len(names) {
			right := renderService(names[i+1], services[names[i+1]])
			rows = append(rows, fmt.Sprintf("  %-36s%s", left, right))
		} else {
			rows = append(rows, "  "+left)
		}
	}

	return strings.Join(rows, "\n")
}

func terraformPlanCmd(dir string) *exec.Cmd {
	cmd := exec.Command("terraform", "-chdir="+dir, "plan")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func renderService(name, status string) string {
	nameW := fmt.Sprintf("%-12s", name)
	if status == "running" || status == "available" {
		return nameW + serviceOk.Render("✓ running    ")
	}
	return nameW + serviceErr.Render("✗ "+status+"  ")
}
