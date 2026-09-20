// Package local discovers repositories already cloned on disk, so the UI can
// mark them and offer "open" instead of a redundant clone.
package local

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maxDepth bounds the scan at owner/repo layouts (~/src/github/owner/repo),
// which is one level deeper than the flat ~/src/repo case.
const maxDepth = 2

// remotePattern extracts "owner/repo" from any github.com remote URL —
// https://github.com/owner/repo.git, git@github.com:owner/repo.git, or the
// ssh:// form.
var remotePattern = regexp.MustCompile(`github\.com[:/]+([^/\s]+)/([^/\s]+?)(?:\.git)?\s*$`)

// Index scans root for git working copies and maps "owner/repo" to the
// directory holding it. An unreadable or unset root yields an empty index
// rather than an error: local awareness is an enhancement, never a
// precondition for using the app.
func Index(root string) map[string]string {
	found := map[string]string{}
	if root == "" {
		return found
	}
	scan(root, 1, found)
	return found
}

func scan(dir string, depth int, found map[string]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if name := repoAt(path); name != "" {
			if _, seen := found[name]; !seen {
				found[name] = path
			}
			continue
		}
		if depth < maxDepth {
			scan(path, depth+1, found)
		}
	}
}

// repoAt returns the "owner/repo" a working copy points at, or "" when dir is
// not a git repository with a github.com remote.
func repoAt(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
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
