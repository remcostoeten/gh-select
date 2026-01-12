package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

type Action struct {
	ID    string
	Label string
	Info  string
}

func (a Action) Title() string       { return a.Label }
func (a Action) Description() string { return a.Info }
func (a Action) FilterValue() string { return a.Label }

type actionModel struct {
	list     list.Model
	selected string
	repo     *gh.Repository
}

func (m actionModel) Init() tea.Cmd {
	return nil
}

func (m actionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			if i, ok := m.list.SelectedItem().(Action); ok {
				m.selected = i.ID
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m actionModel) View() string {
	return docStyle.Render(m.list.View())
}

func SelectAction(repo *gh.Repository) (string, error) {
	actions := []list.Item{
		Action{ID: "clone", Label: "Clone", Info: "Clone repository to current directory"},
		Action{ID: "copy-name", Label: "Copy Name", Info: "Copy owner/repo to clipboard"},
		Action{ID: "copy-url", Label: "Copy URL", Info: "Copy full GitHub URL to clipboard"},
		Action{ID: "open", Label: "Open", Info: "Open in default browser"},
		Action{ID: "print", Label: "Print", Info: "Print name and exit"},
	}

	l := list.New(actions, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf(" Actions for %s ", repo.NameWithOwner)
	l.Styles.Title = StyleTitle
	l.Styles.PaginationStyle = StyleFooter
	l.Styles.HelpStyle = StyleFooter

	m := actionModel{list: l, repo: repo}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := finalModel.(actionModel); ok {
		return m.selected, nil
	}
	return "", nil
}
