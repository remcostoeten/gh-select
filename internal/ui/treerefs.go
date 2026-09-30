package ui

import (
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/sys"
)

type refsLoadedMsg struct {
	branches, tags []string
	err            error
}

type treeLinkMsg struct {
	status   string
	ref, sha string
}

func (t *treeModel) fetchRefsCmd() tea.Cmd {
	repo := t.repo.NameWithOwner
	return func() tea.Msg {
		var msg refsLoadedMsg
		done := make(chan struct{})
		go func() {
			msg.tags, _ = t.client.FetchTags(repo)
			close(done)
		}()
		msg.branches, msg.err = t.client.FetchBranches(repo)
		<-done
		return msg
	}
}

func (t *treeModel) refsLoaded(msg refsLoadedMsg) {
	t.refLoading = false
	if msg.err != nil {
		t.status = "Couldn't load branches: " + msg.err.Error()
		return
	}
	p := newPicker(append(append([]string{}, msg.branches...), msg.tags...))
	p.noun = "branches & tags"
	p.tags = map[string]bool{}
	for _, tag := range msg.tags {
		p.tags[tag] = true
	}
	t.refPicker = p
}

// switchRef reloads the tree at ref. The selection is dropped because paths
// marked on one branch may not exist on another.
func (t *treeModel) switchRef(ref string) tea.Cmd {
	t.ref = ref
	t.refPicker = nil
	t.root, t.cwd, t.stack = nil, nil, nil
	t.cursor, t.filter = 0, ""
	t.selected = map[string]*node{}
	t.status = ""
	t.loading = true
	return t.fetchTreeCmd()
}

func (t *treeModel) refOrHead() string {
	if t.ref == "" {
		return "HEAD"
	}
	return t.ref
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func (t *treeModel) webURL(path string, isDir bool, ref string) string {
	kind := "blob"
	if isDir {
		kind = "tree"
	}
	u := t.repo.URL() + "/" + kind + "/" + escapePath(ref)
	if path != "" {
		u += "/" + escapePath(path)
	}
	return u
}

func (t *treeModel) rawURL(path string) string {
	return "https://raw.githubusercontent.com/" + t.repo.NameWithOwner + "/" + escapePath(t.refOrHead()) + "/" + escapePath(path)
}

// linkTarget is the path o, Y and r act on: the previewed file, or the
// highlighted entry (the current folder when that is "..").
func (t *treeModel) linkTarget() (path string, isDir bool) {
	if t.previewing {
		return t.previewPath, false
	}
	n := t.current()
	if n == nil || n.name == ".." {
		if t.cwd == nil {
			return "", true
		}
		return t.cwd.path, true
	}
	return n.path, n.isDir
}

func (t *treeModel) openWebCmd() tea.Cmd {
	path, isDir := t.linkTarget()
	u := t.webURL(path, isDir, t.refOrHead())
	return func() tea.Msg {
		if err := sys.OpenURL(u); err != nil {
			return treeStatusMsg{"Couldn't open browser: " + err.Error()}
		}
		return treeStatusMsg{"Opened " + u}
	}
}

func (t *treeModel) copyRawCmd() tea.Cmd {
	path, isDir := t.linkTarget()
	if isDir {
		t.status = "r copies a file's raw URL, highlight a file first"
		return nil
	}
	return copyLinkCmd("raw URL", t.rawURL(path))
}

// copyPermalinkCmd copies a link pinned to the commit the current branch or
// tag points at, so it keeps showing this exact version.
func (t *treeModel) copyPermalinkCmd() tea.Cmd {
	path, isDir := t.linkTarget()
	ref, raw := t.refOrHead(), t.ref
	if sha := t.commitSHA[ref]; sha != "" {
		return copyLinkCmd("permalink", t.webURL(path, isDir, sha))
	}
	t.status = "resolving commit…"
	repo := t.repo.NameWithOwner
	return func() tea.Msg {
		sha, err := t.client.CommitSHA(repo, raw)
		if err != nil {
			return treeLinkMsg{status: "Couldn't resolve the commit: " + err.Error()}
		}
		u := t.webURL(path, isDir, sha)
		return treeLinkMsg{status: copyStatus("permalink", u), ref: ref, sha: sha}
	}
}

func copyLinkCmd(label, u string) tea.Cmd {
	return func() tea.Msg { return treeStatusMsg{copyStatus(label, u)} }
}

func copyStatus(label, u string) string {
	if sys.Copy(u) {
		return "Copied " + label + ": " + u
	}
	return "Clipboard unavailable: " + u
}
