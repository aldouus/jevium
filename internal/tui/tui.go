package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/page"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

type Model struct {
	agent  *agent.Agent
	err    error
	width  int
	height int
	done   bool
}

type tickMsg struct{ err error }

func New(a *agent.Agent) Model { return Model{agent: a} }

func (m Model) Init() tea.Cmd { return m.step() }

func (m Model) step() tea.Cmd {
	return func() tea.Msg {
		err := m.agent.Command("tick", "")
		return tickMsg{err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.done = true
			return m, tea.Quit
		}
	case tickMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}
		if m.agent.State.Status == "done" || m.agent.State.Status == "blocked" {
			m.done = true
			return m, tea.Quit
		}
		return m, m.step()
	}
	return m, nil
}

func (m Model) View() string {
	s := m.agent.State
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", titleStyle.Render("jevium"))
	fmt.Fprintf(&b, "%s\n", mutedStyle.Render(s.Goal))
	status := s.Status
	if m.err != nil {
		status = errStyle.Render(m.err.Error())
	} else if s.Status == "done" {
		status = okStyle.Render("done")
	}
	fmt.Fprintf(&b, "status %s  %d actions  %d ms\n", status, len(s.History), s.ElapsedMS)
	fmt.Fprintf(&b, "%s\n", mutedStyle.Render(s.Page.URL+"  "+s.Page.Title))
	b.WriteString(renderActions(s.Page, 12))
	if n := len(s.History); n > 0 {
		h := s.History[n-1]
		fmt.Fprintf(&b, "\nlast %s %s\n", h.Operation, h.Action)
	}
	fmt.Fprintf(&b, "\n%s", mutedStyle.Render("q to quit"))
	return boxStyle.Render(b.String())
}

func renderActions(p page.Page, limit int) string {
	var b strings.Builder
	n := 0
	for _, a := range p.Actions {
		if n >= limit {
			fmt.Fprintf(&b, "  … %d more\n", len(p.Actions)-limit)
			break
		}
		fmt.Fprintf(&b, "  %-6s %s\n", a.ID, a.Label)
		n++
	}
	return b.String()
}

func Run(a *agent.Agent) error {
	p := tea.NewProgram(New(a))
	_, err := p.Run()
	return err
}
