package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/sahilm/fuzzy"
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
	locals map[string]string // "owner/repo" → working copy already on disk
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

	label := r.NameWithOwner
	if d.bare {
		label = r.Name()
	}
	cursor := "  "
	name := lipgloss.NewStyle().Foreground(colFg).Render(label)
	if index == m.Index() {
		cursor = selectedStyle.Render("❯ ")
		name = selectedStyle.Render(label)
	}

	meta := ""
	if _, cloned := d.locals[r.NameWithOwner]; cloned {
		meta += dimStyle.Render(" · ") + publicBadge.Render("local")
	}
	if r.IsPrivate {
		meta += dimStyle.Render(" · ") + privateBadge.Render("private")
	}
	if r.Language != "" {
		meta += dimStyle.Render(" · " + r.Language)
	}
	if r.StargazerCount > 0 {
		meta += dimStyle.Render(fmt.Sprintf(" · ★%d", r.StargazerCount))
	}

	fmt.Fprint(w, truncate(cursor+name+meta, m.Width()))
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

// filterRepos returns the list items matching query, fuzzy-ranked by relevance.
// An empty query yields every repo in its original order.
func filterRepos(repos []gh.Repo, query string) []list.Item {
	if query == "" {
		return repoItems(repos)
	}
	hay := make([]string, len(repos))
	for i, r := range repos {
		hay[i] = r.NameWithOwner + " " + r.Description
	}
	matches := fuzzy.Find(query, hay)
	items := make([]list.Item, 0, len(matches))
	for _, m := range matches {
		items = append(items, repoItem{repo: repos[m.Index]})
	}
	return items
}
