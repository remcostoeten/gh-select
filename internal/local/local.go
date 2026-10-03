// Package local discovers repositories already cloned on disk, so the UI can
// mark them and offer "open" instead of a redundant clone.
package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// maxDepth bounds the scan at owner/repo layouts (~/src/github/owner/repo),
// which is one level deeper than the flat ~/src/repo case.
const maxDepth = 2

// homeDepth reaches layouts like ~/code/work/client/repo when searching home.
const homeDepth = 5

// dirBudget caps how many directories one Discover call reads, so a huge home
// costs a bounded amount of time and IO.
const dirBudget = 40000

// chunk is how many entries are read from a directory at a time; maxEntries
// gives up on a directory that big, which is data rather than projects.
const (
	chunk      = 256
	maxEntries = 4096
)

// skipDirs never hold clones worth listing and are often enormous.
var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, "target": true, "dist": true, "build": true,
	"out": true, "venv": true, "__pycache__": true, "site-packages": true,
	"Library": true, "Applications": true, "snap": true, "Trash": true, "pkg": true, "mod": true, "deps": true, "bower_components": true,
}

// hiddenAllowed are dot directories that commonly hold clones, such as dotfiles.
var hiddenAllowed = map[string]bool{".config": true, ".dotfiles": true}

// remotePattern extracts "owner/repo" from any github.com remote URL —
// https://github.com/owner/repo.git, git@github.com:owner/repo.git, or the
// ssh:// form.
var remotePattern = regexp.MustCompile(`github\.com[:/]+([^/\s]+)/([^/\s]+?)(?:\.git)?\s*$`)

// Root is a directory to search and how many levels below it to look.
type Root struct {
	Dir   string
	Depth int
}

// Index scans root for git working copies and maps "owner/repo" to the
// directory holding it. An unreadable or unset root yields an empty index
// rather than an error: local awareness is an enhancement, never a
// precondition for using the app.
func Index(root string) map[string]string {
	return Discover([]Root{{Dir: root, Depth: maxDepth}})
}

// DefaultRoots are the places Discover searches: any extra scan directories,
// the clone directory, the current directory and the home directory, in that
// order of preference. The built-in roots stay as a fallback behind the extra
// ones, so setting scan directories never hides clones found before.
func DefaultRoots(cloneDir string, scanDirs []string) []Root {
	var roots []Root
	for _, dir := range scanDirs {
		roots = append(roots, Root{Dir: dir, Depth: homeDepth})
	}
	if cloneDir != "" {
		roots = append(roots, Root{Dir: cloneDir, Depth: maxDepth})
	}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, Root{Dir: cwd, Depth: maxDepth})
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, Root{Dir: home, Depth: homeDepth})
	}
	return roots
}

// Discover walks each root for git working copies with a github.com remote
// and maps "owner/repo" to a directory holding it, preferring earlier roots
// and, within a root, the most recently used copy. It skips
// dependency, cache and media trees, never descends into a working copy, and
// stops after dirBudget directories, so memory stays flat on any disk.
func Discover(roots []Root) map[string]string {
	found := map[string]string{}
	seen := map[string]bool{}
	budget := dirBudget
	for _, r := range roots {
		if r.Dir == "" {
			continue
		}
		inRoot := map[string]string{}
		scan(filepath.Clean(r.Dir), r.Depth, inRoot, seen, &budget)
		for name, dir := range inRoot {
			if _, dup := found[name]; !dup {
				found[name] = dir
			}
		}
	}
	return found
}

func scan(dir string, depth int, found map[string]string, seen map[string]bool, budget *int) {
	if seen[dir] || *budget <= 0 {
		return
	}
	seen[dir] = true
	*budget--
	if isRepo(dir) {
		if name := repoAt(dir); name != "" {
			if prev, dup := found[name]; !dup || lastUsed(dir).After(lastUsed(prev)) {
				found[name] = dir
			}
		}
		return
	}
	if depth == 0 {
		return
	}
	for _, sub := range subdirs(dir) {
		scan(filepath.Join(dir, sub), depth-1, found, seen, budget)
	}
}

// subdirs lists the directories under dir worth descending into, reading in
// chunks and giving up on directories with more than maxEntries entries.
func subdirs(dir string) []string {
	f, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer f.Close()
	var names []string
	read := 0
	for read < maxEntries {
		entries, err := f.ReadDir(chunk)
		read += len(entries)
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || skipDirs[name] || (strings.HasPrefix(name, ".") && !hiddenAllowed[name]) {
				continue
			}
			names = append(names, name)
		}
		if err != nil {
			return names
		}
	}
	return nil
}

// lastUsed approximates when a working copy was last worked in: git rewrites
// its index on checkout, add and commit.
func lastUsed(dir string) time.Time {
	info, err := os.Stat(filepath.Join(gitDir(dir), "index"))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func isRepo(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

// gitDir is the directory holding a working copy's git data. Worktrees and
// submodules have a .git file with a "gitdir:" line pointing elsewhere.
func gitDir(dir string) string {
	dotGit := filepath.Join(dir, ".git")
	data, err := os.ReadFile(dotGit)
	if err != nil {
		return dotGit
	}
	path, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return dotGit
	}
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	return filepath.Clean(path)
}

// configDir is where a working copy's config lives: worktrees share it with
// the main checkout, named by the "commondir" file in their git directory.
func configDir(dir string) string {
	git := gitDir(dir)
	data, err := os.ReadFile(filepath.Join(git, "commondir"))
	if err != nil {
		return git
	}
	common := strings.TrimSpace(string(data))
	if !filepath.IsAbs(common) {
		common = filepath.Join(git, common)
	}
	return filepath.Clean(common)
}

// repoAt returns the "owner/repo" a working copy points at, or "" when dir is
// not a git repository with a github.com remote.
func repoAt(dir string) string {
	data, err := os.ReadFile(filepath.Join(configDir(dir), "config"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "url" {
			continue
		}
		if m := remotePattern.FindStringSubmatch(strings.TrimSpace(value)); m != nil {
			return m[1] + "/" + m[2]
		}
	}
	return ""
}

// Destination resolves where nameWithOwner should be cloned. With no root
// configured it returns "", letting git use its own default (the repo name in
// the current directory).
func Destination(root, nameWithOwner string) string {
	if root == "" {
		return ""
	}
	return filepath.Join(root, RepoName(nameWithOwner))
}

// RepoName is the bare repository name from an "owner/repo" pair.
func RepoName(nameWithOwner string) string {
	if i := strings.LastIndex(nameWithOwner, "/"); i >= 0 {
		return nameWithOwner[i+1:]
	}
	return nameWithOwner
}

// Exists reports whether path is a directory that already holds a git working
// copy, which callers use to refuse clobbering an existing clone.
func Exists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// Load reads an index saved by Save, dropping clones that no longer exist on
// disk. ok is false when there is no saved index yet.
func Load(path string) (index map[string]string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}, false
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return map[string]string{}, false
	}
	for name, dir := range index {
		if !isRepo(dir) {
			delete(index, name)
		}
	}
	return index, true
}

// Save writes index to path so the next start knows its clones before the
// background scan finishes.
func Save(path string, index map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
