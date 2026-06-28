package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/gh"
)

type actionItem struct {
	key   string
	label string
	hint  string // dim one-line explanation shown next to the label
	act   ActionType
	group int // items are visually separated when the group changes
}

var actionMenu = []actionItem{
	{"1", "Clone", "the full repository", ActionClone, 0},
	{"2", "Clone branch…", "choose a single branch to clone", ActionCloneBranch, 0},
	{"3", "Browse files…", "pick folders/files for a partial clone", ActionSparseClone, 0},
	{"4", "Copy name", "owner/repo to the clipboard", ActionCopyName, 1},
	{"5", "Copy URL", "the https link to the clipboard", ActionCopyURL, 1},
	{"6", "Open in browser", "the repo's page on github.com", ActionOpenWeb, 1},
}

func (a *App) updateActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch key.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "esc", "left", "h":
		a.screen = screenList
		return a, nil
	case "up", "k":
		if a.actionCursor > 0 {
			a.actionCursor--
		}
		return a, nil
	case "down", "j":
		if a.actionCursor < len(actionMenu)-1 {
			a.actionCursor++
		}
		return a, nil
	case "enter", "right", "l":
		return a.chooseAction(actionMenu[a.actionCursor].act)
	}
	// Number shortcuts.
	for _, it := range actionMenu {
		if key.String() == it.key {
			return a.chooseAction(it.act)
		}
	}
	return a, nil
}

func (a *App) chooseAction(act ActionType) (tea.Model, tea.Cmd) {
	switch act {
	case ActionSparseClone:
		return a.enterTree()
	case ActionCloneBranch:
		return a.enterBranches()
	}
	a.Result = Result{Action: act, Repo: a.selected}
	return a, tea.Quit
}

var actionsFooter = keyHint(
	[2]string{"↑↓", "move"},
	[2]string{"⏎", "select"},
	[2]string{"1-6", "shortcut"},
	[2]string{"esc", "back"},
	[2]string{"^C", "quit"},
)

func (a *App) viewActions() string {
	r := a.selected
	var b strings.Builder

	// Compact repo summary. The chrome header already shows the name, so here we
	// lead with the metadata badge and a (truncated) description for context.
	b.WriteString("\n  ")
	b.WriteString(repoMeta(r))
	b.WriteString("\n")
	if r.Description != "" {
		b.WriteString("  " + dimStyle.Render(truncate(r.Description, a.width-4)) + "\n")
	}
	b.WriteString("\n")

	// Align the dim hints into a column for easy scanning.
	labelW := 0
	for _, it := range actionMenu {
		if w := lipgloss.Width(it.label); w > labelW {
			labelW = w
		}
	}

	for i, it := range actionMenu {
		if i > 0 && it.group != actionMenu[i-1].group {
			b.WriteString("\n") // blank line between clone vs. copy/open groups
		}

		cursor := "   "
		label := it.label
		if i == a.actionCursor {
			cursor = " " + selectedStyle.Render("▸") + " "
			label = selectedStyle.Render(label)
		}
		pad := strings.Repeat(" ", labelW-lipgloss.Width(it.label))
		fmt.Fprintf(&b, "%s%s  %s%s   %s\n",
			cursor, keyStyle.Render(it.key), label, pad, dimStyle.Render(it.hint))
	}

	return b.String()
}

// repoMeta renders the visibility badge plus language and star count.
func repoMeta(r gh.Repo) string {
	meta := publicBadge.Render("public")
	if r.IsPrivate {
		meta = privateBadge.Render("private")
	}
	if r.Language != "" {
		meta += dimStyle.Render(" · " + r.Language)
	}
	if r.StargazerCount > 0 {
		meta += dimStyle.Render(fmt.Sprintf(" · ★%d", r.StargazerCount))
	}
	return meta
}
