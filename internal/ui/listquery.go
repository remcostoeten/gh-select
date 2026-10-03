package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/sahilm/fuzzy"
)

type sortOrder int

const (
	sortDefault sortOrder = iota
	sortUpdated
	sortStars
	sortName
)

// sortCycle is the order ctrl+s steps through per scope. The default is the
// order the source already has: recently updated for your repos, recently
// starred for stars, best match for a GitHub search.
func sortCycle(s searchScope) []sortOrder {
	if s == scopeMine {
		return []sortOrder{sortDefault, sortStars, sortName}
	}
	return []sortOrder{sortDefault, sortUpdated, sortStars, sortName}
}

func sortLabel(o sortOrder, s searchScope) string {
	switch o {
	case sortUpdated:
		return "updated"
	case sortStars:
		return "stars"
	case sortName:
		return "name"
	}
	switch s {
	case scopeStarred:
		return "starred"
	case scopeGitHub:
		return "best match"
	}
	return "updated"
}

func nextSort(o sortOrder, s searchScope) sortOrder {
	cycle := sortCycle(s)
	for i, c := range cycle {
		if c == o {
			return cycle[(i+1)%len(cycle)]
		}
	}
	return sortDefault
}

func sortRepos(repos []gh.Repo, o sortOrder) {
	switch o {
	case sortUpdated:
		sort.SliceStable(repos, func(i, j int) bool { return repos[i].UpdatedAt.After(repos[j].UpdatedAt) })
	case sortStars:
		sort.SliceStable(repos, func(i, j int) bool { return repos[i].StargazerCount > repos[j].StargazerCount })
	case sortName:
		sort.SliceStable(repos, func(i, j int) bool {
			return strings.ToLower(repos[i].Name()) < strings.ToLower(repos[j].Name())
		})
	}
}

// listQuery is the search text split into free text and filter tokens:
// lang:go (or language:go), is:private, is:public, is:fork, is:source,
// is:local and is:mine.
type listQuery struct {
	text string
	lang string
	is   map[string]bool
}

var isFilters = map[string]bool{"private": true, "public": true, "fork": true, "source": true, "local": true, "mine": true}

func parseQuery(q string) listQuery {
	out := listQuery{is: map[string]bool{}}
	var words []string
	for _, f := range strings.Fields(q) {
		key, val, ok := strings.Cut(strings.ToLower(f), ":")
		switch {
		case ok && (key == "lang" || key == "language") && val != "":
			out.lang = val
		case ok && key == "is" && isFilters[val]:
			out.is[val] = true
		default:
			words = append(words, f)
		}
	}
	out.text = strings.Join(words, " ")
	return out
}

func (q listQuery) hasFilters() bool { return q.lang != "" || len(q.is) > 0 }

func (q listQuery) matches(r gh.Repo, locals map[string]string) bool {
	if q.lang != "" && strings.ToLower(r.Language) != q.lang {
		return false
	}
	if q.is["private"] && !r.IsPrivate || q.is["public"] && r.IsPrivate || q.is["mine"] && !r.IsOwner ||
		q.is["fork"] && !r.IsFork || q.is["source"] && r.IsFork {
		return false
	}
	if _, cloned := locals[r.NameWithOwner]; q.is["local"] && !cloned {
		return false
	}
	return true
}

// githubQuery rewrites a list query into GitHub search syntax. is:local and
// is:mine have no GitHub equivalent and are applied to the results instead.
func githubQuery(q string) string {
	parsed := parseQuery(q)
	parts := []string{}
	if parsed.text != "" {
		parts = append(parts, parsed.text)
	}
	if parsed.lang != "" {
		parts = append(parts, "language:"+parsed.lang)
	}
	for _, v := range []string{"private", "public"} {
		if parsed.is[v] {
			parts = append(parts, "is:"+v)
		}
	}
	switch {
	case parsed.is["fork"]:
		parts = append(parts, "fork:only")
	case parsed.is["source"]:
		parts = append(parts, "fork:false")
	}
	return strings.Join(parts, " ")
}

// listRepos narrows repos to the query and orders them: fuzzy-ranked by the
// free text under the default order, otherwise by the chosen sort.
func listRepos(repos []gh.Repo, query string, locals map[string]string, o sortOrder, fuzzyRank bool) []list.Item {
	q := parseQuery(query)
	kept := make([]gh.Repo, 0, len(repos))
	for _, r := range repos {
		if q.matches(r, locals) {
			kept = append(kept, r)
		}
	}
	if fuzzyRank && q.text != "" {
		hay := make([]string, len(kept))
		for i, r := range kept {
			hay[i] = r.NameWithOwner + " " + r.Description
		}
		matches := fuzzy.Find(q.text, hay)
		ranked := make([]gh.Repo, len(matches))
		for i, m := range matches {
			ranked[i] = kept[m.Index]
		}
		kept = ranked
	}
	sortRepos(kept, o)
	return repoItems(kept)
}
