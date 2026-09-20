package ui

import (
	"fmt"
	"strconv"
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

// menuFor builds the action list for a repo. A repo already cloned on disk
// leads with open/pull instead of a clone that would fail or duplicate it.
func (a *App) menuFor(r gh.Repo) []actionItem {
	var items []actionItem
	if path := a.localPath(r.NameWithOwner); path != "" {
		items = append(items,
			actionItem{"", "Open in editor", "the local clone in $EDITOR", ActionOpenEditor, 0},
			actionItem{"", "Pull", "fast-forward the local clone", ActionPull, 0},
			actionItem{"", "Clone again", "a second copy of the repository", ActionClone, 1},
		)
	} else {
		items = append(items,
			actionItem{"", "Clone", "the full repository", ActionClone, 0},
			actionItem{"", "Clone & open", "clone, then open it in $EDITOR", ActionOpenEditor, 0},
		)
	}
	items = append(items,
		actionItem{"", "Clone branch…", "choose a single branch to clone", ActionCloneBranch, 1},
		actionItem{"", "Browse files…", "pick folders/files for a partial clone", ActionSparseClone, 1},
		actionItem{"", "Copy name", "owner/repo to the clipboard", ActionCopyName, 2},
		actionItem{"", "Copy URL", "the https link to the clipboard", ActionCopyURL, 2},
		actionItem{"", "Open in browser", "the repo's page on github.com", ActionOpenWeb, 2},
	)
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

	// Align the dim hints into a column for easy scanning.
	labelW := 0
	for _, it := range a.actions {
		if w := lipgloss.Width(it.label); w > labelW {
			labelW = w
		}
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
		fmt.Fprintf(&b, "%s%s  %s%s   %s\n",
			cursor, keyStyle.Render(it.key), label, pad, dimStyle.Render(it.hint))
	}

	return b.String()
}

// detailColumn renders the right-hand column as two stacked cards — a compact
// key/value info card and a description card — filling exactly h rows so it
// bottom-aligns with the main panel beside it.
func detailColumn(r gh.Repo, localPath string, w, h int) string {
	kv := func(label, value string) string {
		return dimStyle.Render(fmt.Sprintf("%-8s", label)) + value
	}
	access := publicBadge.Render("public")
	if r.IsPrivate {
		access = privateBadge.Render("private")
	}
	lang, stars := dimStyle.Render("—"), dimStyle.Render("—")
	if r.Language != "" {
		lang = r.Language
	}
	if r.StargazerCount > 0 {
		stars = fmt.Sprintf("★%d", r.StargazerCount)
	}
	info := kv("repo", titleStyle.Render(r.NameWithOwner)) + "\n" +
		kv("access", access) + "\n" +
		kv("lang", lang) + "\n" +
		kv("stars", stars)

	infoH := 6 // border (2) + 4 rows
	if localPath != "" {
		info += "\n" + kv("local", truncate(localPath, w-14))
		infoH++
	}
	if h <= infoH+3 {
		return panel("info", info, w, h, false)
	}

	desc := r.Description
	if desc == "" {
		desc = "no description"
	}
	wrapped := lipgloss.NewStyle().Foreground(colDim).Width(w - 4).Render(desc)
	return panel("info", info, w, infoH, false) + "\n\n" +
		panel("description", wrapped, w, h-infoH-1, false)
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
