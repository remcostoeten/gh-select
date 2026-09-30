package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

func openTree(t *testing.T) *App {
	t.Helper()
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionSparseClone))
	return send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{
		{Path: "docs", Type: "tree"},
		{Path: "docs/my file.md", Type: "blob"},
	}}})
}

func TestTreeLinks(t *testing.T) {
	a := openTree(t)
	tr := a.tree
	if got := tr.webURL("docs/my file.md", false, "HEAD"); got != "https://github.com/remcostoeten/alpha/blob/HEAD/docs/my%20file.md" {
		t.Errorf("webURL = %q", got)
	}
	if got := tr.webURL("docs", true, "abc123"); got != "https://github.com/remcostoeten/alpha/tree/abc123/docs" {
		t.Errorf("folder webURL = %q", got)
	}
	tr.ref = "feature/x"
	if got := tr.rawURL("docs/my file.md"); got != "https://raw.githubusercontent.com/remcostoeten/alpha/feature/x/docs/my%20file.md" {
		t.Errorf("rawURL = %q", got)
	}
}

func TestTreeSwitchRef(t *testing.T) {
	a := openTree(t)
	a = send(t, a, key("space"))
	a = send(t, a, refsLoadedMsg{branches: []string{"main", "dev"}, tags: []string{"v1.0.0"}})
	if a.tree.refPicker == nil || !strings.Contains(a.View(), "branches & tags · 3") {
		t.Fatalf("ref picker not shown:\n%s", a.View())
	}
	a = send(t, a, key("v1"))
	a = send(t, a, key("enter"))
	if a.tree.ref != "v1.0.0" || !a.tree.loading || len(a.tree.selected) != 0 {
		t.Fatalf("after switch: ref=%q loading=%v selected=%d", a.tree.ref, a.tree.loading, len(a.tree.selected))
	}

	a = send(t, a, treeLoadedMsg{tree: &gh.Tree{Entries: []gh.TreeEntry{{Path: "src", Type: "tree"}}}})
	if !strings.Contains(a.tree.breadcrumb(), "@v1.0.0") {
		t.Errorf("breadcrumb = %q", a.tree.breadcrumb())
	}
	a = send(t, a, key("space"))
	a = send(t, a, key("c"))
	if a.Result.Action != ActionSparseClone || a.Result.Branch != "v1.0.0" {
		t.Errorf("sparse clone result = %+v", a.Result)
	}
}
