package ui

import (
	"errors"
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

// actionKey is the number shortcut for an action in the menu as currently
// built, so tests stay correct when the menu's composition changes.
func actionKey(t *testing.T, a *App, act ActionType) tea.KeyMsg {
	t.Helper()
	for _, it := range a.actions {
		if it.act == act {
			return key(it.key)
		}
	}
	t.Fatalf("action %v not in menu", act)
	return tea.KeyMsg{}
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

	a = send(t, a, actionKey(t, a, ActionCopyName))
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

	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab}) // -> starred
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
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab}) // -> starred
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
	a = send(t, a, key("enter"))                       // -> actions
	a = send(t, a, actionKey(t, a, ActionSparseClone)) // -> tree
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
	a = send(t, a, actionKey(t, a, ActionSparseClone)) // -> tree
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
	a = send(t, a, actionKey(t, a, ActionCloneBranch))
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
	out := renderPreview("main.go", []byte("package main\n\nfunc main() {}\n"), 80)
	if !strings.Contains(out, "\x1b[") {
		t.Error("expected ANSI escape codes from highlighting")
	}
	if !strings.Contains(out, "main") {
		t.Error("highlighted output dropped source text")
	}
	if got := renderPreview("x.bin", []byte{0x00, 0x01}, 80); !strings.Contains(got, "binary") {
		t.Errorf("binary guard missing: %q", got)
	}
}

// Markdown previews are rendered as prose: the syntax markers are consumed and
// the text survives.
func TestMarkdownPreview(t *testing.T) {
	out := renderPreview("README.md", []byte("# Title\n\nSome **bold** text.\n"), 60)
	if strings.Contains(out, "**bold**") {
		t.Errorf("markdown left unrendered: %q", out)
	}
	for _, want := range []string{"Title", "bold", "text"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered markdown dropped %q: %q", want, out)
		}
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

	a = send(t, a, actionKey(t, a, ActionSparseClone)) // -> tree
	mustRender(t, a, "tree-loading")

	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "src", Type: "tree"},
		{Path: "src/main.go", Type: "blob"},
	}}})
	mustRender(t, a, "tree-root") // breadcrumb at root must not panic

	a = send(t, a, key("enter")) // drill into src
	mustRender(t, a, "tree-nested")

	a.screen = screenActions
	a = send(t, a, actionKey(t, a, ActionViewReadme))
	mustRender(t, a, "readme-loading")

	a = send(t, a, readmeLoadedMsg{path: "README.md", content: []byte("# Hi\n\ntext\n")})
	mustRender(t, a, "readme")

	a = send(t, a, key("esc"))
	if a.screen != screenActions {
		t.Errorf("esc from readme went to screen %v, want actions", a.screen)
	}
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
	a = send(t, a, actionKey(t, a, ActionSparseClone)) // -> tree
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

// Background refresh progress renders in the reserved status line, which can
// never be clipped by a full list body.
func TestRefreshStatusVisible(t *testing.T) {
	a := NewApp(nil, sampleRepos, true, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(a.View(), "Refreshing repositories") {
		t.Error("refresh status not visible in view")
	}
}

// A failed branch fetch must drop back to the action menu with the error shown.
func TestBranchErrorSurfaced(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionCloneBranch))
	a = send(t, a, branchesLoadedMsg{err: errors.New("boom")})
	if a.screen != screenActions {
		t.Fatalf("screen = %v, want actions", a.screen)
	}
	if !strings.Contains(a.View(), "Couldn't load branches") {
		t.Error("branch error not visible on actions screen")
	}
}

// An in-flight GitHub search keeps the previous results on screen and reports
// progress via the status line instead of blanking the list.
func TestSearchKeepsStaleResults(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab}) // -> starred
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab})
	a = send(t, a, key("linux"))
	a = send(t, a, remoteReposMsg{query: "linux", repos: []gh.Repo{{NameWithOwner: "torvalds/linux"}}})

	a = send(t, a, key("x")) // query is now "linuxx"; schedules a debounce
	a = send(t, a, debounceMsg{seq: a.searchSeq})
	if !a.searching {
		t.Fatal("debounce did not start a search")
	}
	if len(a.list.Items()) != 1 {
		t.Errorf("stale results dropped during search: items=%d", len(a.list.Items()))
	}
	if !strings.Contains(a.View(), "Searching GitHub") {
		t.Error("search progress not visible in view")
	}
}

// My-repos rows drop the owner prefix; GitHub search rows keep it.
func TestListOwnerPrefixByScope(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if v := a.list.View(); strings.Contains(v, "remcostoeten/alpha") {
		t.Error("my-repos list still shows the owner prefix")
	} else if !strings.Contains(v, "alpha") {
		t.Error("my-repos list lost the repo name")
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab}) // -> starred
	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab})
	a = send(t, a, key("linux"))
	a = send(t, a, remoteReposMsg{query: "linux", repos: []gh.Repo{{NameWithOwner: "torvalds/linux"}}})
	if !strings.Contains(a.list.View(), "torvalds/linux") {
		t.Error("GitHub search list dropped the owner prefix")
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyTab})
	if strings.Contains(a.list.View(), "remcostoeten/alpha") {
		t.Error("owner prefix returned after switching back to my repos")
	}
}

// ? toggles the tree's full-keymap overlay.
func TestTreeHelpToggle(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionSparseClone))
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{{Path: "src", Type: "tree"}}}})

	a = send(t, a, key("?"))
	if !a.tree.helpVisible || !strings.Contains(a.View(), "Tree browser keys") {
		t.Fatal("help overlay not shown")
	}
	a = send(t, a, key("?"))
	if a.tree.helpVisible {
		t.Error("help overlay did not close")
	}
}

// Confirming with nothing selected must not produce a clone action.
func TestTreeConfirmEmpty(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionSparseClone)) // -> tree
	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{{Path: "src", Type: "tree"}}}})
	a = send(t, a, key("c")) // confirm with no selection
	if a.Result.Action != ActionNone {
		t.Fatalf("action = %v, want ActionNone", a.Result.Action)
	}
	if !strings.Contains(a.tree.status, "select at least one") {
		t.Errorf("missing guidance status, got %q", a.tree.status)
	}
}

// A repo already on disk leads with open/pull and is badged in the list.
func TestLocalCloneAwareMenu(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a.SetLocalClones(map[string]string{"remcostoeten/alpha": "/src/alpha"})
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if !strings.Contains(a.View(), iconLocal) {
		t.Error("list view missing the local badge")
	}

	a = send(t, a, key("enter"))
	if a.actions[0].act != ActionOpenEditor || a.actions[1].act != ActionPull {
		t.Fatalf("menu leads with %v, %v", a.actions[0].act, a.actions[1].act)
	}
	if !strings.Contains(a.View(), "/src/alpha") {
		t.Error("actions view missing the local path")
	}

	a = send(t, a, actionKey(t, a, ActionPull))
	if a.Result.Action != ActionPull || a.Result.LocalPath != "/src/alpha" {
		t.Fatalf("result = %v %q", a.Result.Action, a.Result.LocalPath)
	}
}

// Without a local clone the menu offers clone variants, never pull.
func TestMenuWithoutLocalClone(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	if a.actions[0].act != ActionClone {
		t.Fatalf("menu leads with %v, want ActionClone", a.actions[0].act)
	}
	for _, it := range a.actions {
		if it.act == ActionPull {
			t.Fatal("pull offered for a repo that isn't cloned")
		}
	}
}

// print-path mode skips the action menu and returns the path straight away.
func TestPrintPathMode(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a.SetLocalClones(map[string]string{"remcostoeten/alpha": "/src/alpha"})
	a.SetPrintPath(true)
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))

	if a.screen == screenActions {
		t.Error("print-path mode opened the action menu")
	}
	if a.Result.Action != ActionPrintPath || a.Result.LocalPath != "/src/alpha" {
		t.Fatalf("result = %v %q", a.Result.Action, a.Result.LocalPath)
	}
}

func TestListHelpToggle(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("?"))
	if !a.listHelp || !strings.Contains(a.View(), "Repository list keys") {
		t.Fatal("help overlay not shown")
	}
	a = send(t, a, key("?"))
	if a.listHelp {
		t.Fatal("help overlay did not close")
	}
	a = send(t, a, key("x"))
	a = send(t, a, key("?"))
	if a.listHelp || a.query != "x?" {
		t.Errorf("? mid-query: help=%v query=%q", a.listHelp, a.query)
	}
}

func TestFitHintsDropsWholeHints(t *testing.T) {
	keys := keyHint([2]string{"enter", "select"}, [2]string{"tab", "search GitHub"}, [2]string{"^C", "quit"})
	got := fitHints(keys, 30)
	if strings.Contains(got, "tab") || !strings.Contains(got, "quit") || strings.Contains(got, "…") {
		t.Errorf("fitHints = %q", got)
	}
}

func TestShortCount(t *testing.T) {
	for n, want := range map[int]string{12: "12", 1000: "1k", 1234: "1.2k", 45678: "45k", 2_300_000: "2.3M"} {
		if got := shortCount(n); got != want {
			t.Errorf("shortCount(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestBackspaceLeavesActionMenu(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, tea.KeyMsg{Type: tea.KeyBackspace})
	if a.screen != screenList {
		t.Fatalf("screen = %v, want list", a.screen)
	}
}

func TestBackspaceLeavesPickerOnlyWhenQueryIsEmpty(t *testing.T) {
	p := newPicker([]string{"main", "dev"})
	p.handleKey("d", []rune("d"), true)
	if got := p.handleKey("backspace", nil, false); got != pickerContinue || p.query != "" {
		t.Fatalf("first backspace: outcome=%v query=%q", got, p.query)
	}
	if got := p.handleKey("backspace", nil, false); got != pickerBack {
		t.Fatalf("second backspace: outcome=%v, want back", got)
	}
}
