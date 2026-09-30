package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// repoItem adapts gh.Repo to the bubbles list item interface.
type repoItem struct{ repo gh.Repo }

func (i repoItem) Title() string { return i.repo.NameWithOwner }

// FilterValue lets the built-in filter match on name and description.
func (i repoItem) FilterValue() string {
	return i.repo.NameWithOwner + " " + i.repo.Description
}

// compactDelegate renders each repo as a single dense row — cursor, name, and
// dim metadata — the way dashboard TUIs list entries, instead of the default
// two-line title/description blocks.
//
// bare drops the owner prefix, which is dead weight in the "my repos" scope
// where every row has the same owner. GitHub search keeps it: there the owner
// is what tells two same-named repos apart.
type compactDelegate struct {
	bare   bool
	locals map[string]string  // "owner/repo" → working copy already on disk
	marked map[string]gh.Repo // "owner/repo" → marked for bulk deletion
}

func (compactDelegate) Height() int                         { return 1 }
func (compactDelegate) Spacing() int                        { return 0 }
func (compactDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(repoItem)
	if !ok {
		return
	}
	r := it.repo
	width := m.Width()

	cursor := "  "
	nameStyle := lipgloss.NewStyle().Foreground(colFg)
	if d.bare && !r.IsOwner {
		nameStyle = dimStyle
	}
	if index == m.Index() {
		cursor = selectedStyle.Render("❯ ")
		nameStyle = selectedStyle
	}

	// The mark column only exists once something is marked, so every row shifts
	// together and an unmarked list keeps its original density.
	mark := ""
	if len(d.marked) > 0 {
		mark = "  "
		if _, ok := d.marked[r.NameWithOwner]; ok {
			mark = errStyle.Render("✗ ")
			nameStyle = errStyle.Strikethrough(true)
		}
	}

	cols := d.columns(m.Items())
	right := ""
	if cols.lang+cols.stars > 0 && width >= 48 {
		right = "  " + padRight(langLabel(r.Language), cols.lang) + "  " +
			padLeft(dimStyle.Render(starLabel(r.StargazerCount)), cols.stars)
	}

	prefix := cursor + mark
	leftW := width - lipgloss.Width(right)
	nameW := max(min(cols.name, leftW-lipgloss.Width(prefix)-cols.tags), 8)
	left := prefix + padRight(nameStyle.Render(truncate(d.label(r), nameW)), nameW) + d.tags(r)
	fmt.Fprint(w, padRight(left, leftW)+right)
}

func (d compactDelegate) label(r gh.Repo) string {
	if d.bare {
		return r.Name()
	}
	return r.NameWithOwner
}

func (d compactDelegate) tags(r gh.Repo) string {
	var out string
	if _, cloned := d.locals[r.NameWithOwner]; cloned {
		out += " " + publicBadge.Render("local")
	}
	if r.IsPrivate {
		out += " " + privateBadge.Render("private")
	}
	if out != "" {
		out = " " + out
	}
	return out
}

type rowColumns struct{ name, tags, lang, stars int }

func (d compactDelegate) columns(items []list.Item) rowColumns {
	var c rowColumns
	for _, item := range items {
		it, ok := item.(repoItem)
		if !ok {
			continue
		}
		c.name = max(c.name, lipgloss.Width(d.label(it.repo)))
		c.tags = max(c.tags, lipgloss.Width(d.tags(it.repo)))
		c.lang = max(c.lang, lipgloss.Width(langLabel(it.repo.Language)))
		c.stars = max(c.stars, lipgloss.Width(starLabel(it.repo.StargazerCount)))
	}
	c.lang = min(c.lang, 16)
	return c
}

func padRight(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return truncate(s, w)
}

func padLeft(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

func newRepoList(repos []gh.Repo, width, height int) list.Model {
	l := list.New(repoItems(repos), compactDelegate{bare: true}, width, height)
	// The persistent chrome (header/footer) and our own always-on type-to-search
	// replace the list's built-in title, status bar, help and filter.
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Paginator.Type = paginator.Arabic // a dim "2/12" beats a row of dots
	l.Styles.PaginationStyle = dimStyle
	return l
}

func repoItems(repos []gh.Repo) []list.Item {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = repoItem{repo: r}
	}
	return items
}
