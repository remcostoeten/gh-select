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

func TestDiscoverSkipsHeavyDirsAndPrefersEarlierRoots(t *testing.T) {
	first, home := t.TempDir(), t.TempDir()
	writeClone(t, filepath.Join(first, "alpha"), "https://github.com/me/alpha.git")
	writeClone(t, filepath.Join(home, "code", "alpha"), "https://github.com/me/alpha.git")
	writeClone(t, filepath.Join(home, "code", "web", "node_modules", "dep"), "https://github.com/x/dep.git")
	writeClone(t, filepath.Join(home, "code", "beta", "inner"), "https://github.com/me/inner.git")
	writeClone(t, filepath.Join(home, ".cache", "tool"), "https://github.com/x/tool.git")
	writeClone(t, filepath.Join(home, ".config", "dotfiles"), "https://github.com/me/dotfiles.git")

	idx := Discover([]Root{{Dir: first, Depth: 2}, {Dir: home, Depth: 5}})
	if got := idx["me/alpha"]; got != filepath.Join(first, "alpha") {
		t.Errorf("alpha = %q, want the earlier root's copy", got)
	}
	if got := idx["me/dotfiles"]; got == "" {
		t.Error("dotfiles under .config not found")
	}
	if got := idx["me/inner"]; got == "" {
		t.Error("inner not found")
	}
	for _, name := range []string{"x/dep", "x/tool"} {
		if got, ok := idx[name]; ok {
			t.Errorf("%s found at %q, want skipped", name, got)
		}
	}
}

func TestDiscoverFindsWorktreesAndSubmodules(t *testing.T) {
	root := t.TempDir()
	main := filepath.Join(root, "main")
	writeClone(t, main, "https://github.com/me/main.git")

	wtGit := filepath.Join(main, ".git", "worktrees", "feature")
	if err := os.MkdirAll(wtGit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGit, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+wtGit+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	modGit := filepath.Join(root, "other", ".git", "modules", "lib")
	writeClone(t, filepath.Dir(filepath.Dir(modGit)), "https://github.com/me/other.git")
	if err := os.MkdirAll(modGit, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[remote \"origin\"]\n\turl = git@github.com:acme/lib.git\n"
	if err := os.WriteFile(filepath.Join(modGit, "config"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "vendored", "lib")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(sub, modGit)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: "+rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx := Discover([]Root{{Dir: filepath.Join(root, "work"), Depth: 2}, {Dir: filepath.Join(root, "vendored"), Depth: 2}})
	if got := idx["me/main"]; got != wt {
		t.Errorf("worktree = %q, want %q", got, wt)
	}
	if got := idx["acme/lib"]; got != sub {
		t.Errorf("submodule = %q, want %q", got, sub)
	}
}

func TestDefaultRootsPutsScanDirsFirst(t *testing.T) {
	roots := DefaultRoots("/clones", []string{"/extra/a", "/extra/b"})
	if len(roots) < 3 {
		t.Fatalf("roots = %v, want scan dirs then clone dir", roots)
	}
	for i, want := range []string{"/extra/a", "/extra/b", "/clones"} {
		if roots[i].Dir != want {
			t.Errorf("roots[%d] = %q, want %q", i, roots[i].Dir, want)
		}
	}
}

func TestLoadDropsVanishedClones(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "kept")
	writeClone(t, kept, "https://github.com/me/kept.git")
	path := filepath.Join(dir, "locals.json")
	if err := Save(path, map[string]string{"me/kept": kept, "me/gone": filepath.Join(dir, "gone")}); err != nil {
		t.Fatal(err)
	}
	idx, ok := Load(path)
	if !ok || len(idx) != 1 || idx["me/kept"] != kept {
		t.Errorf("Load = %v, %v", idx, ok)
	}
}
