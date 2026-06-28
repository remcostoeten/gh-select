package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/sahilm/fuzzy"
)

// repoItem adapts gh.Repo to the bubbles list item interface.
type repoItem struct{ repo gh.Repo }

func (i repoItem) Title() string { return i.repo.NameWithOwner }

func (i repoItem) Description() string {
	badge := publicBadge.Render("public")
	if i.repo.IsPrivate {
		badge = privateBadge.Render("private")
	}
	parts := badge
	if i.repo.Language != "" {
		parts += dimStyle.Render(" · " + i.repo.Language)
	}
	if i.repo.StargazerCount > 0 {
		parts += dimStyle.Render(fmt.Sprintf(" · ★%d", i.repo.StargazerCount))
	}
	desc := i.repo.Description
	if desc == "" {
		desc = "No description"
	}
	return parts + dimStyle.Render(" · "+desc)
}

// FilterValue lets the built-in filter match on name and description.
func (i repoItem) FilterValue() string {
	return i.repo.NameWithOwner + " " + i.repo.Description
}

func newRepoList(repos []gh.Repo, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(colPink).BorderForeground(colPink)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(colHL).BorderForeground(colPink)

	l := list.New(repoItems(repos), delegate, width, height)
	// The persistent chrome (header/footer) and our own always-on type-to-search
	// replace the list's built-in title, status bar, help and filter.
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
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
