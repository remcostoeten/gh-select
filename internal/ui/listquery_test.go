package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

var queryRepos = []gh.Repo{
	{NameWithOwner: "me/alpha", Language: "Go", StargazerCount: 5, IsOwner: true},
	{NameWithOwner: "me/beta", Language: "TypeScript", StargazerCount: 50, IsPrivate: true, IsOwner: true},
	{NameWithOwner: "org/gamma", Language: "Go", StargazerCount: 500},
}

func repoNames(items []list.Item) string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.(repoItem).repo.Name())
	}
	return strings.Join(out, ",")
}

func itemNames(a *App) string { return repoNames(a.list.Items()) }

func TestFilterTokens(t *testing.T) {
	locals := map[string]string{"org/gamma": "/src/gamma"}
	cases := map[string]string{
		"lang:go":             "alpha,gamma",
		"language:typescript": "beta",
		"is:private":          "beta",
		"is:public lang:go":   "alpha,gamma",
		"is:local":            "gamma",
		"is:mine lang:go":     "alpha",
		"is:bogus":            "",
	}
	for q, want := range cases {
		if n := repoNames(listRepos(queryRepos, q, locals, sortDefault, true)); n != want {
			t.Errorf("%q = %q, want %q", q, n, want)
		}
	}
}

func TestGitHubQueryTranslation(t *testing.T) {
	if got := githubQuery("cli lang:go is:private is:local"); got != "cli language:go is:private" {
		t.Errorf("githubQuery = %q", got)
	}
}

func TestSortCycle(t *testing.T) {
	a := NewApp(nil, queryRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if got := itemNames(a); got != "alpha,beta,gamma" {
		t.Fatalf("default order = %q", got)
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := itemNames(a); got != "gamma,beta,alpha" || !strings.Contains(a.listPanelTitle(), "by stars") {
		t.Fatalf("by stars = %q, title %q", got, a.listPanelTitle())
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := itemNames(a); got != "alpha,beta,gamma" || !strings.Contains(a.listPanelTitle(), "by name") {
		t.Fatalf("by name = %q", got)
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlS})
	if a.sortBy != sortDefault {
		t.Errorf("sort did not wrap to default: %v", a.sortBy)
	}
}

func TestStarredScope(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a.SetStarred([]gh.Repo{{NameWithOwner: "charmbracelet/bubbletea", Language: "Go"}}, nil)
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab})
	if a.scope != scopeStarred || itemNames(a) != "bubbletea" {
		t.Fatalf("starred: scope=%v items=%q", a.scope, itemNames(a))
	}
	if !strings.Contains(a.list.View(), "charmbracelet/bubbletea") {
		t.Error("starred rows should keep the owner prefix")
	}
	a = send(t, a, starredLoadedMsg{repos: []gh.Repo{{NameWithOwner: "a/one"}, {NameWithOwner: "b/two"}}})
	if itemNames(a) != "one,two" {
		t.Errorf("fresh starred list not applied: %q", itemNames(a))
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.scope != scopeMine {
		t.Errorf("esc from starred should return to my repos, scope=%v", a.scope)
	}
}
