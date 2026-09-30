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
	"github.com/charmbracelet/lipgloss"
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

	helpVisible bool // full-keymap overlay (the footer only shows essentials)

	truncated bool
	loading   bool
	status    string
	err       error

	preview     viewport.Model
	previewing  bool
	previewPath string
	previewRaw  []byte // kept so a resize can re-wrap the rendered preview

	saveDir string // where s saves files; "" means the cwd

	ref        string // branch or tag being browsed; "" is the default branch
	refPicker  *pickerModel
	refLoading bool
	commitSHA  map[string]string // ref → commit, for permalinks

	width, height int
}

func newTreeModel(client *gh.Client, repo gh.Repo, w, h int) *treeModel {
	t := &treeModel{
		client:    client,
		repo:      repo,
		selected:  map[string]*node{},
		commitSHA: map[string]string{},
		loading:   true,
		width:     w,
		height:    h,
	}
	t.preview = viewport.New(contentWidth(w)-4, h-chromeLines-2) // inside the preview panel
	return t
}

func (t *treeModel) setSize(w, h int) {
	t.width, t.height = w, h
	t.preview.Width = contentWidth(w) - 4
	t.preview.Height = h - chromeLines - 2
	if t.previewing {
		t.preview.SetContent(renderPreview(t.previewPath, t.previewRaw, t.preview.Width))
	}
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
	ref := t.ref
	return func() tea.Msg {
		// Empty ref → HEAD / default branch (resolved server-side).
		tree, err := t.client.FetchTree(t.repo.NameWithOwner, ref)
		return treeLoadedMsg{tree: tree, err: err}
	}
}

func (t *treeModel) fetchFileCmd(path string) tea.Cmd {
	ref := t.ref
	return func() tea.Msg {
		content, err := t.client.FetchFile(t.repo.NameWithOwner, ref, path)
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
			t.status = "⚠ tree truncated by GitHub (very large repo), some entries hidden"
		}
		return nil, treeContinue

	case fileLoadedMsg:
		if msg.err != nil {
			t.status = "preview failed: " + msg.err.Error()
			return nil, treeContinue
		}
		t.status = "" // clear the "loading …" message now the preview is up
		t.previewing = true
		t.previewPath = msg.path
		t.previewRaw = msg.content
		t.preview.SetContent(renderPreview(msg.path, msg.content, t.preview.Width))
		t.preview.GotoTop()
		return nil, treeContinue

	case treeStatusMsg:
		t.status = msg.text
		return nil, treeContinue

	case treeLinkMsg:
		t.status = msg.status
		if msg.sha != "" {
			t.commitSHA[msg.ref] = msg.sha
		}
		return nil, treeContinue

	case refsLoadedMsg:
		t.refsLoaded(msg)
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
			t.status = ""
			return nil, treeContinue
		case "y":
			return t.copyFileCmd(t.previewPath, t.previewRaw), treeContinue
		case "s":
			return t.startSave([]string{t.previewPath}, t.previewRaw), treeContinue
		case "o":
			return t.openWebCmd(), treeContinue
		case "Y":
			return t.copyPermalinkCmd(), treeContinue
		case "r":
			return t.copyRawCmd(), treeContinue
		}
		var cmd tea.Cmd
		t.preview, cmd = t.preview.Update(key)
		return cmd, treeContinue
	}

	if t.refPicker != nil {
		switch t.refPicker.handleKey(key.String(), key.Runes, key.Type == tea.KeyRunes) {
		case pickerQuit:
			return nil, treeQuit
		case pickerBack:
			t.refPicker = nil
		case pickerChosen:
			return t.switchRef(t.refPicker.selection()), treeContinue
		}
		return nil, treeContinue
	}

	if t.helpVisible {
		switch key.String() {
		case "ctrl+c":
			return nil, treeQuit
		case "?", "esc", "q", "enter":
			t.helpVisible = false
		}
		return nil, treeContinue
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
	case "?":
		t.helpVisible = true
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
	case "s":
		return t.startSave(t.saveTargets(), nil), treeContinue
	case "y":
		if n := t.current(); n != nil && !n.isDir {
			t.status = "copying " + n.path + "…"
			return t.copyFileCmd(n.path, nil), treeContinue
		}
		t.status = "y copies a file's contents, highlight a file first"
	case "b":
		if !t.refLoading {
			t.refLoading = true
			return t.fetchRefsCmd(), treeContinue
		}
	case "o":
		return t.openWebCmd(), treeContinue
	case "Y":
		return t.copyPermalinkCmd(), treeContinue
	case "r":
		return t.copyRawCmd(), treeContinue
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
	t.status = "" // the status line now shows the selection summary instead
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
	if t.ref != "" {
		crumb += "@" + t.ref
	}
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

// The footer shows only the essentials; ? opens the full keymap overlay.
func treeBrowseFooter() string {
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"→", "open"},
		[2]string{"space", "select"},
		[2]string{"c", "clone"},
		[2]string{"s", "save"},
		[2]string{"y", "copy"},
		[2]string{"b", "branch/tag"},
		[2]string{"o", "open on GitHub"},
		[2]string{"?", "help"},
	)
}

func treePreviewFooter() string {
	return keyHint(
		[2]string{"↑↓", "scroll"},
		[2]string{"y", "copy"},
		[2]string{"s", "save"},
		[2]string{"o", "open on GitHub"},
		[2]string{"Y", "permalink"},
		[2]string{"r", "raw URL"},
		[2]string{"esc", "close"},
	)
}

func treeHelpFooter() string {
	return keyHint(
		[2]string{"?", "close"},
		[2]string{"^C", "quit"},
	)
}

// treeHelpBody is the full keymap overlay opened with ?.
func treeHelpBody() string {
	return helpBody("Tree browser keys", [][2]string{
		{"↑/k  ↓/j", "move"},
		{"enter → l", "open folder · preview file"},
		{"← h backspace", "parent folder"},
		{"space / tab", "select or unselect a file or folder"},
		{"c", "partial clone of the selection (git)"},
		{"s", "save the selection, or the highlighted file or folder, without git"},
		{"y", "copy the highlighted file's contents to the clipboard"},
		{"b", "browse another branch or tag"},
		{"o", "open the file or folder on github.com"},
		{"Y", "copy a permalink pinned to the current commit"},
		{"r", "copy the file's raw download URL"},
		{"/", "filter the current folder"},
		{"esc", "clear filter · back"},
		{"q", "back to actions"},
		{"ctrl+c", "quit"},
	})
}

// statusLine feeds the chrome's status row: a transient message when there is
// one, otherwise (only on narrow terminals, where the selected panel is
// hidden) a summary of everything selected so far.
func (t *treeModel) statusLine() string {
	if t.status != "" {
		return statusStyle.Render(t.status)
	}
	if sideWidth(contentWidth(t.width)) > 0 {
		return "" // the selected panel already shows the marks
	}
	if n := len(t.selected); n > 0 {
		paths := append(t.selectedFolders(), t.selectedFiles()...)
		return markedStyle.Render(fmt.Sprintf("%d selected", n)) +
			dimStyle.Render(" · "+strings.Join(paths, ", "))
	}
	return ""
}

// selectedPanelBody lists every marked path, or a hint when nothing is marked.
func (t *treeModel) selectedPanelBody() string {
	if len(t.selected) == 0 {
		return dimStyle.Render("space marks a file or folder\nfor the partial clone")
	}
	var b strings.Builder
	for _, p := range append(t.selectedFolders(), t.selectedFiles()...) {
		b.WriteString(markedStyle.Render("x ") + p + "\n")
	}
	return b.String()
}

// chromeParts returns the header context, body, status line, and footer key
// hints for the tree screen, sized to fit innerH body rows. The App wraps
// these in chrome.
func (t *treeModel) chromeParts(spinnerFrame string, innerH int) (context, body, status, keys string) {
	cw := contentWidth(t.width)
	if t.err != nil {
		return t.repo.NameWithOwner,
			panel("files", errStyle.Render("Error: "+t.err.Error()), cw, innerH, false),
			"", treeBrowseFooter()
	}
	if t.loading {
		return t.repo.NameWithOwner,
			panel("files", "", cw, innerH, false),
			spinnerFrame + statusStyle.Render("Loading file tree…"),
			treeBrowseFooter()
	}
	if t.previewing {
		status := dimStyle.Render(fmt.Sprintf("%3.0f%%", t.preview.ScrollPercent()*100))
		if t.status != "" {
			status = statusStyle.Render(t.status)
		}
		return t.previewPath,
			panel("preview", t.preview.View(), cw, innerH, true),
			status, treePreviewFooter()
	}
	if t.refPicker != nil {
		return t.breadcrumb(),
			panel(t.refPicker.context(), t.refPicker.body(innerH-2), cw, innerH, true),
			"", refPickerFooter()
	}
	if t.helpVisible {
		return t.breadcrumb(),
			panel("keys", treeHelpBody(), cw, innerH, true),
			"", treeHelpFooter()
	}

	var top string
	overhead := 0
	if t.filtering || t.filter != "" {
		caret := ""
		if t.filtering {
			caret = "_"
		}
		top = statusStyle.Render("filter: "+t.filter+caret) +
			dimStyle.Render("   (esc to clear)") + "\n"
		overhead = 1
	}

	// The listing scrolls within the panel's inner rows.
	visible := innerH - 2 - overhead
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
	b.WriteString(top)
	for i := start; i < end; i++ {
		b.WriteString(t.renderRow(i, rows[i]))
	}

	mw, sw := splitWidths(cw)
	files := panel("files", b.String(), mw, innerH, true)
	if sw > 0 {
		files = hsplit(files, panel(fmt.Sprintf("selected · %d", len(t.selected)),
			t.selectedPanelBody(), sw, innerH, false))
	}
	return t.breadcrumb(), files, t.statusLine(), treeBrowseFooter()
}

func (t *treeModel) renderRow(i int, n *node) string {
	pointer := "  "
	if i == t.cursor {
		pointer = selectedStyle.Render(" ❯")
	}

	if n.name == ".." {
		return fmt.Sprintf("%s   %s\n", pointer, dimStyle.Render("../"))
	}

	box := dimStyle.Render("[ ]")
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

// renderPreview prepares file bytes for display: it guards against binary
// blobs, caps very large files, renders markdown as prose, and otherwise
// applies syntax highlighting chosen from the file path (falling back to
// content analysis). width is the wrap width for markdown.
func renderPreview(path string, content []byte, width int) string {
	for _, c := range content {
		if c == 0 {
			return dimStyle.Render("(binary file, no preview)")
		}
	}
	const maxBytes = 100 * 1024
	truncated := false
	if len(content) > maxBytes {
		content = content[:maxBytes]
		truncated = true
	}

	var out string
	if isMarkdown(path) {
		out = renderMarkdown(string(content), width)
	} else {
		out = highlight(path, string(content))
	}
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
	styleName := "catppuccin-mocha"
	if !lipgloss.HasDarkBackground() {
		styleName = "catppuccin-latte"
	}
	style := styles.Get(styleName)
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

func refPickerFooter() string {
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"type", "search"},
		[2]string{"enter", "browse"},
		[2]string{"esc", "back"},
		[2]string{"^C", "quit"},
	)
}
