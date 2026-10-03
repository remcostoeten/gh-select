package ui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// langsDelay keeps fast scrolling from firing a request per row passed.
const langsDelay = 150 * time.Millisecond

type langsDueMsg struct{ repo string }
type langsLoadedMsg struct {
	repo  string
	langs []string
}

// highlighted is the repo under the list cursor, or "" when there is none.
func (a *App) highlighted() string {
	if it, ok := a.list.SelectedItem().(repoItem); ok {
		return it.repo.NameWithOwner
	}
	return ""
}

// langsCmd schedules a languages fetch for the highlighted repo once the
// cursor has rested on it, unless one is already known or on its way.
func (a *App) langsCmd() tea.Cmd {
	repo := a.highlighted()
	if repo == "" || repo == a.langsWant || a.client == nil {
		return nil
	}
	if _, known := a.langs[repo]; known {
		return nil
	}
	a.langsWant = repo
	return tea.Tick(langsDelay, func(time.Time) tea.Msg { return langsDueMsg{repo: repo} })
}

func (a *App) updateLangs(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case langsDueMsg:
		if msg.repo != a.highlighted() {
			a.langsWant = ""
			return nil
		}
		return func() tea.Msg {
			langs, _ := a.client.FetchLanguages(msg.repo)
			return langsLoadedMsg{repo: msg.repo, langs: langs}
		}
	case langsLoadedMsg:
		if a.langs == nil {
			a.langs = map[string][]string{}
		}
		a.langs[msg.repo] = msg.langs
		if a.langsWant == msg.repo {
			a.langsWant = ""
		}
	}
	return nil
}

// langList renders languages side by side, dropping the ones that don't fit
// in w cells behind a "+N" count.
func langList(langs []string, w int) string {
	gap := "  "
	var b strings.Builder
	for i, l := range langs {
		item := langLabel(l)
		if i > 0 {
			item = gap + item
		}
		rest := len(langs) - i - 1
		more := ""
		if rest > 0 {
			more = gap + dimStyle.Render("+"+strconv.Itoa(rest))
		}
		if lipgloss.Width(b.String()+item+more) > w {
			if i == 0 {
				return truncate(item, w)
			}
			return b.String() + gap + dimStyle.Render("+"+strconv.Itoa(len(langs)-i))
		}
		b.WriteString(item)
	}
	return b.String()
}
