package ui

import "testing"

import "github.com/remcostoeten/gh-select/internal/gh"

func TestBuildTree(t *testing.T) {
	entries := []gh.TreeEntry{
		{Path: "README.md", Type: "blob"},
		{Path: "src", Type: "tree"},
		{Path: "src/main.go", Type: "blob"},
		{Path: "src/util", Type: "tree"},
		{Path: "src/util/x.go", Type: "blob"},
		{Path: ".github/workflows/ci.yml", Type: "blob"}, // intermediate dirs implied
	}
	root := buildTree(entries)

	// Root children: .github (dir), src (dir), README.md (file) — dirs first, sorted.
	if got := len(root.children); got != 3 {
		t.Fatalf("root children = %d, want 3", got)
	}
	if root.children[0].name != ".github" || !root.children[0].isDir {
		t.Errorf("first child = %q (dir=%v), want .github dir", root.children[0].name, root.children[0].isDir)
	}
	if root.children[2].name != "README.md" || root.children[2].isDir {
		t.Errorf("last child = %q (dir=%v), want README.md file", root.children[2].name, root.children[2].isDir)
	}

	// Implied intermediate dir gets the right full path.
	gh := root.byName[".github"]
	if gh == nil || gh.path != ".github" {
		t.Fatalf(".github node missing or wrong path")
	}
	wf := gh.byName["workflows"]
	if wf == nil || wf.path != ".github/workflows" || !wf.isDir {
		t.Fatalf("workflows node missing/wrong: %+v", wf)
	}

	// Nested file path is correct.
	src := root.byName["src"]
	util := src.byName["util"]
	if util.byName["x.go"].path != "src/util/x.go" {
		t.Errorf("nested path = %q, want src/util/x.go", util.byName["x.go"].path)
	}
}
