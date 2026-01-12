package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/cache"
	"github.com/remcostoeten/gh-select/internal/gh"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	repo gh.Repository
}

func (i item) Title() string       { return i.repo.NameWithOwner }
func (i item) Description() string { return i.repo.Description }
func (i item) FilterValue() string { return i.repo.NameWithOwner }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := strings.Builder{}

	// Privacy Tag
	privacy := "🌐"
	tag := StyleTagPublic.Render("PUBLIC")
	if i.repo.IsPrivate {
		privacy = "🔒"
		tag = StyleTagPrivate.Render("PRIVATE")
	}

	title := i.Title()
	desc := i.Description()
	if len(desc) > 50 {
		desc = desc[:47] + "..."
	}

	if index == m.Index() {
		str.WriteString(StyleSelectedItem.Render(
			fmt.Sprintf("%s %-40s %s\n   %s", privacy, title, tag, StyleDimText.Render(desc)),
		))
	} else {
		str.WriteString(fmt.Sprintf("  %s %-40s %s\n     %s", privacy, title, tag, StyleDimText.Render(desc)))
	}

	fmt.Fprint(w, str.String())
}

type model struct {
	list     list.Model
	selected *item
	err      error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = &i
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

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func Start(noCache bool) (*gh.Repository, error) {
	var repos []gh.Repository
	var err error

	if !noCache {
		repos, err = cache.Load()
	}

	if repos == nil {
		repos, err = gh.FetchRepos(1000, false)
		if err != nil {
			return nil, err
		}
		_ = cache.Save(repos)
	}

	items := make([]list.Item, len(repos))
	for i, repo := range repos {
		items[i] = item{repo: repo}
	}

	l := list.New(items, itemDelegate{}, 0, 0)
	l.Title = " GH SELECT "
	l.Styles.Title = StyleTitle
	l.Styles.PaginationStyle = StyleFooter
	l.Styles.HelpStyle = StyleFooter

	m := model{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := finalModel.(model); ok && m.selected != nil {
		return &m.selected.repo, nil
	}
	return nil, nil
}

