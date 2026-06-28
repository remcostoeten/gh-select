// Package cache persists the repository list to disk so the UI can render
// instantly (stale-while-revalidate) while fresh data is fetched in the
// background.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/remcostoeten/gh-select/internal/gh"
)

// Cache reads and writes the repo list under a directory.
type Cache struct {
	dir  string
	file string
	ttl  time.Duration
}

// New returns a Cache rooted at dir with the given freshness window.
func New(dir string, ttl time.Duration) *Cache {
	return &Cache{dir: dir, file: filepath.Join(dir, "repos.json"), ttl: ttl}
}

// Entry is the cached payload plus the age of the data on disk.
type Entry struct {
	Repos []gh.Repo
	Age   time.Duration
	Fresh bool
}

// Load returns the cached repos and whether they are still fresh. ok is false
// when nothing is cached yet.
func (c *Cache) Load() (Entry, bool) {
	info, err := os.Stat(c.file)
	if err != nil {
		return Entry{}, false
	}
	data, err := os.ReadFile(c.file)
	if err != nil {
		return Entry{}, false
	}
	var repos []gh.Repo
	if err := json.Unmarshal(data, &repos); err != nil {
		return Entry{}, false
	}
	age := time.Since(info.ModTime())
	return Entry{Repos: repos, Age: age, Fresh: age < c.ttl}, true
}

// Save atomically writes repos to the cache file.
func (c *Cache) Save(repos []gh.Repo) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(repos)
	if err != nil {
		return err
	}
	tmp := c.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.file)
}
