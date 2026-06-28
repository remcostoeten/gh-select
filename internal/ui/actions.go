package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type actionItem struct {
	key   string
	label string
	act   ActionType
}

var actionMenu = []actionItem{
	{"1", "Clone repository (full)", ActionClone},
	{"2", "Clone a specific branch…", ActionCloneBranch},
	{"3", "Browse & partial clone (pick folders/files)", ActionSparseClone},
	{"4", "Copy repository name", ActionCopyName},
	{"5", "Copy repository URL", ActionCopyURL},
	{"6", "Open in browser", ActionOpenWeb},
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
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  " + r.NameWithOwner))
	b.WriteString("\n")

	// Metadata line: visibility · language · stars.
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
	b.WriteString("  " + meta + "\n")
	b.WriteString("  " + dimStyle.Render(r.URL()) + "\n")
	if r.Description != "" {
		b.WriteString("  " + dimStyle.Render(r.Description) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(headerStyle.Render("  Actions"))
	b.WriteString("\n\n")

	for i, it := range actionMenu {
		cursor := "   "
		label := it.label
		if i == a.actionCursor {
			cursor = " " + selectedStyle.Render("▸") + " "
			label = selectedStyle.Render(label)
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, keyStyle.Render(it.key), label))
	}

	return b.String()
}
