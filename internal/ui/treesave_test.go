package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/remcostoeten/gh-select/internal/gh"
)

func TestWriteNewNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "notes.md")
	first, err := writeNew(path, []byte("one"))
	if err != nil || first != path {
		t.Fatalf("first write: %q, %v", first, err)
	}
	second, err := writeNew(path, []byte("two"))
	if err != nil || second != filepath.Join(dir, "sub", "notes-1.md") {
		t.Fatalf("second write: %q, %v", second, err)
	}
	if b, _ := os.ReadFile(path); string(b) != "one" {
		t.Errorf("original overwritten: %q", b)
	}
}

func TestSaveTargetsExpandsFolders(t *testing.T) {
	tr := &treeModel{selected: map[string]*node{}}
	tr.root = buildTree([]gh.TreeEntry{
		{Path: "docs", Type: "tree"},
		{Path: "docs/a.md", Type: "blob"},
		{Path: "docs/deep", Type: "tree"},
		{Path: "docs/deep/b.md", Type: "blob"},
		{Path: "main.go", Type: "blob"},
	})
	tr.cwd = tr.root

	if got := tr.saveTargets(); !reflect.DeepEqual(got, []string{"docs/a.md", "docs/deep/b.md"}) {
		t.Errorf("highlighted folder: %v", got)
	}

	tr.selected["main.go"] = tr.root.byName["main.go"]
	tr.selected["docs/deep"] = tr.root.byName["docs"].byName["deep"]
	if got := tr.saveTargets(); !reflect.DeepEqual(got, []string{"docs/deep/b.md", "main.go"}) {
		t.Errorf("selection: %v", got)
	}
}
