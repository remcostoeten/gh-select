package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(t *testing.T, a *App, msg tea.Msg) *App {
	t.Helper()
	m, _ := a.Update(msg)
	return m.(*App)
}

var sampleRepos = []gh.Repo{
	{NameWithOwner: "remcostoeten/alpha", Description: "first", Language: "Go"},
	{NameWithOwner: "remcostoeten/beta", Description: "second", IsPrivate: true},
}

// Drive the full list → actions → copy path and assert the resulting action.
func TestSelectThenCopyName(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if strings.TrimSpace(a.View()) == "" {
		t.Fatal("list view rendered empty")
	}

	a = send(t, a, key("enter")) // select first repo
	if a.screen != screenActions {
		t.Fatalf("screen = %v, want actions", a.screen)
	}
	if a.selected.NameWithOwner != "remcostoeten/alpha" {
		t.Fatalf("selected = %q", a.selected.NameWithOwner)
	}
	if !strings.Contains(a.View(), "remcostoeten/alpha") {
		t.Error("actions view missing repo name")
	}

	a = send(t, a, key("4")) // copy name
	if a.Result.Action != ActionCopyName {
		t.Fatalf("action = %v, want ActionCopyName", a.Result.Action)
	}
	if a.Result.Repo.NameWithOwner != "remcostoeten/alpha" {
		t.Fatalf("result repo = %q", a.Result.Repo.NameWithOwner)
	}
}

// Typing on the list screen filters repos live, and esc clears the query.
func TestListTypeToSearch(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if got := len(a.list.Items()); got != 2 {
		t.Fatalf("initial items = %d, want 2", got)
	}

	a = send(t, a, key("beta")) // fuzzy-matches only remcostoeten/beta
	if a.query != "beta" {
		t.Fatalf("query = %q, want beta", a.query)
	}
	if got := len(a.list.Items()); got != 1 {
		t.Fatalf("filtered items = %d, want 1", got)
	}
	if it, ok := a.list.SelectedItem().(repoItem); !ok || it.repo.NameWithOwner != "remcostoeten/beta" {
		t.Fatalf("selected = %v, want remcostoeten/beta", a.list.SelectedItem())
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc}) // clear search
	if a.query != "" || len(a.list.Items()) != 2 {
		t.Fatalf("esc did not clear: query=%q items=%d", a.query, len(a.list.Items()))
	}

	// Selecting after a search must carry the matched repo into actions.
	a = send(t, a, key("alpha"))
	a = send(t, a, key("enter"))
	if a.screen != screenActions || a.selected.NameWithOwner != "remcostoeten/alpha" {
		t.Fatalf("after search-select: screen=%v selected=%q", a.screen, a.selected.NameWithOwner)
	}
}

// Tab toggles between filtering owned repos and GitHub search; the query and
// scope transitions must behave without touching the network.
func TestScopeToggle(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.scope != scopeMine || len(a.list.Items()) != 2 {
		t.Fatalf("initial: scope=%v items=%d", a.scope, len(a.list.Items()))
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab}) // -> GitHub scope, empty query
	if a.scope != scopeGitHub {
		t.Fatalf("scope = %v, want GitHub", a.scope)
	}
	if len(a.list.Items()) != 0 {
		t.Fatalf("GitHub scope with empty query should clear items, got %d", len(a.list.Items()))
	}

	a = send(t, a, key("torvalds/li")) // schedules a debounced search (not run here)
	if a.query != "torvalds/li" {
		t.Fatalf("query = %q", a.query)
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc}) // clears query, stays in GitHub
	if a.query != "" || a.scope != scopeGitHub {
		t.Fatalf("after clear: query=%q scope=%v", a.query, a.scope)
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc}) // empty query in GitHub -> back to mine
	if a.scope != scopeMine || len(a.list.Items()) != 2 {
		t.Fatalf("after back: scope=%v items=%d", a.scope, len(a.list.Items()))
	}
}

// A GitHub search response only applies when it still matches the live query.
func TestRemoteResultsStaleGuard(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab})
	a = send(t, a, key("linux"))

	found := []gh.Repo{{NameWithOwner: "torvalds/linux", Description: "kernel"}}
	a = send(t, a, remoteReposMsg{query: "linux", repos: found})
	if len(a.list.Items()) != 1 {
		t.Fatalf("matching results not applied: items=%d", len(a.list.Items()))
	}

	// A response for a query the user has since edited away must be ignored.
	a = send(t, a, remoteReposMsg{query: "old-query", repos: make([]gh.Repo, 5)})
	if len(a.list.Items()) != 1 {
		t.Fatalf("stale results applied: items=%d", len(a.list.Items()))
	}
}

// Drive list → actions → tree browser → folder select → confirm, asserting the
// sparse-clone result carries the chosen folders.
func TestTreeSelectAndConfirm(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter")) // -> actions
	a = send(t, a, key("3"))     // Browse & partial clone -> tree
	if a.screen != screenTree || a.tree == nil {
		t.Fatalf("did not enter tree screen")
	}

	// Simulate the tree having loaded.
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "src", Type: "tree"},
		{Path: "src/main.go", Type: "blob"},
		{Path: "docs", Type: "tree"},
		{Path: "README.md", Type: "blob"},
	}}})

	if a.tree.cwd == nil || len(a.tree.rows()) != 3 { // docs, src, README.md
		t.Fatalf("tree rows = %d, want 3", len(a.tree.rows()))
	}

	// Cursor starts on first row (docs, a dir). Mark it.
	a = send(t, a, key("space"))
	if len(a.tree.selectedFolders()) != 1 || a.tree.selectedFolders()[0] != "docs" {
		t.Fatalf("selected = %v, want [docs]", a.tree.selectedFolders())
	}

	// Confirm.
	a = send(t, a, key("c"))
	if a.Result.Action != ActionSparseClone {
		t.Fatalf("action = %v, want ActionSparseClone", a.Result.Action)
	}
	if len(a.Result.Folders) != 1 || a.Result.Folders[0] != "docs" {
		t.Fatalf("folders = %v, want [docs]", a.Result.Folders)
	}
}

// Individual files (not just folders) can be selected for a partial clone.
func TestTreeFileSelect(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, key("3")) // Browse & partial clone -> tree
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "docs", Type: "tree"},
		{Path: "README.md", Type: "blob"},
	}}})

	// rows: docs (dir), README.md (file). Move to the file and mark it.
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, key("space"))
	if got := a.tree.selectedFiles(); len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("selectedFiles = %v, want [README.md]", got)
	}
	if len(a.tree.selectedFolders()) != 0 {
		t.Fatalf("unexpected folders selected: %v", a.tree.selectedFolders())
	}

	a = send(t, a, key("c"))
	if a.Result.Action != ActionSparseClone {
		t.Fatalf("action = %v, want ActionSparseClone", a.Result.Action)
	}
	if len(a.Result.Files) != 1 || a.Result.Files[0] != "README.md" {
		t.Fatalf("result files = %v, want [README.md]", a.Result.Files)
	}
}

// The branch picker loads branches, filters them, and clones the chosen one.
func TestBranchPicker(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, key("2")) // Clone a specific branch…
	if a.screen != screenBranches || !a.branchLoading {
		t.Fatalf("did not enter branch screen: screen=%v loading=%v", a.screen, a.branchLoading)
	}

	a = send(t, a, branchesLoadedMsg{branches: []string{"main", "develop", "feat/x"}})
	if a.picker == nil || a.branchLoading {
		t.Fatal("branches not loaded into picker")
	}

	a = send(t, a, key("dev")) // fuzzy-filter to "develop"
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionClone || a.Result.Branch != "develop" {
		t.Fatalf("result = %v branch=%q, want clone of develop", a.Result.Action, a.Result.Branch)
	}
}

// Syntax highlighting wraps source in ANSI escapes without dropping content.
func TestHighlightPreview(t *testing.T) {
	out := renderPreview("main.go", []byte("package main\n\nfunc main() {}\n"))
	if !strings.Contains(out, "\x1b[") {
		t.Error("expected ANSI escape codes from highlighting")
	}
	if !strings.Contains(out, "main") {
		t.Error("highlighted output dropped source text")
	}
	if got := renderPreview("x.bin", []byte{0x00, 0x01}); !strings.Contains(got, "binary") {
		t.Errorf("binary guard missing: %q", got)
	}
}

// View() must never panic on any screen, including the tree at its root (a
// regression guard for the breadcrumb bug).
func TestViewsDoNotPanic(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	mustRender(t, a, "list")

	a = send(t, a, key("enter"))
	mustRender(t, a, "actions")

	a = send(t, a, key("3")) // Browse & partial clone -> tree
	mustRender(t, a, "tree-loading")

	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "src", Type: "tree"},
		{Path: "src/main.go", Type: "blob"},
	}}})
	mustRender(t, a, "tree-root") // breadcrumb at root must not panic

	a = send(t, a, key("enter")) // drill into src
	mustRender(t, a, "tree-nested")
}

func mustRender(t *testing.T, a *App, name string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked on %s screen: %v", name, r)
		}
	}()
	if a.View() == "" {
		t.Errorf("%s screen rendered empty", name)
	}
}

// Filtering narrows the directory listing and esc clears it.
func TestTreeFilter(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, key("3")) // Browse & partial clone -> tree
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "docs", Type: "tree"},
		{Path: "src", Type: "tree"},
		{Path: "scripts", Type: "tree"},
	}}})

	a = send(t, a, key("/"))  // enter filter mode
	a = send(t, a, key("sc")) // matches "scripts" only (not docs/src... "src" has no "sc")
	rows := a.tree.rows()
	if len(rows) != 1 || rows[0].name != "scripts" {
		t.Fatalf("filtered rows = %v, want [scripts]", names(rows))
	}
	a = send(t, a, key("enter")) // exit filter input, keep applied
	if a.tree.filtering {
		t.Error("still in filtering input mode after enter")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc}) // clear filter
	if a.tree.filter != "" || len(a.tree.rows()) != 3 {
		t.Fatalf("filter not cleared: filter=%q rows=%d", a.tree.filter, len(a.tree.rows()))
	}
}

func names(ns []*node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.name
	}
	return out
}

// Confirming with nothing selected must not produce a clone action.
func TestTreeConfirmEmpty(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, key("3")) // Browse & partial clone -> tree
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{{Path: "src", Type: "tree"}}}})
	a = send(t, a, key("c")) // confirm with no selection
	if a.Result.Action != ActionNone {
		t.Fatalf("action = %v, want ActionNone", a.Result.Action)
	}
	if !strings.Contains(a.tree.status, "select at least one") {
		t.Errorf("missing guidance status, got %q", a.tree.status)
	}
}
