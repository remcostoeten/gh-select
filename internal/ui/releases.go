package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/remcostoeten/gh-select/internal/sys"
	"github.com/sahilm/fuzzy"
)

// releasesLevel is how deep the releases screen is drilled in.
type releasesLevel int

const (
	levelReleases releasesLevel = iota // every release, with notes alongside
	levelAssets                        // the downloads of one release
	levelReader                        // full-screen notes or an asset's contents
)

// previewLimit is how much of an asset is fetched to preview it: enough for a
// checksum list or a changelog, and renderPreview truncates anything longer.
const previewLimit = 100*1024 + 1

type releasesOutcome int

const (
	releasesContinue releasesOutcome = iota
	releasesBack
	releasesQuit
	releasesDownload
)

type releasesLoadedMsg struct {
	releases []gh.Release
	err      error
}

type assetPreviewMsg struct {
	name    string
	content []byte
	err     error
}

// releasesStatusMsg reports the outcome of a side effect run from the screen,
// such as opening a page or copying a link.
type releasesStatusMsg struct{ text string }

// releasesModel browses a repository's releases, their notes and assets, and
// hands the assets chosen for download back to the caller.
type releasesModel struct {
	client *gh.Client
	repo   gh.Repo

	releases []gh.Release
	loading  bool
	err      error
	cursor   int            // index into releaseRows()
	notes    map[int]string // rendered notes per release index, for the current width
	open     int            // release index whose downloads are shown

	assets      []gh.Asset
	assetCursor int // index into assetRows()
	marked      map[int]bool
	suggested   int // asset index that fits this machine, or -1
	extract     bool

	filter        string
	filtering     bool
	releaseFilter string // the releases level's filter, kept while in downloads

	level         releasesLevel
	readerReturn  releasesLevel
	reader        viewport.Model
	readerTitle   string
	readerRaw     []byte
	readerLoading bool

	status        string
	width, height int
}

func newReleasesModel(client *gh.Client, repo gh.Repo, w, h int) *releasesModel {
	r := &releasesModel{client: client, repo: repo, loading: true, notes: map[int]string{}, suggested: -1}
	r.reader = viewport.New(0, 0)
	r.setSize(w, h)
	return r
}

func (r *releasesModel) setSize(w, h int) {
	r.width, r.height = w, h
	r.reader.Width, r.reader.Height = contentWidth(w)-4, h-chromeLines-2
	r.notes = map[int]string{}
	if r.level == levelReader && !r.readerLoading {
		r.reader.SetContent(renderPreview(r.readerTitle, r.readerRaw, r.reader.Width))
	}
}

func (r *releasesModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		releases, err := r.client.FetchReleases(r.repo.NameWithOwner)
		return releasesLoadedMsg{releases: releases, err: err}
	}
}

func (r *releasesModel) previewCmd(a gh.Asset) tea.Cmd {
	return func() tea.Msg {
		body, _, err := r.client.OpenAsset(a)
		if err != nil {
			return assetPreviewMsg{name: a.Name, err: err}
		}
		defer body.Close()
		content, err := io.ReadAll(io.LimitReader(body, previewLimit))
		return assetPreviewMsg{name: a.Name, content: content, err: err}
	}
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := sys.OpenURL(url); err != nil {
			return releasesStatusMsg{"Couldn't open browser: " + err.Error()}
		}
		return releasesStatusMsg{"Opened " + url}
	}
}

func copyCmd(label, text string) tea.Cmd {
	return func() tea.Msg {
		if sys.Copy(text) {
			return releasesStatusMsg{"Copied " + label + " to clipboard"}
		}
		return releasesStatusMsg{"Clipboard unavailable: " + text}
	}
}

func (r *releasesModel) loaded(msg releasesLoadedMsg) {
	r.loading = false
	r.err = msg.err
	r.releases = msg.releases
}

func (r *releasesModel) previewLoaded(msg assetPreviewMsg) {
	if r.level != levelReader || msg.name != r.readerTitle {
		return
	}
	r.readerLoading = false
	if msg.err != nil {
		r.level = r.readerReturn
		r.status = "Couldn't preview " + msg.name + ": " + msg.err.Error()
		return
	}
	r.openReader(msg.name, msg.content)
}

func (r *releasesModel) openReader(title string, content []byte) {
	r.readerTitle, r.readerRaw = title, content
	r.reader.SetContent(renderPreview(title, content, r.reader.Width))
	r.reader.GotoTop()
}

// filterIndices returns the indices of names matching the filter, fuzzy-ranked,
// or every index when there is no filter.
func filterIndices(names []string, filter string) []int {
	if filter == "" {
		out := make([]int, len(names))
		for i := range names {
			out[i] = i
		}
		return out
	}
	matches := fuzzy.Find(filter, names)
	out := make([]int, len(matches))
	for i, m := range matches {
		out[i] = m.Index
	}
	return out
}

func (r *releasesModel) releaseRows() []int {
	names := make([]string, len(r.releases))
	for i, rel := range r.releases {
		names[i] = rel.TagName + " " + rel.Title()
	}
	return filterIndices(names, r.filter)
}

func (r *releasesModel) assetRows() []int {
	names := make([]string, len(r.assets))
	for i, a := range r.assets {
		names[i] = a.Name
	}
	return filterIndices(names, r.filter)
}

// current is the highlighted release on the releases level, and the opened
// one below it.
func (r *releasesModel) current() (gh.Release, bool) {
	if r.level != levelReleases {
		if r.open >= 0 && r.open < len(r.releases) {
			return r.releases[r.open], true
		}
		return gh.Release{}, false
	}
	rows := r.releaseRows()
	if r.cursor < 0 || r.cursor >= len(rows) {
		return gh.Release{}, false
	}
	return r.releases[rows[r.cursor]], true
}

func (r *releasesModel) currentAsset() (int, bool) {
	rows := r.assetRows()
	if r.assetCursor < 0 || r.assetCursor >= len(rows) {
		return 0, false
	}
	return rows[r.assetCursor], true
}

// chosen is what a download takes: every marked asset, or the highlighted one
// when nothing is marked.
func (r *releasesModel) chosen() []gh.Asset {
	if len(r.marked) == 0 {
		if i, ok := r.currentAsset(); ok {
			return []gh.Asset{r.assets[i]}
		}
		return nil
	}
	idx := make([]int, 0, len(r.marked))
	for i := range r.marked {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	out := make([]gh.Asset, 0, len(idx))
	for _, i := range idx {
		out = append(out, r.assets[i])
	}
	return out
}

// checksumAssets are the release's digest files, used to verify downloads.
func (r *releasesModel) checksumAssets() []gh.Asset {
	var out []gh.Asset
	for _, a := range r.assets {
		if !a.Source && gh.IsChecksumFile(a.Name) {
			out = append(out, a)
		}
	}
	return out
}

func (r *releasesModel) openAssets(releaseIndex int) {
	r.open = releaseIndex
	r.releaseFilter = r.filter
	r.assets = r.releases[releaseIndex].Downloads(r.repo.Name())
	r.marked = map[int]bool{}
	r.filter, r.filtering = "", false
	r.suggested = bestForPlatform(r.assets)
	r.assetCursor = max(r.suggested, 0)
	r.level = levelAssets
}

func (r *releasesModel) closeAssets() {
	r.filter, r.filtering = r.releaseFilter, false
	r.level = levelReleases
}

func (r *releasesModel) update(msg tea.Msg) (tea.Cmd, releasesOutcome) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, releasesContinue
	}
	if key.String() == "ctrl+c" {
		if r.filter == "" || r.level == levelReader {
			return nil, releasesQuit
		}
		r.filter = ""
		r.cursor, r.assetCursor = 0, 0
		return nil, releasesContinue
	}
	r.status = ""
	if r.filtering {
		r.updateFilter(key)
		return nil, releasesContinue
	}
	switch r.level {
	case levelAssets:
		return r.updateAssets(key)
	case levelReader:
		return r.updateReader(key)
	}
	return r.updateReleases(key)
}

// updateFilter edits the filter while "/" mode is active: enter keeps it and
// returns to navigation, esc drops it.
func (r *releasesModel) updateFilter(key tea.KeyMsg) {
	switch key.String() {
	case "esc":
		r.filter, r.filtering = "", false
	case "enter", "up", "down":
		r.filtering = false
		return
	case "backspace":
		if r.filter == "" {
			r.filtering = false
			return
		}
		rs := []rune(r.filter)
		r.filter = string(rs[:len(rs)-1])
	default:
		switch key.Type {
		case tea.KeyRunes:
			r.filter += string(key.Runes)
		case tea.KeySpace:
			r.filter += " "
		default:
			return
		}
	}
	r.cursor, r.assetCursor = 0, 0
}

// moveCursor applies a navigation key to cursor over n rows.
func moveCursor(key string, cursor *int, n int) bool {
	switch key {
	case "up", "k":
		if *cursor > 0 {
			*cursor--
		}
	case "down", "j":
		if *cursor < n-1 {
			*cursor++
		}
	case "home", "g":
		*cursor = 0
	case "end", "G":
		*cursor = max(n-1, 0)
	case "pgup":
		*cursor = max(*cursor-10, 0)
	case "pgdown":
		*cursor = max(min(*cursor+10, n-1), 0)
	default:
		return false
	}
	return true
}

func (r *releasesModel) updateReleases(key tea.KeyMsg) (tea.Cmd, releasesOutcome) {
	rows := r.releaseRows()
	if moveCursor(key.String(), &r.cursor, len(rows)) {
		return nil, releasesContinue
	}
	switch key.String() {
	case "esc":
		if r.filter != "" {
			r.filter, r.cursor = "", 0
			return nil, releasesContinue
		}
		return nil, releasesBack
	case "q", "left", "h", "backspace":
		return nil, releasesBack
	case "/":
		r.filtering = true
	case "enter", "right", "l":
		if r.cursor < len(rows) {
			r.openAssets(rows[r.cursor])
		}
	case "n":
		if rel, ok := r.current(); ok {
			r.readerReturn = levelReleases
			r.level = levelReader
			r.openReader("notes.md", []byte(notesSource(rel)))
		}
	case "o":
		if rel, ok := r.current(); ok && rel.HTMLURL != "" {
			return openURLCmd(rel.HTMLURL), releasesContinue
		}
	case "y":
		if rel, ok := r.current(); ok && rel.HTMLURL != "" {
			return copyCmd("release URL", rel.HTMLURL), releasesContinue
		}
	}
	return nil, releasesContinue
}

func (r *releasesModel) updateAssets(key tea.KeyMsg) (tea.Cmd, releasesOutcome) {
	rows := r.assetRows()
	if moveCursor(key.String(), &r.assetCursor, len(rows)) {
		return nil, releasesContinue
	}
	switch key.String() {
	case "esc":
		if r.filter != "" {
			r.filter, r.assetCursor = "", 0
			return nil, releasesContinue
		}
		r.closeAssets()
	case "left", "h", "backspace":
		r.closeAssets()
	case "q":
		return nil, releasesBack
	case "/":
		r.filtering = true
	case " ":
		if i, ok := r.currentAsset(); ok {
			if r.marked[i] {
				delete(r.marked, i)
			} else {
				r.marked[i] = true
			}
		}
		if r.assetCursor < len(rows)-1 {
			r.assetCursor++
		}
	case "a":
		all := true
		for _, i := range rows {
			all = all && r.marked[i]
		}
		for _, i := range rows {
			if all {
				delete(r.marked, i)
			} else {
				r.marked[i] = true
			}
		}
	case "v":
		i, ok := r.currentAsset()
		if !ok {
			return nil, releasesContinue
		}
		a := r.assets[i]
		if a.Source {
			r.status = "source archives can't be previewed, download them instead"
			return nil, releasesContinue
		}
		r.readerReturn = levelAssets
		r.level = levelReader
		r.readerTitle = a.Name
		r.readerLoading = true
		return r.previewCmd(a), releasesContinue
	case "y":
		if i, ok := r.currentAsset(); ok {
			url := r.assets[i].DownloadURL
			if url == "" {
				url = r.assets[i].URL
			}
			return copyCmd("download URL", url), releasesContinue
		}
	case "o":
		if rel, ok := r.current(); ok && rel.HTMLURL != "" {
			return openURLCmd(rel.HTMLURL), releasesContinue
		}
	case "enter", "d", "x":
		if len(r.chosen()) > 0 {
			r.extract = key.String() == "x"
			return nil, releasesDownload
		}
	}
	return nil, releasesContinue
}

func (r *releasesModel) updateReader(key tea.KeyMsg) (tea.Cmd, releasesOutcome) {
	switch key.String() {
	case "esc", "q", "left", "h", "backspace":
		r.level = r.readerReturn
		r.readerLoading = false
		return nil, releasesContinue
	}
	var cmd tea.Cmd
	r.reader, cmd = r.reader.Update(key)
	return cmd, releasesContinue
}

// notesSource is the markdown shown for a release: its heading, then the body.
func notesSource(rel gh.Release) string {
	body := strings.TrimSpace(rel.Body)
	if body == "" {
		body = "_No release notes._"
	}
	return "# " + rel.Title() + "\n\n" + body
}

func (r *releasesModel) renderedNotes(i, width int) string {
	if s, ok := r.notes[i]; ok {
		return s
	}
	body := strings.TrimSpace(r.releases[i].Body)
	s := dimStyle.Render("no release notes")
	if body != "" {
		s = renderMarkdown(body, width)
	}
	r.notes[i] = s
	return s
}

// chromeParts returns the header context, body, status line, and footer for
// the releases screen, sized to innerH body rows.
func (r *releasesModel) chromeParts(spinnerFrame string, innerH int) (context, body, status, keys string) {
	context = r.repo.NameWithOwner + " · releases"
	cw := contentWidth(r.width)
	if r.status != "" {
		status = statusStyle.Render(r.status)
	}
	switch {
	case r.loading:
		return context, panel("releases", "", cw, innerH, false),
			spinnerFrame + statusStyle.Render(" Loading releases…"), releasesFooter()
	case r.err != nil:
		return context, panel("releases", errStyle.Render("Error: "+r.err.Error()), cw, innerH, false),
			"", releasesFooter()
	case len(r.releases) == 0:
		return context, panel("releases", dimStyle.Render("this repository has no releases"), cw, innerH, false),
			"", releasesFooter()
	}

	switch r.level {
	case levelAssets:
		rel, _ := r.current()
		return context + " · " + rel.TagName, r.withFilter(r.assetsBody(rel, cw, innerH-searchBoxLines), cw),
			status, r.footer(assetsFooter())
	case levelReader:
		if r.readerLoading {
			return context, panel(r.readerTitle, "", cw, innerH, false),
				spinnerFrame + statusStyle.Render(" Loading "+r.readerTitle+"…"), readerFooter()
		}
		title := r.readerTitle
		if r.readerReturn == levelReleases {
			rel, _ := r.current()
			title = rel.TagName + " · notes"
		}
		pct := dimStyle.Render(fmt.Sprintf("%3.0f%%", r.reader.ScrollPercent()*100))
		return context, panel(title, r.reader.View(), cw, innerH, true), pct, readerFooter()
	}
	return context, r.withFilter(r.releasesBody(cw, innerH-searchBoxLines), cw), status, r.footer(releasesFooter())
}

// withFilter stacks the filter panel above a list body.
func (r *releasesModel) withFilter(body string, cw int) string {
	placeholder := "press / to filter"
	if r.filtering {
		placeholder = "type to filter…"
	}
	field := dimStyle.Render("  " + placeholder)
	if r.filtering || r.filter != "" {
		field = searchField(r.filter, placeholder)
	}
	return panel("filter", field, cw, searchBoxLines, r.filtering) + "\n" + body
}

func (r *releasesModel) footer(normal string) string {
	if r.filtering {
		return filterFooter()
	}
	return normal
}

func (r *releasesModel) releasesBody(cw, h int) string {
	mw, sw := splitWidths(cw)
	rows := r.releaseRows()
	var b strings.Builder
	start, end := window(r.cursor, len(rows), h-2)
	tagW := 0
	for _, i := range rows[start:end] {
		tagW = max(tagW, lipgloss.Width(r.releases[i].TagName))
	}
	for pos := start; pos < end; pos++ {
		rel := r.releases[rows[pos]]
		pointer, tag := "  ", keyStyle.Render(rel.TagName)
		if pos == r.cursor {
			pointer, tag = selectedStyle.Render("❯ "), selectedStyle.Render(rel.TagName)
		}
		line := pointer + tag + strings.Repeat(" ", tagW-lipgloss.Width(rel.TagName)) + "  " +
			dimStyle.Render(rel.Date().Format("2006-01-02"))
		if badge := releaseBadge(rel); badge != "" {
			line += "  " + badge
		}
		if rel.Title() != rel.TagName {
			line += "  " + lipgloss.NewStyle().Foreground(colFg).Render(rel.Title())
		}
		b.WriteString(line + "\n")
	}
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render("no releases match “" + r.filter + "”"))
	}
	title := fmt.Sprintf("releases · %d", len(r.releases))
	if r.filter != "" {
		title = fmt.Sprintf("releases · %d/%d", len(rows), len(r.releases))
	}
	if sw == 0 {
		return panel(title, b.String(), cw, h, true)
	}
	notes := dimStyle.Render("no selection")
	if r.cursor < len(rows) {
		notes = r.renderedNotes(rows[r.cursor], sw-4)
	}
	return hsplit(panel(title, b.String(), mw, h, true), panel("notes", notes, sw, h, false))
}

func (r *releasesModel) assetsBody(rel gh.Release, cw, h int) string {
	mw, sw := splitWidths(cw)
	rows := r.assetRows()
	nameW := 0
	for _, a := range r.assets {
		nameW = max(nameW, lipgloss.Width(a.Name))
	}
	nameW = min(nameW, max(mw-34, 10))

	var b strings.Builder
	start, end := window(r.assetCursor, len(rows), h-2)
	for pos := start; pos < end; pos++ {
		i := rows[pos]
		a := r.assets[i]
		pointer := "  "
		name := truncate(a.Name, nameW)
		pad := strings.Repeat(" ", nameW-lipgloss.Width(name))
		if pos == r.assetCursor {
			pointer, name = selectedStyle.Render("❯ "), selectedStyle.Render(name)
		} else {
			name = lipgloss.NewStyle().Foreground(colFg).Render(name)
		}
		check := dimStyle.Render("○ ")
		if r.marked[i] {
			check = markedStyle.Render("● ")
		}
		meta := dimStyle.Render("source archive")
		if !a.Source {
			meta = dimStyle.Render(fmt.Sprintf("%9s  ↓%d", HumanSize(a.Size), a.DownloadCount))
		}
		if i == r.suggested {
			meta += markedStyle.Render("  ◆ this machine")
		}
		b.WriteString(pointer + check + name + pad + "  " + meta + "\n")
	}
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render("no downloads match “" + r.filter + "”"))
	}
	title := fmt.Sprintf("%s · %s", rel.TagName, plural(len(r.assets), "download", "downloads"))
	if len(r.marked) > 0 {
		title += fmt.Sprintf(" · %d marked", len(r.marked))
	}
	if sw == 0 {
		return panel(title, b.String(), cw, h, true)
	}
	return hsplit(panel(title, b.String(), mw, h, true), panel("release", r.releaseInfo(rel), sw, h, false))
}

func (r *releasesModel) releaseInfo(rel gh.Release) string {
	kv := func(label, value string) string {
		return dimStyle.Render(fmt.Sprintf("%-10s", label)) + value
	}
	var total int64
	for _, a := range r.chosen() {
		total += a.Size
	}
	lines := []string{
		kv("tag", titleStyle.Render(rel.TagName)),
		kv("name", rel.Title()),
		kv("date", rel.Date().Format("2006-01-02 15:04")),
	}
	if badge := releaseBadge(rel); badge != "" {
		lines = append(lines, kv("kind", badge))
	}
	verify := dimStyle.Render("no checksum file")
	if n := len(r.checksumAssets()); n > 0 {
		verify = markedStyle.Render("against " + plural(n, "checksum file", "checksum files"))
	}
	lines = append(lines,
		"",
		kv("selected", fmt.Sprintf("%d · %s", len(r.chosen()), HumanSize(total))),
		kv("verify", verify),
	)
	return strings.Join(lines, "\n")
}

func releaseBadge(rel gh.Release) string {
	switch {
	case rel.Draft:
		return privateBadge.Render("draft")
	case rel.Prerelease:
		return privateBadge.Render("pre-release")
	}
	return ""
}

// window returns the [start, end) slice of n rows that keeps cursor visible in
// a viewport of size rows.
func window(cursor, n, size int) (int, int) {
	size = max(size, 1)
	start := 0
	if cursor >= size {
		start = cursor - size + 1
	}
	return start, min(start+size, n)
}

// HumanSize formats a byte count with a binary unit, e.g. "12.4 MB".
func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func releasesFooter() string {
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"enter", "assets"},
		[2]string{"/", "filter"},
		[2]string{"n", "notes"},
		[2]string{"o", "open"},
		[2]string{"y", "copy URL"},
		[2]string{"esc", "back"},
	)
}

func assetsFooter() string {
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"space", "mark"},
		[2]string{"a", "all"},
		[2]string{"/", "filter"},
		[2]string{"v", "preview"},
		[2]string{"y", "copy URL"},
		[2]string{"enter", "download"},
		[2]string{"x", "download+extract"},
		[2]string{"esc", "back"},
	)
}

func filterFooter() string {
	return keyHint(
		[2]string{"type", "filter"},
		[2]string{"enter", "done"},
		[2]string{"esc", "clear"},
	)
}

func readerFooter() string {
	return keyHint(
		[2]string{"↑↓", "scroll"},
		[2]string{"esc", "back"},
		[2]string{"^C", "quit"},
	)
}
