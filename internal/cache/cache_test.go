package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/remcostoeten/gh-select/internal/gh"
)

func TestLoadMissingCache(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "nested"), time.Hour)
	if _, ok := c.Load(); ok {
		t.Fatal("Load on an empty directory reported a cache hit")
	}
}

func TestSaveCreatesDirAndRoundTrips(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "nested", "dir"), time.Hour)
	updated := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	want := []gh.Repo{
		{NameWithOwner: "me/dotfiles", IsOwner: true, UpdatedAt: updated},
		{NameWithOwner: "acme/fork", IsFork: true, Parent: "upstream/lib"},
	}
	if err := c.Save(want); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Load()
	if !ok || !got.Fresh {
		t.Fatalf("got ok=%v fresh=%v, want a fresh hit", ok, got.Fresh)
	}
	if len(got.Repos) != 2 || got.Repos[0].NameWithOwner != "me/dotfiles" || !got.Repos[0].UpdatedAt.Equal(updated) || got.Repos[1].Parent != "upstream/lib" {
		t.Errorf("round trip changed the repos: %+v", got.Repos)
	}
	if _, err := os.Stat(c.file + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("Save left its temp file behind: %v", err)
	}
}

func TestStaleCacheIsStillServed(t *testing.T) {
	c := New(t.TempDir(), time.Hour)
	if err := c.Save([]gh.Repo{{NameWithOwner: "me/old"}}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(c.file, old, old); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Load()
	if !ok || got.Fresh || len(got.Repos) != 1 {
		t.Fatalf("got ok=%v fresh=%v repos=%d, want a stale hit with 1 repo", ok, got.Fresh, len(got.Repos))
	}
	if got.Age < time.Hour {
		t.Errorf("Age = %v, want at least the 2h set on the file", got.Age)
	}
}

func TestCorruptCacheIsAMiss(t *testing.T) {
	c := New(t.TempDir(), time.Hour)
	if err := os.WriteFile(c.file, []byte(`{"truncated`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Load(); ok {
		t.Fatal("a corrupt cache file was reported as a hit")
	}
}

func TestNamedCachesDoNotShareData(t *testing.T) {
	repos := New(t.TempDir(), time.Hour)
	starred := repos.Named("starred")
	if err := repos.Save([]gh.Repo{{NameWithOwner: "me/mine"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := starred.Load(); ok {
		t.Fatal("the starred cache read the repos cache file")
	}
}

func TestOlderSchemaFileIsIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "repos.json"), []byte(`[{"nameWithOwner":"me/v1"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := New(dir, time.Hour).Load(); ok {
		t.Fatal("a cache file from before the schema bump was loaded")
	}
}
