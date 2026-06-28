package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// node is one entry in the in-memory file tree built from the flat git tree.
type node struct {
	name     string
	path     string
	isDir    bool
	children []*node
	byName   map[string]*node
}

func (n *node) child(name string, isDir bool, path string) *node {
	if n.byName == nil {
		n.byName = map[string]*node{}
	}
	if c, ok := n.byName[name]; ok {
		return c
	}
	c := &node{name: name, path: path, isDir: isDir}
	n.byName[name] = c
	n.children = append(n.children, c)
	return c
}

// sortRec orders every directory's children: folders first, then files,
// alphabetically within each group.
func (n *node) sortRec() {
	sort.SliceStable(n.children, func(i, j int) bool {
		a, b := n.children[i], n.children[j]
		if a.isDir != b.isDir {
			return a.isDir
		}
		return a.name < b.name
	})
	for _, c := range n.children {
		c.sortRec()
	}
}

func buildTree(entries []gh.TreeEntry) *node {
	root := &node{name: "", path: "", isDir: true}
	for _, e := range entries {
		parts := strings.Split(e.Path, "/")
		cur := root
		for i, p := range parts {
			isDir := e.IsDir() || i < len(parts)-1
			full := strings.Join(parts[:i+1], "/")
			cur = cur.child(p, isDir, full)
		}
	}
	root.sortRec()
	return root
}

type treeOutcome int

const (
	treeContinue treeOutcome = iota
	treeBack
	treeConfirm
	treeQuit
)

// treeModel is the folder browser: navigate directories, multi-select folders,
// and preview file contents.
type treeModel struct {
	client *gh.Client
	repo   gh.Repo

	root     *node
	cwd      *node
	stack    []*node // ancestors for breadcrumb / going up
	cursor   int
	selected map[string]*node // path → selected node (folder or file)

	filtering bool   // filter input is active (capturing keystrokes)
	filter    string // current filter text applied to the directory listing

	truncated bool
	loading   bool
	status    string
	err       error

	preview     viewport.Model
	previewing  bool
	previewPath string

	width, height int
}

func newTreeModel(client *gh.Client, repo gh.Repo, w, h int) *treeModel {
	t := &treeModel{
		client:   client,
		repo:     repo,
		selected: map[string]*node{},
		loading:  true,
		width:    w,
		height:   h,
	}
	t.preview = viewport.New(w, h-chromeLines-1) // -1 for previewBody's lead line
	return t
}

func (t *treeModel) setSize(w, h int) {
	t.width, t.height = w, h
	t.preview.Width = w
	t.preview.Height = h - chromeLines - 1
}

// Tree-screen messages.
type treeLoadedMsg struct {
	tree *gh.Tree
	err  error
}
type fileLoadedMsg struct {
	path    string
	content []byte
	err     error
}

func (t *treeModel) fetchTreeCmd() tea.Cmd {
	return func() tea.Msg {
		// Empty ref → HEAD / default branch (resolved server-side).
		tree, err := t.client.FetchTree(t.repo.NameWithOwner, "")
		return treeLoadedMsg{tree: tree, err: err}
	}
}

func (t *treeModel) fetchFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := t.client.FetchFile(t.repo.NameWithOwner, "", path)
		return fileLoadedMsg{path: path, content: content, err: err}
	}
}

// rows returns the entries shown for the current directory: a leading ".."
// when not at the root (and not filtering), then the children matching the
// active filter.
func (t *treeModel) rows() []*node {
	if t.cwd == nil {
		return nil
	}
	rows := make([]*node, 0, len(t.cwd.children)+1)
	if t.filter == "" && len(t.stack) > 0 {
		rows = append(rows, &node{name: "..", isDir: true})
	}
	needle := strings.ToLower(t.filter)
	for _, c := range t.cwd.children {
		if needle == "" || strings.Contains(strings.ToLower(c.name), needle) {
			rows = append(rows, c)
		}
	}
	return rows
}

// clampCursor keeps the cursor within the (possibly filtered) row range.
func (t *treeModel) clampCursor() {
	n := len(t.rows())
	if t.cursor >= n {
		t.cursor = n - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *treeModel) update(msg tea.Msg) (tea.Cmd, treeOutcome) {
	switch msg := msg.(type) {
	case treeLoadedMsg:
		t.loading = false
		if msg.err != nil {
			t.err = msg.err
			return nil, treeContinue
		}
		t.root = buildTree(msg.tree.Entries)
		t.cwd = t.root
		t.truncated = msg.tree.Truncated
		if t.truncated {
			t.status = "⚠ tree truncated by GitHub (very large repo) — some entries hidden"
		}
		return nil, treeContinue

	case fileLoadedMsg:
		if msg.err != nil {
			t.status = "preview failed: " + msg.err.Error()
			return nil, treeContinue
		}
		t.previewing = true
		t.previewPath = msg.path
		t.preview.SetContent(renderPreview(msg.path, msg.content))
		t.preview.GotoTop()
		return nil, treeContinue

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return nil, treeContinue
}

func (t *treeModel) handleKey(key tea.KeyMsg) (tea.Cmd, treeOutcome) {
	if t.previewing {
		switch key.String() {
		case "esc", "q", "left", "h", "backspace":
			t.previewing = false
			return nil, treeContinue
		}
		var cmd tea.Cmd
		t.preview, cmd = t.preview.Update(key)
		return cmd, treeContinue
	}

	// While the filter input is active, capture text instead of navigating.
	if t.filtering {
		switch key.String() {
		case "enter", "down", "up":
			t.filtering = false // keep the filter applied, return to navigation
		case "esc":
			t.filtering = false
			t.filter = ""
		case "backspace":
			if t.filter != "" {
				t.filter = t.filter[:len(t.filter)-1]
			}
		default:
			if key.Type == tea.KeyRunes {
				t.filter += string(key.Runes)
			}
		}
		t.clampCursor()
		return nil, treeContinue
	}

	switch key.String() {
	case "ctrl+c":
		return nil, treeQuit
	case "esc":
		if t.filter != "" { // first esc clears an applied filter
			t.filter = ""
			t.clampCursor()
			return nil, treeContinue
		}
		return nil, treeBack
	case "q":
		return nil, treeBack
	case "/":
		t.filtering = true
		t.cursor = 0
	case "up", "k":
		if t.cursor > 0 {
			t.cursor--
		}
	case "down", "j":
		if t.cursor < len(t.rows())-1 {
			t.cursor++
		}
	case " ", "tab":
		t.toggleSelection()
	case "enter", "right", "l":
		return t.activate()
	case "left", "h", "backspace":
		t.goUp()
	case "c":
		return nil, treeConfirm
	}
	return nil, treeContinue
}

func (t *treeModel) current() *node {
	rows := t.rows()
	if t.cursor < 0 || t.cursor >= len(rows) {
		return nil
	}
	return rows[t.cursor]
}

// toggleSelection marks/unmarks the entry under the cursor. Both folders and
// individual files can be selected for a partial clone.
func (t *treeModel) toggleSelection() {
	n := t.current()
	if n == nil || n.name == ".." {
		return
	}
	if t.selected[n.path] != nil {
		delete(t.selected, n.path)
	} else {
		t.selected[n.path] = n
	}
}

// activate drills into a folder or previews a file.
func (t *treeModel) activate() (tea.Cmd, treeOutcome) {
	n := t.current()
	if n == nil {
		return nil, treeContinue
	}
	if n.name == ".." {
		t.goUp()
		return nil, treeContinue
	}
	if n.isDir {
		t.stack = append(t.stack, t.cwd)
		t.cwd = n
		t.cursor = 0
		t.filter = ""
		return nil, treeContinue
	}
	t.status = "loading " + n.path + "…"
	return t.fetchFileCmd(n.path), treeContinue
}

func (t *treeModel) goUp() {
	if len(t.stack) == 0 {
		return
	}
	t.cwd = t.stack[len(t.stack)-1]
	t.stack = t.stack[:len(t.stack)-1]
	t.cursor = 0
	t.filter = ""
}

func (t *treeModel) selectedFolders() []string {
	folders := make([]string, 0, len(t.selected))
	for p, n := range t.selected {
		if n.isDir {
			folders = append(folders, p)
		}
	}
	sort.Strings(folders)
	return folders
}

func (t *treeModel) selectedFiles() []string {
	files := make([]string, 0, len(t.selected))
	for p, n := range t.selected {
		if !n.isDir {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files
}

func (t *treeModel) breadcrumb() string {
	crumb := t.repo.NameWithOwner
	// stack holds ancestor directories (the root included); render the path of
	// entered directories, skipping the unnamed root.
	for _, n := range t.stack {
		if n == t.root {
			continue
		}
		crumb += "/" + n.name
	}
	if t.cwd != nil && t.cwd != t.root {
		crumb += "/" + t.cwd.name
	}
	return crumb
}

var treeBrowseFooter = keyHint(
	[2]string{"↑↓", "move"},
	[2]string{"→", "open"},
	[2]string{"←", "up"},
	[2]string{"space", "select"},
	[2]string{"/", "filter"},
	[2]string{"c", "clone"},
	[2]string{"esc", "back"},
)

var treePreviewFooter = keyHint(
	[2]string{"↑↓", "scroll"},
	[2]string{"esc", "close"},
)

// chromeParts returns the header context, body, and footer key hints for the
// tree screen, sized to fit innerH body rows. The App wraps these in chrome.
func (t *treeModel) chromeParts(spinnerFrame string, innerH int) (context, body, keys string) {
	if t.err != nil {
		return t.repo.NameWithOwner, errStyle.Render("Error: " + t.err.Error()), treeBrowseFooter
	}
	if t.loading {
		return t.repo.NameWithOwner,
			"\n  " + spinnerFrame + statusStyle.Render("Loading file tree…"),
			treeBrowseFooter
	}
	if t.previewing {
		return t.previewPath, t.previewBody(), treePreviewFooter
	}

	var top strings.Builder
	if n := len(t.selected); n > 0 {
		top.WriteString(markedStyle.Render(fmt.Sprintf("  %d selected", n)))
		top.WriteString("\n")
	}
	if t.filtering || t.filter != "" {
		caret := ""
		if t.filtering {
			caret = "_"
		}
		top.WriteString(statusStyle.Render("  filter: "+t.filter+caret) +
			dimStyle.Render("   (esc to clear)"))
		top.WriteString("\n")
	}
	top.WriteString("\n")

	var status string
	if t.status != "" {
		status = statusStyle.Render("  "+t.status) + "\n"
	}

	// Reserve rows for the header/status lines so the listing fits innerH.
	overhead := strings.Count(top.String(), "\n") + strings.Count(status, "\n")
	visible := innerH - overhead
	if visible < 1 {
		visible = 1
	}

	rows := t.rows()
	start := 0
	if t.cursor >= visible {
		start = t.cursor - visible + 1
	}
	end := start + visible
	if end > len(rows) {
		end = len(rows)
	}

	var b strings.Builder
	b.WriteString(top.String())
	for i := start; i < end; i++ {
		b.WriteString(t.renderRow(i, rows[i]))
	}
	b.WriteString(status)
	return t.breadcrumb(), b.String(), treeBrowseFooter
}

func (t *treeModel) renderRow(i int, n *node) string {
	pointer := "  "
	if i == t.cursor {
		pointer = selectedStyle.Render(" ▸")
	}

	if n.name == ".." {
		return fmt.Sprintf("%s   %s\n", pointer, dimStyle.Render("../"))
	}

	box := "[ ]"
	if t.selected[n.path] != nil {
		box = markedStyle.Render("[x]")
	}

	if n.isDir {
		name := n.name + "/"
		if i == t.cursor {
			name = selectedStyle.Render(name)
		} else {
			name = keyStyle.Render(name)
		}
		return fmt.Sprintf("%s %s %s\n", pointer, box, name)
	}

	name := n.name
	if i == t.cursor {
		name = selectedStyle.Render(name)
	} else {
		name = dimStyle.Render(name)
	}
	return fmt.Sprintf("%s %s %s\n", pointer, box, name)
}

func (t *treeModel) previewBody() string {
	return "\n" + t.preview.View()
}

// renderPreview prepares file bytes for display: it guards against binary
// blobs, caps very large files, and applies syntax highlighting chosen from the
// file path (falling back to content analysis).
func renderPreview(path string, content []byte) string {
	for _, c := range content {
		if c == 0 {
			return dimStyle.Render("(binary file — preview unavailable)")
		}
	}
	const maxBytes = 100 * 1024
	truncated := false
	if len(content) > maxBytes {
		content = content[:maxBytes]
		truncated = true
	}

	out := highlight(path, string(content))
	if truncated {
		out += "\n" + dimStyle.Render("… (truncated)")
	}
	return out
}

// highlight returns ANSI-colored source using chroma, picking a lexer by
// filename then by content. On any failure it returns the source unchanged.
func highlight(path, source string) string {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	it, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}
	var buf strings.Builder
	if err := formatter.Format(&buf, style, it); err != nil {
		return source
	}
	return buf.String()
}
