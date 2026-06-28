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
	scopeMine   searchScope = iota // fuzzy-filter the viewer's cached repos
	scopeGitHub                    // query the GitHub search API for any repo
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
	ActionCopyName
	ActionCopyURL
	ActionOpenWeb
)

// Result is what the program should do once the TUI returns.
type Result struct {
	Action  ActionType
	Repo    gh.Repo
	Branch  string   // non-empty to clone a specific branch
	Folders []string // selected folders, for ActionSparseClone
	Files   []string // selected individual files, for ActionSparseClone
}

type screen int

const (
	screenList screen = iota
	screenActions
	screenBranches
	screenTree
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

	// action menu
	actionCursor int

	// branch picker
	picker        *pickerModel
	branchLoading bool

	// tree browser
	tree *treeModel

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
		spinner: sp,
		loading: refresh,
		width:   80,
		height:  24,
	}
	a.list = newRepoList(initial, a.width, a.height-chromeLines-searchBoxLines)
	return a
}

// applyFilter recomputes the visible repo items for the current query and
// resets the selection to the top match.
func (a *App) applyFilter() {
	a.list.SetItems(filterRepos(a.repos, a.query))
	a.list.Select(0)
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
		repos, err := a.client.SearchRepos(query)
		return remoteReposMsg{query: query, repos: repos, err: err}
	}
}

// toggleScope flips between filtering the viewer's repos and searching all of
// GitHub, re-applying the current query under the new scope.
func (a *App) toggleScope() (tea.Model, tea.Cmd) {
	a.status = ""
	if a.scope == scopeMine {
		a.scope = scopeGitHub
		a.searchSeq++
		if a.query == "" {
			a.list.SetItems(nil) // wait for input before hitting the API
			return a, nil
		}
		a.searching = true
		return a, tea.Batch(a.remoteSearchCmd(a.query), a.spinner.Tick)
	}
	a.scope = scopeMine
	a.searching = false
	a.searchSeq++ // invalidate any in-flight GitHub response
	a.applyFilter()
	return a, nil
}

// editQuery mutates the search text and refreshes results for the active scope:
// instant local fuzzy filtering, or a debounced GitHub search.
func (a *App) editQuery(next string) (tea.Model, tea.Cmd) {
	a.query = next
	if a.scope == scopeMine {
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
	return a.loading || a.searching || a.branchLoading || (a.tree != nil && a.tree.loading)
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
	case spinner.TickMsg:
		if !a.anyLoading() {
			return a, nil // let the spinner stop once nothing is loading
		}
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.list.SetSize(msg.Width, msg.Height-chromeLines-searchBoxLines)
		if a.tree != nil {
			a.tree.setSize(msg.Width, msg.Height)
		}
		return a, nil

	case reposLoadedMsg:
		a.loading = false
		a.status = ""
		if a.saveFn != nil {
			a.saveFn(msg.repos)
		}
		a.repos = msg.repos
		a.applyFilter() // keep any active search applied to the fresh data
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
		a.list.SetItems(repoItems(msg.repos))
		a.list.Select(0)
		return a, nil

	case branchesLoadedMsg:
		a.branchLoading = false
		if msg.err != nil {
			// Non-fatal: report and drop back to the action menu.
			a.screen = screenActions
			return a, nil
		}
		a.picker = newPicker(msg.branches)
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

	switch key.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "tab":
		return a.toggleScope()
	case "enter":
		if it, ok := a.list.SelectedItem().(repoItem); ok {
			a.selected = it.repo
			a.screen = screenActions
			a.actionCursor = 0
		}
		return a, nil
	case "esc":
		// esc clears the query; on an empty query it leaves GitHub scope, and
		// from an empty My-repos search it quits.
		if a.query != "" {
			return a.editQuery("")
		}
		if a.scope == scopeGitHub {
			return a.toggleScope()
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
		a.Result = Result{Action: ActionSparseClone, Repo: a.selected, Folders: folders, Files: files}
		return a, tea.Quit
	}
	return a, cmd
}

func (a *App) View() string {
	if a.err != nil {
		return errStyle.Render("Error: "+a.err.Error()) + "\n"
	}
	switch a.screen {
	case screenActions:
		return compose(a.width, a.height, a.version,
			a.selected.NameWithOwner, a.viewActions(), actionsFooter)
	case screenBranches:
		return a.viewBranches()
	case screenTree:
		context, body, keys := a.tree.chromeParts(a.spinner.View(), a.height-chromeLines)
		return compose(a.width, a.height, a.version, context, body, keys)
	default:
		return compose(a.width, a.height, a.version, a.listContext(), a.viewList(), a.listFooter())
	}
}

// listContext is the right-hand header text for the repo list: the active scope
// and a result count. The live query itself lives in the search box below.
func (a *App) listContext() string {
	if a.scope == scopeGitHub {
		if a.query == "" {
			return contextStyle.Render("GitHub")
		}
		return contextStyle.Render("GitHub") + dimStyle.Render(fmt.Sprintf(" · %d", len(a.list.Items())))
	}
	if a.query == "" {
		return dimStyle.Render(fmt.Sprintf("My repos · %d", len(a.repos)))
	}
	return dimStyle.Render("My repos · ") + fmt.Sprintf("%d/%d", len(a.list.Items()), len(a.repos))
}

// searchBox renders the always-on search field for the repo list, with a
// scope-appropriate placeholder when nothing has been typed yet.
func (a *App) searchBox() string {
	placeholder := "filter your repos…"
	if a.scope == scopeGitHub {
		placeholder = "search GitHub — e.g. torvalds/linux"
	}
	return inputBox(searchField(a.query, placeholder), a.width)
}

// viewList renders the search box, the repo list body, and any background
// search/refresh status. When the list is empty it substitutes a
// context-appropriate line for the list's blunt built-in "No items." placeholder.
func (a *App) viewList() string {
	box := a.searchBox()

	if a.searching {
		return box + "\n  " + a.spinner.View() + statusStyle.Render(" Searching GitHub…")
	}
	if len(a.list.Items()) == 0 {
		var msg string
		switch {
		case a.status != "":
			msg = statusStyle.Render(a.status)
		case a.scope == scopeGitHub && a.query == "":
			msg = dimStyle.Render("type a username, repo, or owner/name — e.g. torvalds/linux")
		case a.query != "":
			msg = dimStyle.Render("no repositories match “" + a.query + "”")
		default:
			msg = dimStyle.Render("no repositories")
		}
		return box + "\n  " + msg
	}

	view := box + "\n" + a.list.View()
	if a.loading {
		view += "\n" + a.spinner.View() + statusStyle.Render(" Refreshing repositories…")
	} else if a.status != "" {
		view += "\n" + statusStyle.Render(a.status)
	}
	return view
}

// viewBranches renders the branch picker (or its loading state) in chrome.
func (a *App) viewBranches() string {
	context := a.selected.NameWithOwner
	if a.branchLoading || a.picker == nil {
		body := "\n  " + a.spinner.View() + statusStyle.Render(" Loading branches…")
		return compose(a.width, a.height, a.version, context, body, pickerFooter)
	}
	context = a.selected.NameWithOwner + dimStyle.Render("  ·  ") + a.picker.context()
	box := inputBox(searchField(a.picker.query, "filter branches…"), a.width)
	body := box + "\n" + a.picker.body(a.height-chromeLines-searchBoxLines)
	return compose(a.width, a.height, a.version, context, body, pickerFooter)
}

func (a *App) listFooter() string {
	scopeHint := [2]string{"tab", "search GitHub"}
	if a.scope == scopeGitHub {
		scopeHint = [2]string{"tab", "my repos"}
	}
	return keyHint(
		[2]string{"↑↓", "move"},
		[2]string{"type", "search"},
		scopeHint,
		[2]string{"⏎", "select"},
		[2]string{"esc", "clear"},
		[2]string{"^C", "quit"},
	)
}
