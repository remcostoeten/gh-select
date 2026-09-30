package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

// menuFor builds the action list for a repo. A repo already cloned on disk
// leads with open/pull instead of a clone that would fail or duplicate it.
func (a *App) menuFor(r gh.Repo) []actionItem {
	var items []actionItem
	if path := a.localPath(r.NameWithOwner); path != "" {
		items = append(items,
			actionItem{"", "Open in editor", "the local clone in $EDITOR", ActionOpenEditor, 0},
			actionItem{"", "Pull", "fast-forward the local clone", ActionPull, 0},
			actionItem{"", "Clone again", "a second, separate copy", ActionClone, 1},
		)
	} else {
		items = append(items,
			actionItem{"", "Clone", "the full repository", ActionClone, 0},
			actionItem{"", "Clone & open", "then open it in $EDITOR", ActionOpenEditor, 0},
		)
	}
	items = append(items,
		actionItem{"", "Clone branch…", "just the branch you pick", ActionCloneBranch, 1},
		actionItem{"", "Browse files…", "preview, save or clone paths", ActionSparseClone, 1},
		actionItem{"", "View README", "rendered, right here", ActionViewReadme, 1},
		actionItem{"", "Releases…", "notes and asset downloads", ActionReleases, 1},
		actionItem{"", "Copy name", "owner/repo to clipboard", ActionCopyName, 2},
		actionItem{"", "Copy URL", "https link to clipboard", ActionCopyURL, 2},
		actionItem{"", "Open in browser", "its page on github.com", ActionOpenWeb, 2},
	)
	if deletable(r) {
		items = append(items,
			actionItem{"", "Delete repository…", "on GitHub, cannot be undone", ActionDelete, 3})
	}
	for i := range items {
		items[i].key = strconv.Itoa(i + 1)
	}
	return items
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
		a.status = "" // don't carry an actions-screen message back to the list
		return a, nil
	case "up", "k":
		if a.actionCursor > 0 {
			a.actionCursor--
		}
		return a, nil
	case "down", "j":
		if a.actionCursor < len(a.actions)-1 {
			a.actionCursor++
		}
		return a, nil
	case "enter", "right", "l":
		return a.chooseAction(a.actions[a.actionCursor].act)
	}
	// Number shortcuts.
	for _, it := range a.actions {
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
	case ActionViewReadme:
		return a.enterReadme()
	case ActionReleases:
		return a.enterReleases()
	case ActionDelete:
		a.confirmReturn = screenActions
		return a.enterConfirmDelete([]gh.Repo{a.selected})
	}
	a.Result = Result{
		Action:    act,
		Repo:      a.selected,
		LocalPath: a.localPath(a.selected.NameWithOwner),
	}
	return a, tea.Quit
}

func (a *App) actionsFooter() string {
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"enter", "select"},
		[2]string{fmt.Sprintf("1-%d", len(a.actions)), "shortcut"},
		[2]string{"esc", "back"},
		[2]string{"^C", "quit"},
	)
}

// viewActions renders the action menu panel with the repo's details alongside
// when the terminal is wide enough; on narrow terminals the summary moves into
// the menu panel instead.
func (a *App) viewActions() string {
	h := a.height - chromeLines
	mw, sw := splitWidths(contentWidth(a.width))
	main := panel("actions", a.actionsBody(sw == 0), mw, h, true)
	if sw > 0 {
		main = hsplit(main, detailColumn(a.selected, a.localPath(a.selected.NameWithOwner), sw, h))
	}
	return main
}

func (a *App) actionsBody(includeSummary bool) string {
	var b strings.Builder

	if includeSummary {
		b.WriteString("\n  ")
		b.WriteString(repoMeta(a.selected))
		b.WriteString("\n")
		if a.selected.Description != "" {
			b.WriteString("  " + dimStyle.Render(truncate(a.selected.Description, a.width-8)) + "\n")
		}
	}
	b.WriteString("\n")

	labelW, keyW := 0, 0
	for _, it := range a.actions {
		labelW = max(labelW, lipgloss.Width(it.label))
		keyW = max(keyW, lipgloss.Width(it.key))
	}

	for i, it := range a.actions {
		if i > 0 && it.group != a.actions[i-1].group {
			b.WriteString("\n") // blank line between clone vs. copy/open groups
		}

		cursor := "   "
		label := it.label
		if i == a.actionCursor {
			cursor = " " + selectedStyle.Render("❯") + " "
			label = selectedStyle.Render(label)
		}
		pad := strings.Repeat(" ", labelW-lipgloss.Width(it.label))
		hint := dimStyle.Render(it.hint)
		if it.act == ActionDelete {
			label = errStyle.UnsetBold().Render(it.label)
			if i == a.actionCursor {
				label = errStyle.Render(it.label)
			}
		}
		fmt.Fprintf(&b, "%s%s  %s%s   %s\n",
			cursor, padLeft(keyStyle.Render(it.key), keyW), label, pad, hint)
	}

	return b.String()
}

func detailColumn(r gh.Repo, localPath string, w, h int) string {
	inner := w - 4
	kv := func(label, value string) string {
		return dimStyle.Render(fmt.Sprintf("%-9s", label)) + value
	}
	access := publicBadge.Render("public")
	if r.IsPrivate {
		access = privateBadge.Render("private")
	}
	facts := []string{kv("access", access)}
	if r.Language != "" {
		facts = append(facts, kv("language", langLabel(r.Language)))
	}
	if r.StargazerCount > 0 {
		facts = append(facts, kv("stars", starLabel(r.StargazerCount)))
	}
	if !r.UpdatedAt.IsZero() {
		facts = append(facts, kv("updated", age(r.UpdatedAt, time.Now())))
	}
	if localPath != "" {
		facts = append(facts, kv("local", publicBadge.Render(truncate(localPath, inner-9))))
	}

	desc := dimStyle.Render("no description")
	if r.Description != "" {
		desc = lipgloss.NewStyle().Foreground(colFg).Width(inner).Render(r.Description)
	}
	descLines := strings.Split(desc, "\n")
	if room := h - 2 - len(facts) - 1; len(descLines) > room {
		descLines = descLines[:max(room, 0)]
		if room > 0 {
			descLines[room-1] = truncate(descLines[room-1]+"…", inner)
		}
	}
	body := strings.Join(descLines, "\n")
	if len(descLines) > 0 {
		body += "\n\n"
	}
	return panel(r.NameWithOwner, body+strings.Join(facts, "\n"), w, h, false)
}

// repoMeta renders the visibility badge plus language and star count.
func repoMeta(r gh.Repo) string {
	meta := publicBadge.Render("public")
	if r.IsPrivate {
		meta = privateBadge.Render("private")
	}
	if r.Language != "" {
		meta += dimStyle.Render(" · ") + langLabel(r.Language)
	}
	if r.StargazerCount > 0 {
		meta += dimStyle.Render(" · " + starLabel(r.StargazerCount))
	}
	return meta
}
