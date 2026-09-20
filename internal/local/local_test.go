package local

import (
	"os"
	"path/filepath"
	"testing"
)

// writeClone fakes a working copy: a .git/config carrying a remote URL.
func writeClone(t *testing.T, dir, remote string) {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[remote \"origin\"]\n\turl = " + remote + "\n\tfetch = +refs/heads/*\n"
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexFindsFlatAndNestedClones(t *testing.T) {
	root := t.TempDir()
	writeClone(t, filepath.Join(root, "alpha"), "https://github.com/remcostoeten/alpha.git")
	writeClone(t, filepath.Join(root, "org", "beta"), "git@github.com:acme/beta.git")
	if err := os.MkdirAll(filepath.Join(root, "notrepo"), 0o755); err != nil {
		t.Fatal(err)
	}

	idx := Index(root)
	if got := idx["remcostoeten/alpha"]; got != filepath.Join(root, "alpha") {
		t.Errorf("alpha = %q", got)
	}
	if got := idx["acme/beta"]; got != filepath.Join(root, "org", "beta") {
		t.Errorf("beta = %q", got)
	}
	if len(idx) != 2 {
		t.Errorf("index has %d entries, want 2: %v", len(idx), idx)
	}
}

func TestIndexIgnoresMissingRoot(t *testing.T) {
	if idx := Index(filepath.Join(t.TempDir(), "nope")); len(idx) != 0 {
		t.Errorf("index = %v, want empty", idx)
	}
	if idx := Index(""); len(idx) != 0 {
		t.Errorf("index = %v, want empty", idx)
	}
}

func TestDestinationAndExists(t *testing.T) {
	root := t.TempDir()
	if got := Destination("", "acme/beta"); got != "" {
		t.Errorf("Destination with no root = %q, want empty", got)
	}
	if got := Destination(root, "acme/beta"); got != filepath.Join(root, "beta") {
		t.Errorf("Destination = %q", got)
	}
	if Exists(filepath.Join(root, "beta")) {
		t.Error("Exists reported a directory that isn't there")
	}
	writeClone(t, filepath.Join(root, "beta"), "https://github.com/acme/beta")
	if !Exists(filepath.Join(root, "beta")) {
		t.Error("Exists missed an existing working copy")
	}
}
