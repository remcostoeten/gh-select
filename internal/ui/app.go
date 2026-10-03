// Package ui implements the Bubble Tea terminal interface: repository
// selection, an action menu, and (next) the folder tree browser.
package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// searchScope selects what the list's type-to-search queries against.
type searchScope int

const (
	scopeMine    searchScope = iota // fuzzy-filter the viewer's cached repos
	scopeStarred                    // fuzzy-filter the viewer's starred repos
	scopeGitHub                     // query the GitHub search API for any repo
)

// searchDebounce is how long typing must pause before a GitHub search fires,
// so a burst of keystrokes costs one request instead of one per character.
const searchDebounce = 350 * time.Millisecond

// ActionType is the side-effecting operation chosen by the user, executed by
// the caller after the TUI exits (so git/clone output owns the terminal).
type ActionType int

const (
	ActionNone ActionType = iota
	ActionClone
	ActionCloneBranch // opens the branch picker; resolves to ActionClone
	ActionSparseClone
	ActionViewReadme // opens the rendered readme inside the TUI
	ActionReleases   // opens the releases browser; resolves to ActionDownload
	ActionCopyName
	ActionCopyURL
	ActionOpenWeb
	ActionOpenEditor // open the local clone in $EDITOR, cloning first if needed
	ActionPull       // fast-forward an existing local clone
	ActionPrintPath  // print the local path for a shell wrapper to cd into
	ActionDelete     // permanently delete the repositories in Result.Deletes
	ActionDownload   // download Result.Assets of release Result.Release
)

// Result is what the program should do once the TUI returns.
type Result struct {
	Action    ActionType
	Repo      gh.Repo
	Branch    string     // non-empty to clone a specific branch
	Folders   []string   // selected folders, for ActionSparseClone
	Files     []string   // selected individual files, for ActionSparseClone
	LocalPath string     // existing working copy, when the repo is already cloned
	Deletes   []gh.Repo  // repositories confirmed for deletion, for ActionDelete
	Release   string     // release tag, for ActionDownload
	Assets    []gh.Asset // release assets to download, for ActionDownload
	Checksums []gh.Asset // the release's checksum files, to verify Assets against
	Extract   bool       // unpack downloaded archives, for ActionDownload
}

type screen int

const (
	screenList screen = iota
	screenActions
	screenBranches
	screenTree
	screenReadme
	screenConfirmDelete
	screenReleases
)

// Messages emitted by background commands.
type reposLoadedMsg struct{ repos []gh.Repo }
type errMsg struct{ err error }

// debounceMsg fires after a typing pause; seq lets us ignore it if more keys
// were pressed in the meantime. remoteReposMsg carries GitHub search results,
// tagged with the query they answer so stale responses can be discarded.
type debounceMsg struct{ seq int }
type remoteReposMsg struct {
	query string
	repos []gh.Repo
	err   error
}
type starredLoadedMsg struct {
	repos []gh.Repo
	err   error
}
type branchesLoadedMsg struct {
	branches []string
	err      error
}

// App is the root Bubble Tea model coordinating the screens.
type App struct {
	client  *gh.Client
	saveFn  func([]gh.Repo) // persist freshly fetched repos to cache
	version string          // shown in the header chrome

	screen   screen
	list     list.Model
	repos    []gh.Repo // viewer's repos (scopeMine); list items are query matches
	query    string    // always-on search text for the repo list
	selected gh.Repo

	// search scope + debounced GitHub search state
	scope     searchScope
	searchSeq int  // increments per keystroke; guards stale debounce/results
	searching bool // a GitHub search request is in flight
	remote    []gh.Repo
	sortBy    sortOrder

	starred        []gh.Repo
	starredSave    func([]gh.Repo)
	starredFetched bool // fetched this session, so tab doesn't refetch
	starredLoading bool

	// action menu
	actionCursor int
	actions      []actionItem // rebuilt per repo: local clones get extra entries

	// local clones, keyed by "owner/repo"; empty when no clone dir is set
	locals    map[string]string
	printPath bool // select-and-print mode: enter returns a path, no action menu
	saveDir   string

	// bulk selection for deletion, keyed by "owner/repo"
	marked   map[string]gh.Repo
	markNote string

	listHelp bool
	rate     rateLimits

	// retype-to-confirm delete screen; confirmReturn is the screen esc goes back to
	confirm       *confirmModel
	confirmReturn screen

	// branch picker
	picker        *pickerModel
	branchLoading bool

	// tree browser
	tree *treeModel

	// readme reader
	readme *readmeModel

	// release browser
	releases *releasesModel

	spinner spinner.Model
	loading bool
	err     error
	status  string
	width   int
	height  int

	Result Result
}

// NewApp builds the root model. initial holds any cached repos to show
// immediately; refresh requests a background fetch (stale-while-revalidate).
func NewApp(client *gh.Client, initial []gh.Repo, refresh bool, saveFn func([]gh.Repo), version string) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = statusStyle

	a := &App{
		client:  client,
		saveFn:  saveFn,
		version: version,
		screen:  screenList,
		repos:   initial,
		marked:  map[string]gh.Repo{},
		spinner: sp,
		loading: refresh,
		width:   80,
		height:  24,
	}
	lw, lh := a.listPanelSize()
	a.list = newRepoList(initial, lw-4, lh-2)
	a.refreshDelegate()
	return a
}

// refreshDelegate rebuilds the row renderer so it sees the current scope and
// the live locals/marked maps.
func (a *App) refreshDelegate() {
	a.list.SetDelegate(compactDelegate{
		bare:   a.scope == scopeMine,
		locals: a.locals,
		marked: a.marked,
	})
}

// SetLocalClones tells the UI which repos already exist on disk, keyed by
// "owner/repo", so they can be badged and offered "open" over "clone".
func (a *App) SetLocalClones(locals map[string]string) {
	a.locals = locals
	a.refreshDelegate()
}

// SetPrintPath switches the app into select-and-print mode, where choosing a
// repo returns its path instead of opening the action menu.
func (a *App) SetPrintPath(on bool) { a.printPath = on }

// SetSaveDir sets where files saved from the tree browser land; "" means the
// current directory.
func (a *App) SetSaveDir(dir string) { a.saveDir = dir }

// localPath is the working copy for a repo, or "" when it isn't cloned.
func (a *App) localPath(nameWithOwner string) string { return a.locals[nameWithOwner] }

// listPanelSize is the outer size of the repo list panel: the content column
// minus the details column, and the body height minus the search panel.
func (a *App) listPanelSize() (w, h int) {
	main, _ := splitWidths(contentWidth(a.width))
	return main, a.height - chromeLines - searchBoxLines
}

// SetStarred seeds the starred scope from cache and sets how a fresh fetch
// is persisted.
func (a *App) SetStarred(cached []gh.Repo, save func([]gh.Repo)) {
	a.starred, a.starredSave = cached, save
}

// applyFilter recomputes the visible repo items for the current scope, query
// and sort, and resets the selection to the top match.
func (a *App) applyFilter() {
	switch a.scope {
	case scopeMine:
		a.list.SetItems(listRepos(a.repos, a.query, a.locals, a.sortBy, true))
	case scopeStarred:
		a.list.SetItems(listRepos(a.starred, a.query, a.locals, a.sortBy, true))
	case scopeGitHub:
		if a.query == "" {
			a.list.SetItems(nil)
		} else {
			a.list.SetItems(listRepos(a.remote, a.query, a.locals, a.sortBy, false))
		}
	}
	a.list.Select(0)
}

func (a *App) fetchStarredCmd() tea.Cmd {
	return func() tea.Msg {
		repos, err := a.client.FetchStarred()
		return starredLoadedMsg{repos: repos, err: err}
	}
}

// debounceSearchCmd schedules a debounceMsg tagged with the current sequence
// number; only the latest keystroke's timer triggers an actual search.
func (a *App) debounceSearchCmd() tea.Cmd {
	seq := a.searchSeq
	return tea.Tick(searchDebounce, func(time.Time) tea.Msg {
		return debounceMsg{seq: seq}
	})
}

// remoteSearchCmd performs a GitHub repository search off the UI goroutine.
func (a *App) remoteSearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		repos, err := a.client.SearchRepos(githubQuery(query))
		return remoteReposMsg{query: query, repos: repos, err: err}
	}
}

// toggleScope steps to the next scope (my repos, starred, GitHub) and
// re-applies the current query there. Starred repos are fetched on the first
// visit of a session, showing the cached list meanwhile.
func (a *App) toggleScope() (tea.Model, tea.Cmd) {
	return a.setScope((a.scope + 1) % 3)
}

func (a *App) setScope(next searchScope) (tea.Model, tea.Cmd) {
	a.status = ""
	a.scope = next
	a.sortBy = sortDefault
	a.searching = false
	a.searchSeq++ // invalidate any in-flight GitHub response
	a.refreshDelegate()
	a.remote = nil
	a.applyFilter()
	switch {
	case next == scopeStarred && !a.starredFetched && a.client != nil:
		a.starredFetched, a.starredLoading = true, true
		return a, tea.Batch(a.fetchStarredCmd(), a.spinner.Tick)
	case next == scopeGitHub && a.query != "":
		a.searching = true
		return a, tea.Batch(a.remoteSearchCmd(a.query), a.spinner.Tick)
	}
	return a, nil
}

// editQuery mutates the search text and refreshes results for the active scope:
// instant local fuzzy filtering, or a debounced GitHub search.
func (a *App) editQuery(next string) (tea.Model, tea.Cmd) {
	a.query = next
	if a.scope != scopeGitHub && gh.IsRepoURL(next) {
		return a.setScope(scopeGitHub)
	}
	if a.scope != scopeGitHub {
		a.applyFilter()
		return a, nil
	}
	a.searchSeq++
	if a.query == "" {
		a.searching = false
		a.list.SetItems(nil)
		return a, nil
	}
	return a, a.debounceSearchCmd()
}

func (a *App) Init() tea.Cmd {
	if a.loading {
		return tea.Batch(a.fetchCmd(), a.spinner.Tick)
	}
	return nil
}

// anyLoading reports whether any screen is awaiting a network fetch (used to
// keep the spinner ticking only while needed).
func (a *App) anyLoading() bool {
	return a.loading || a.searching || a.branchLoading || a.starredLoading ||
		(a.tree != nil && (a.tree.loading || a.tree.refLoading)) || (a.readme != nil && a.readme.loading) ||
		(a.releases != nil && (a.releases.loading || a.releases.readerLoading))
}

// enterBranches opens the branch picker, fetching the repo's branches.
func (a *App) enterBranches() (tea.Model, tea.Cmd) {
	a.screen = screenBranches
	a.picker = nil
	a.branchLoading = true
	a.status = ""
	return a, tea.Batch(a.fetchBranchesCmd(), a.spinner.Tick)
}

func (a *App) fetchBranchesCmd() tea.Cmd {
	repo := a.selected.NameWithOwner
	return func() tea.Msg {
		branches, err := a.client.FetchBranches(repo)
		return branchesLoadedMsg{branches: branches, err: err}
	}
}

func (a *App) updateBranches(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	if a.branchLoading {
		switch key.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "esc":
			a.screen = screenActions
			return a, nil
		}
		return a, nil
	}
	switch p := a.picker.handleKey(key.String(), key.Runes, key.Type == tea.KeyRunes); p {
	case pickerQuit:
		return a, tea.Quit
	case pickerBack:
		a.screen = screenActions
		return a, nil
	case pickerChosen:
		a.Result = Result{Action: ActionClone, Repo: a.selected, Branch: a.picker.selection()}
		return a, tea.Quit
	}
	return a, nil
}

func (a *App) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		repos, err := a.client.FetchRepos()
		if err != nil {
			return errMsg{err}
		}
		return reposLoadedMsg{repos}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rateLimitsMsg, rateTickMsg:
		return a, a.updateRate(msg)

	case spinner.TickMsg:
		if !a.anyLoading() {
			return a, nil // let the spinner stop once nothing is loading
		}
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		lw, lh := a.listPanelSize()
		a.list.SetSize(lw-4, lh-2)
		if a.tree != nil {
			a.tree.setSize(msg.Width, msg.Height)
		}
		if a.readme != nil {
			a.readme.setSize(msg.Width, msg.Height)
		}
		if a.releases != nil {
			a.releases.setSize(msg.Width, msg.Height)
		}
		return a, nil

	case reposLoadedMsg:
		a.loading = false
		a.status = ""
		if a.saveFn != nil {
			a.saveFn(msg.repos)
		}
		a.repos = msg.repos
		if a.scope == scopeMine {
			a.applyFilter() // keep any active search applied to the fresh data
		}
		return a, nil

	case starredLoadedMsg:
		a.starredLoading = false
		if msg.err != nil {
			a.status = "Couldn't load starred repos: " + msg.err.Error()
			return a, nil
		}
		a.starred = msg.repos
		if a.starredSave != nil {
			a.starredSave(msg.repos)
		}
		if a.scope == scopeStarred {
			a.applyFilter()
		}
		return a, nil

	case errMsg:
		a.loading = false
		// A failed background refresh is non-fatal when we already show cache.
		if len(a.list.Items()) == 0 {
			a.err = msg.err
			return a, tea.Quit
		}
		a.status = "Refresh failed: " + msg.err.Error()
		return a, nil

	case debounceMsg:
		// Fire the search only if no newer keystroke arrived and we're still in
		// GitHub scope with a non-empty query.
		if msg.seq != a.searchSeq || a.scope != scopeGitHub || a.query == "" {
			return a, nil
		}
		a.searching = true
		a.status = ""
		return a, tea.Batch(a.remoteSearchCmd(a.query), a.spinner.Tick)

	case remoteReposMsg:
		a.searching = false
		if msg.query != a.query || a.scope != scopeGitHub {
			return a, nil // a stale response for an older query
		}
		if msg.err != nil {
			a.status = "GitHub search failed: " + msg.err.Error()
			a.list.SetItems(nil)
			return a, nil
		}
		a.status = ""
		a.remote = msg.repos
		a.applyFilter()
		return a, nil

	case branchesLoadedMsg:
		a.branchLoading = false
		if msg.err != nil {
			// Non-fatal: drop back to the action menu with the failure visible in
			// the status line, so the bounce-back is explained.
			a.status = "Couldn't load branches: " + msg.err.Error()
			a.screen = screenActions
			return a, nil
		}
		a.picker = newPicker(msg.branches)
		return a, nil

	case readmeLoadedMsg:
		if a.readme != nil {
			a.readme.loaded(msg)
		}
		return a, nil

	case releasesLoadedMsg:
		if a.releases != nil {
			a.releases.loaded(msg)
		}
		return a, nil

	case releasesStatusMsg:
		if a.releases != nil {
			a.releases.status = msg.text
		}
		return a, nil

	case assetPreviewMsg:
		if a.releases != nil {
			a.releases.previewLoaded(msg)
		}
		return a, nil
	}

	switch a.screen {
	case screenList:
		return a.updateList(msg)
	case screenActions:
		return a.updateActions(msg)
	case screenBranches:
		return a.updateBranches(msg)
	case screenTree:
		return a.updateTree(msg)
	case screenReadme:
		return a.updateReadme(msg)
	case screenConfirmDelete:
		return a.updateConfirmDelete(msg)
	case screenReleases:
		return a.updateReleases(msg)
	}
	return a, nil
}

func (a *App) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		return a, cmd
	}

	if a.listHelp {
		switch key.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "?", "esc", "q", "enter":
			a.listHelp = false
		}
		return a, nil
	}

	switch key.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "?":
		if a.query == "" {
			return a.openListHelp()
		}
	case "tab":
		return a.toggleScope()
	case "ctrl+s":
		a.sortBy = nextSort(a.sortBy, a.scope)
		a.applyFilter()
		return a, nil
	case "ctrl+x":
		// ctrl-prefixed, because bare keys are search input on this screen.
		return a.toggleMark()
	case "ctrl+d":
		a.confirmReturn = screenList
		return a.enterConfirmDelete(a.markedRepos())
	case "enter":
		if it, ok := a.list.SelectedItem().(repoItem); ok {
			a.selected = it.repo
			if a.printPath {
				a.Result = Result{
					Action:    ActionPrintPath,
					Repo:      it.repo,
					LocalPath: a.localPath(it.repo.NameWithOwner),
				}
				return a, tea.Quit
			}
			a.screen = screenActions
			a.actions = a.menuFor(it.repo)
			a.actionCursor = 0
			a.status = ""
		}
		return a, nil
	case "esc":
		// esc clears the query; on an empty query it leaves GitHub scope, and
		// from an empty My-repos search it quits.
		if a.query != "" {
			return a.editQuery("")
		}
		if a.scope != scopeMine {
			return a.setScope(scopeMine)
		}
		return a, tea.Quit
	case "backspace":
		if a.query != "" {
			r := []rune(a.query)
			return a.editQuery(string(r[:len(r)-1]))
		}
		return a, nil
	case "up", "down", "pgup", "pgdown", "home", "end":
		// Navigation keys move the selection; everything else is search input.
		a.markNote = ""
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		return a, cmd
	}

	if key.Type == tea.KeyRunes {
		return a.editQuery(a.query + string(key.Runes))
	}
	return a, nil
}

func (a *App) enterTree() (tea.Model, tea.Cmd) {
	a.tree = newTreeModel(a.client, a.selected, a.width, a.height)
	a.tree.saveDir = a.saveDir
	a.screen = screenTree
	return a, tea.Batch(a.tree.fetchTreeCmd(), a.spinner.Tick)
}

func (a *App) updateTree(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd, outcome := a.tree.update(msg)
	switch outcome {
	case treeQuit:
		return a, tea.Quit
	case treeBack:
		a.screen = screenActions
		return a, nil
	case treeConfirm:
		folders := a.tree.selectedFolders()
		files := a.tree.selectedFiles()
		if len(folders) == 0 && len(files) == 0 {
			a.tree.status = "select at least one folder or file (space) before cloning"
			return a, nil
		}
		a.Result = Result{Action: ActionSparseClone, Repo: a.selected, Branch: a.tree.ref, Folders: folders, Files: files}
		return a, tea.Quit
	}
	return a, cmd
}

func (a *App) enterReadme() (tea.Model, tea.Cmd) {
	a.readme = newReadmeModel(a.client, a.selected, a.width, a.height)
	a.screen = screenReadme
	a.status = ""
	return a, tea.Batch(a.readme.fetchCmd(), a.spinner.Tick)
}

func (a *App) updateReadme(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch key.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "esc", "q", "left", "h", "backspace":
		a.screen = screenActions
		return a, nil
	}
	var cmd tea.Cmd
	a.readme.view, cmd = a.readme.view.Update(msg)
	return a, cmd
}

func (a *App) enterReleases() (tea.Model, tea.Cmd) {
	a.releases = newReleasesModel(a.client, a.selected, a.width, a.height)
	a.screen = screenReleases
	a.status = ""
	return a, tea.Batch(a.releases.fetchCmd(), a.spinner.Tick)
}

func (a *App) updateReleases(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd, outcome := a.releases.update(msg)
	switch outcome {
	case releasesQuit:
		return a, tea.Quit
	case releasesBack:
		a.screen = screenActions
		return a, nil
	case releasesDownload:
		rel, _ := a.releases.current()
		a.Result = Result{
			Action:    ActionDownload,
			Repo:      a.selected,
			Release:   rel.TagName,
			Assets:    a.releases.chosen(),
			Checksums: a.releases.checksumAssets(),
			Extract:   a.releases.extract,
		}
		return a, tea.Quit
	}
	if cmd != nil && a.releases.readerLoading {
		return a, tea.Batch(cmd, a.spinner.Tick)
	}
	return a, cmd
}

func (a *App) View() string {
	return paintBackground(a.view(), a.width)
}

func (a *App) view() string {
	if a.err != nil {
		return errStyle.Render("Error: "+a.err.Error()) + "\n"
	}
	switch a.screen {
	case screenActions:
		status := ""
		if a.status != "" {
			status = statusStyle.Render(a.status)
		}
		return compose(a.width, a.height, a.version,
			a.selected.NameWithOwner, a.viewActions(), status, a.actionsFooter())
	case screenBranches:
		return a.viewBranches()
	case screenTree:
		context, body, status, keys := a.tree.chromeParts(a.spinner.View(), a.height-chromeLines)
		return compose(a.width, a.height, a.version, context, body, status, keys)
	case screenReadme:
		context, body, status, keys := a.readme.chromeParts(a.spinner.View(), a.width, a.height-chromeLines)
		return compose(a.width, a.height, a.version, context, body, status, keys)
	case screenConfirmDelete:
		return a.viewConfirmDelete()
	case screenReleases:
		context, body, status, keys := a.releases.chromeParts(a.spinner.View(), a.height-chromeLines)
		return compose(a.width, a.height, a.version, context, body, status, keys)
	default:
		return compose(a.width, a.height, a.version,
			"", a.viewList(), a.listStatus(), a.listFooter())
	}
}

// listPanelTitle names the repo panel with the active scope, result count, and
// how many repos are marked for deletion (marks survive scope and query
// changes, so the count has to stay visible).
func (a *App) listPanelTitle() string {
	names := map[searchScope]string{scopeMine: "my repos", scopeStarred: "starred", scopeGitHub: "github"}
	title := names[a.scope]
	total := len(a.repos)
	if a.scope == scopeStarred {
		total = len(a.starred)
	}
	switch {
	case a.scope == scopeGitHub && a.query == "":
	case a.scope == scopeGitHub:
		title += fmt.Sprintf(" · %d", len(a.list.Items()))
	case a.query == "":
		title += fmt.Sprintf(" · %d", total)
	default:
		title += fmt.Sprintf(" · %d/%d", len(a.list.Items()), total)
	}
	if a.sortBy != sortDefault {
		title += " · by " + sortLabel(a.sortBy, a.scope)
	}
	if n := len(a.marked); n > 0 {
		title += fmt.Sprintf(" · %d to delete", n)
	}
	return title
}

// viewList renders the search panel, the repo panel, and (on wide terminals) a
// details panel for the highlighted repo. Background search/refresh progress
// lives in the chrome's status line (listStatus), so stale results stay on
// screen while a new GitHub search is in flight instead of blanking away on
// every debounced keystroke.
func (a *App) viewList() string {
	placeholder := "filter your repos… try lang:go or is:private"
	switch a.scope {
	case scopeStarred:
		placeholder = "filter your starred repos…"
	case scopeGitHub:
		placeholder = "search GitHub: torvalds/linux, an owner, a name or a pasted URL"
	}
	cw := contentWidth(a.width)
	search := panel("search", searchField(a.query, placeholder), cw, searchBoxLines, true)
	if a.listHelp {
		_, lh := a.listPanelSize()
		return search + "\n" + panel("help", listHelpBody()+"\n"+a.rateLimitsBody(), cw, lh, true)
	}

	body := a.list.View()
	if len(a.list.Items()) == 0 {
		switch {
		case a.searching, a.status != "", a.scope == scopeStarred && a.starredLoading:
			body = "" // the status line explains the empty panel
		case a.scope == scopeStarred && a.query == "":
			body = dimStyle.Render("no starred repositories")
		case a.scope == scopeGitHub && a.query == "":
			body = dimStyle.Render("type a username, repo, or owner/name, e.g. torvalds/linux")
		case a.query != "" && a.scope != scopeGitHub:
			body = dimStyle.Render("no repositories match “"+a.query+"”") + "\n\n" +
				keyStyle.Render("tab") + dimStyle.Render(" to search all of GitHub for it")
		case a.query != "":
			body = dimStyle.Render("no repositories match “" + a.query + "”")
		default:
			body = dimStyle.Render("no repositories")
		}
	}

	lw, lh := a.listPanelSize()
	repos := panel(a.listPanelTitle(), body, lw, lh, false)
	if _, sw := splitWidths(cw); sw > 0 {
		detail := panel("details", dimStyle.Render("no selection"), sw, lh, false)
		if it, ok := a.list.SelectedItem().(repoItem); ok {
			detail = detailColumn(it.repo, a.localPath(it.repo.NameWithOwner), sw, lh)
		}
		repos = hsplit(repos, detail)
	}
	return search + "\n" + repos
}

// listStatus feeds the chrome's status line: in-flight work first, then any
// sticky message from a failed refresh or search.
func (a *App) listStatus() string {
	switch {
	case a.searching:
		return a.spinner.View() + statusStyle.Render(" Searching GitHub…")
	case a.starredLoading && a.scope == scopeStarred:
		return a.spinner.View() + statusStyle.Render(" Loading starred repositories…")
	case a.loading:
		return a.spinner.View() + statusStyle.Render(" Refreshing repositories…")
	case a.status != "":
		return statusStyle.Render(a.status)
	case a.markNote != "":
		return a.markNote
	}
	return ""
}

// viewBranches renders the branch picker (or its loading state) as a search
// panel plus a branches panel, with the repo's details alongside when wide.
func (a *App) viewBranches() string {
	context := a.selected.NameWithOwner
	if a.branchLoading || a.picker == nil {
		status := a.spinner.View() + statusStyle.Render(" Loading branches…")
		return compose(a.width, a.height, a.version, context, "", status, pickerFooter())
	}
	cw := contentWidth(a.width)
	search := panel("search", searchField(a.picker.query, "filter branches…"), cw, searchBoxLines, true)
	lw, lh := a.listPanelSize()
	branches := panel(a.picker.context(), a.picker.body(lh-2), lw, lh, false)
	if _, sw := splitWidths(cw); sw > 0 {
		branches = hsplit(branches, detailColumn(a.selected, a.localPath(a.selected.NameWithOwner), sw, lh))
	}
	return compose(a.width, a.height, a.version, context, search+"\n"+branches, "", pickerFooter())
}

func (a *App) listFooter() string {
	if a.listHelp {
		return keyHint([2]string{"?", "close"}, [2]string{"^C", "quit"})
	}
	scopeHint := map[searchScope][2]string{
		scopeMine:    {"tab", "starred"},
		scopeStarred: {"tab", "search GitHub"},
		scopeGitHub:  {"tab", "my repos"},
	}[a.scope]
	sep := dimStyle.Render(hintSep)
	footer := keyHint([2]string{"enter", "select"})
	if it, ok := a.list.SelectedItem().(repoItem); ok && deletable(it.repo) {
		markHint := [2]string{"^x", "mark for deletion"}
		if _, marked := a.marked[it.repo.NameWithOwner]; marked {
			markHint = [2]string{"^x", "unmark"}
		}
		footer += sep + keyHint(markHint)
	}
	if n := len(a.marked); n > 0 {
		footer += sep + dangerHint("^d", "delete "+plural(n, "repo", "repos"))
	}
	last := [2]string{"?", "all keys"}
	if a.query != "" {
		last = [2]string{"^C", "quit"}
	}
	return footer + sep + keyHint(
		scopeHint,
		[2]string{"^s", "sort: " + sortLabel(a.sortBy, a.scope)},
		[2]string{"esc", "clear"},
		[2]string{"↑↓", "move"},
		[2]string{"type", "search"},
		last,
	)
}

func listHelpBody() string {
	return helpBody("Repository list keys", [][2]string{
		{"type", "filter the list, or search GitHub"},
		{"tab", "switch: my repos → starred → GitHub search"},
		{"ctrl+s", "sort: recent · stars · name"},
		{"lang:go", "only repos in that language"},
		{"is:private", "also is:public, is:local (cloned here), is:mine"},
		{"↑↓ pgup pgdn", "move"},
		{"enter", "open the action menu"},
		{"esc", "clear the search · leave GitHub · quit"},
		{"", ""},
		{"ctrl+x", "mark or unmark a repo you own for deletion"},
		{"ctrl+d", "review and delete every marked repo"},
		{"", ""},
		{"?", "this overlay with API rate limits (on an empty search)"},
		{"ctrl+c", "quit"},
	})
}
