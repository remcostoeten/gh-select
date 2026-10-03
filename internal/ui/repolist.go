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

	cursor := ""
	nameStyle := lipgloss.NewStyle().Foreground(colFg)
	if d.bare && !r.IsOwner {
		nameStyle = dimStyle
	}
	if index == m.Index() {
		cursor = selectRow("")
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

	l := d.layout(m)
	row := cursor + mark + padRight(nameStyle.Render(truncate(d.label(r), l.name)), l.name)
	if l.cols.tags > 0 {
		row += "  " + padRight(d.tags(r), l.cols.tags)
	}
	if l.cols.lang > 0 {
		row += "  " + padRight(langLabel(r.Language), l.cols.lang)
	}
	stars := ""
	if l.cols.stars > 0 {
		stars = padLeft(dimStyle.Render(starLabel(r.StargazerCount)), l.cols.stars)
	}
	fmt.Fprint(w, padRight(row, width-lipgloss.Width(stars))+stars)
}

// rowLayout is how a list row splits its width: the name column and the
// widths of the tags, language and stars columns after it.
type rowLayout struct {
	name int
	cols rowColumns
}

// layout sizes the name column to the longest name, capped at about half the
// row so the language column sits mid-row, and drops the metadata columns on
// narrow lists.
func (d compactDelegate) layout(m list.Model) rowLayout {
	cols := d.columns(m.Items())
	width := m.Width()
	if width < 48 {
		cols = rowColumns{name: cols.name}
	}
	mark := 0
	if len(d.marked) > 0 {
		mark = 2
	}
	fixed := mark
	for _, c := range []int{cols.tags, cols.lang, cols.stars} {
		if c > 0 {
			fixed += c + 2
		}
	}
	name := min(cols.name, max(width*9/20, 24), width-fixed)
	return rowLayout{name: max(name, 8), cols: cols}
}

func (d compactDelegate) label(r gh.Repo) string {
	if d.bare {
		return r.Name()
	}
	return r.NameWithOwner
}

func (d compactDelegate) tags(r gh.Repo) string {
	var tags []string
	if r.IsPrivate {
		tags = append(tags, tag(privateBadge, iconPrivate, "private"))
	}
	if r.IsFork {
		tags = append(tags, tag(forkBadge, iconFork, "fork"))
	}
	if _, cloned := d.locals[r.NameWithOwner]; cloned {
		tags = append(tags, tag(publicBadge, iconLocal, "local"))
	}
	return strings.Join(tags, " ")
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
	if c.lang > 0 {
		c.lang = max(c.lang, lipgloss.Width("Language"))
	}

	return c
}

// header renders the column titles laid out exactly like Render lays out a row.
func (d compactDelegate) header(m list.Model) string {
	l := d.layout(m)
	row := ""
	if len(d.marked) > 0 {
		row = "  "
	}
	row += padRight(columnStyle.Render("Name"), l.name)
	if l.cols.tags > 0 {
		row += "  " + strings.Repeat(" ", l.cols.tags)
	}
	if l.cols.lang > 0 {
		row += "  " + padRight(columnStyle.Render("Language"), l.cols.lang)
	}
	return padRight(row, m.Width())
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
