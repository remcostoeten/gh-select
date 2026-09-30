package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/sys"
)

const maxSaveFiles = 200

type treeStatusMsg struct{ text string }

func (t *treeModel) copyFileCmd(path string, content []byte) tea.Cmd {
	ref := t.ref
	return func() tea.Msg {
		if content == nil {
			var err error
			if content, err = t.client.FetchFile(t.repo.NameWithOwner, ref, path); err != nil {
				return treeStatusMsg{"copy failed: " + err.Error()}
			}
		}
		if bytes.IndexByte(content, 0) >= 0 {
			return treeStatusMsg{filepath.Base(path) + " is binary, press s to save it instead"}
		}
		if !sys.Copy(string(content)) {
			return treeStatusMsg{"no clipboard tool found (install wl-clipboard, xclip or xsel)"}
		}
		return treeStatusMsg{"Copied " + path + " to clipboard"}
	}
}

// saveTargets is what s saves: every file in the selection (folders expanded),
// or the highlighted file or folder when nothing is selected.
func (t *treeModel) saveTargets() []string {
	var roots []*node
	if len(t.selected) > 0 {
		for _, n := range t.selected {
			roots = append(roots, n)
		}
	} else if n := t.current(); n != nil && n.name != ".." {
		roots = append(roots, n)
	}
	seen := map[string]bool{}
	var files []string
	var walk func(n *node)
	walk = func(n *node) {
		if !n.isDir {
			if !seen[n.path] {
				seen[n.path] = true
				files = append(files, n.path)
			}
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, n := range roots {
		walk(n)
	}
	sort.Strings(files)
	return files
}

func (t *treeModel) startSave(files []string, cached []byte) tea.Cmd {
	switch {
	case len(files) == 0:
		t.status = "nothing to save here"
		return nil
	case len(files) > maxSaveFiles:
		t.status = fmt.Sprintf("%d files is too many to save one by one, press c for a partial clone", len(files))
		return nil
	}
	t.status = "saving " + plural(len(files), "file", "files") + "…"
	return t.saveCmd(files, cached)
}

// saveCmd writes files into the download directory without git: a single file
// lands directly in it, several keep their repo paths under a folder named
// after the repo. Existing files are never overwritten.
func (t *treeModel) saveCmd(files []string, cached []byte) tea.Cmd {
	dir, ref := t.saveDir, t.ref
	if dir == "" {
		dir = "."
	}
	return func() tea.Msg {
		target := func(p string) string { return filepath.Join(dir, filepath.Base(p)) }
		if len(files) > 1 {
			target = func(p string) string { return filepath.Join(dir, t.repo.Name(), filepath.FromSlash(p)) }
		}

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			failed []string
			last   string
		)
		sem := make(chan struct{}, 6)
		for _, p := range files {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer func() { <-sem; wg.Done() }()
				content := cached
				if content == nil {
					var err error
					if content, err = t.client.FetchFile(t.repo.NameWithOwner, ref, p); err != nil {
						mu.Lock()
						failed = append(failed, p)
						mu.Unlock()
						return
					}
				}
				out, err := writeNew(target(p), content)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failed = append(failed, p)
					return
				}
				last = out
			}()
		}
		wg.Wait()

		saved := len(files) - len(failed)
		where := shortHome(last)
		if len(files) > 1 {
			where = shortHome(filepath.Join(dir, t.repo.Name()))
		}
		switch {
		case saved == 0:
			return treeStatusMsg{"save failed for " + plural(len(failed), "file", "files")}
		case len(failed) > 0:
			return treeStatusMsg{fmt.Sprintf("Saved %d of %d files to %s, failed: %s",
				saved, len(files), where, strings.Join(failed, ", "))}
		case saved == 1:
			return treeStatusMsg{"Saved " + where}
		}
		return treeStatusMsg{"Saved " + plural(saved, "file", "files") + " to " + where}
	}
}

// writeNew writes content to path, or to "name-1.ext", "name-2.ext" … when
// path is taken, and returns where it landed.
func writeNew(path string, content []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 0; ; i++ {
		candidate := path
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		f, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(content); err != nil {
			f.Close()
			return "", err
		}
		return candidate, f.Close()
	}
}

func shortHome(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(abs, home+string(filepath.Separator)) {
		return "~" + abs[len(home):]
	}
	return abs
}
