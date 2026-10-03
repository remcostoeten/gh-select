// Command gh-select is a GitHub CLI extension: an interactive TUI for finding
// your repositories, previewing their codebase, and cloning them — fully or by
// selecting only the folders you want (partial/sparse clone).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/cache"
	"github.com/remcostoeten/gh-select/internal/config"
	"github.com/remcostoeten/gh-select/internal/download"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/remcostoeten/gh-select/internal/gitx"
	"github.com/remcostoeten/gh-select/internal/local"
	"github.com/remcostoeten/gh-select/internal/sys"
	"github.com/remcostoeten/gh-select/internal/ui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

// resolveVersion prefers the linker-injected version (set in release builds),
// falling back to the VCS commit embedded by the Go toolchain for local builds,
// e.g. "dev+270fe07-dirty".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	var rev string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return version
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	v := "dev+" + rev
	if dirty {
		v += "-dirty"
	}
	return v
}

func main() {
	var (
		noCache     = flag.Bool("no-cache", false, "bypass cache and fetch fresh data")
		refreshOnly = flag.Bool("refresh", false, "refresh the cache and exit")
		showVer     = flag.Bool("version", false, "show version information")
		showLimits  = flag.Bool("limits", false, "show GitHub API rate limit usage and exit")
		theme       = flag.String("theme", "", "color theme (also GH_SELECT_THEME)")
		border      = flag.String("border", "", "panel border style (also GH_SELECT_BORDER)")
		transparent = flag.Bool("transparent", false, "don't paint the app background (also GH_SELECT_TRANSPARENT)")
		cloneDir    = flag.String("dir", "", "directory to clone into (also GH_SELECT_CLONE_DIR)")
		printPath   = flag.Bool("print-path", false, "print the selected repo's local path, cloning it if needed")
		downloadDir = flag.String("download-dir", "", "directory release assets and saved files go to (also GH_SELECT_DOWNLOAD_DIR)")
	)
	flag.BoolVar(noCache, "n", false, "bypass cache (shorthand)")
	flag.BoolVar(refreshOnly, "r", false, "refresh cache and exit (shorthand)")
	flag.BoolVar(showVer, "v", false, "show version (shorthand)")
	flag.StringVar(cloneDir, "d", "", "clone directory (shorthand)")
	flag.BoolVar(printPath, "p", false, "print the selected repo's local path (shorthand)")
	flag.Usage = usage
	flag.Parse()

	if *showVer {
		fmt.Printf("gh-select %s\n", resolveVersion())
		return
	}
	if *showLimits || flag.Arg(0) == "limits" {
		if err := limits(); err != nil {
			fmt.Fprintln(os.Stderr, errLine(err.Error()))
			os.Exit(1)
		}
		return
	}

	opts := options{
		noCache:     *noCache,
		refreshOnly: *refreshOnly,
		theme:       *theme,
		border:      *border,
		transparent: *transparent,
		cloneDir:    *cloneDir,
		printPath:   *printPath,
		downloadDir: *downloadDir,
	}
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, errLine(err.Error()))
		os.Exit(1)
	}
}

// options carries the parsed command line into run, which needs enough knobs
// that positional parameters had stopped being readable.
type options struct {
	noCache     bool
	refreshOnly bool
	theme       string
	border      string
	transparent bool
	cloneDir    string
	printPath   bool
	downloadDir string
}

func run(opts options) error {
	if flag.Arg(0) == "doctor" {
		return doctor()
	}

	cfg := config.Load()

	// Flags win over environment; empty keeps the built-in default.
	theme, border := opts.theme, opts.border
	if theme == "" {
		theme = cfg.Theme
	}
	if border == "" {
		border = cfg.Border
	}
	cloneDir := config.ExpandHome(opts.cloneDir)
	if cloneDir == "" {
		cloneDir = cfg.CloneDir
	}
	if err := ui.SetTheme(theme); err != nil {
		return err
	}
	if err := ui.SetBorder(border); err != nil {
		return err
	}
	ui.SetTransparent(opts.transparent || cfg.Transparent)

	// --print-path owns stdout: only the path may end up there, so the TUI and
	// git's progress both go to stderr.
	if opts.printPath {
		gitx.SetOutput(os.Stderr)
	}

	client, err := gh.NewClient()
	if err != nil {
		return err
	}

	c := cache.New(cfg.CacheDir, cfg.CacheTTL)
	noCache, refreshOnly := opts.noCache, opts.refreshOnly
	entry, cached := c.Load()

	// refresh-only: fetch synchronously, persist, and exit.
	if refreshOnly {
		repos, err := client.FetchRepos()
		if err != nil {
			return err
		}
		if err := c.Save(repos); err != nil {
			return err
		}
		fmt.Printf("Refreshed %d repositories\n", len(repos))
		return nil
	}

	// Cold start with no usable cache: fetch up front so we never show an empty
	// list.
	var initial []gh.Repo
	needRefresh := noCache || !cached || !entry.Fresh
	if cached && !noCache {
		initial = entry.Repos
	}
	if len(initial) == 0 {
		// No usable cache: fetch synchronously and tell the user we're working,
		// otherwise the terminal appears to hang for a few seconds.
		fmt.Fprintln(os.Stderr, "Loading repositories…")
		repos, err := client.FetchRepos()
		if err != nil {
			return err
		}
		_ = c.Save(repos)
		initial = repos
		needRefresh = false
	}

	downloadDir := config.ExpandHome(opts.downloadDir)
	if downloadDir == "" {
		downloadDir = cfg.DownloadDir
	}

	saveFn := func(repos []gh.Repo) { _ = c.Save(repos) }
	locals := local.Index(cloneDir)
	app := ui.NewApp(client, initial, needRefresh, saveFn, resolveVersion())
	app.SetLocalClones(locals)
	app.SetPrintPath(opts.printPath)
	app.SetSaveDir(downloadDir)
	starredCache := c.Named("starred")
	starred, _ := starredCache.Load()
	app.SetStarred(starred.Repos, func(repos []gh.Repo) { _ = starredCache.Save(repos) })

	teaOpts := []tea.ProgramOption{tea.WithAltScreen()}
	if opts.printPath {
		teaOpts = append(teaOpts, tea.WithOutput(os.Stderr))
	}
	final, err := tea.NewProgram(app, teaOpts...).Run()
	if err != nil {
		return err
	}

	return execute(final.(*ui.App).Result, env{cloneDir: cloneDir, downloadDir: downloadDir, client: client, cache: c, locals: locals})
}

// env carries what the post-TUI actions need beyond the Result itself.
type env struct {
	cloneDir    string // root every new clone lands under; "" means the cwd
	downloadDir string // where release assets land; "" means the cwd
	client      *gh.Client
	cache       *cache.Cache
	locals      map[string]string // "owner/repo" → existing working copy
}

// execute performs the side-effecting action chosen in the TUI, after the
// alternate screen has been restored.
func execute(res ui.Result, e env) error {
	cloneDir := e.cloneDir
	switch res.Action {
	case ui.ActionClone:
		_, err := cloneTo(res, cloneDir)
		return err
	case ui.ActionSparseClone:
		dir, err := destination(res.Repo.NameWithOwner, cloneDir)
		if err != nil {
			return err
		}
		paths := append(append([]string{}, res.Folders...), res.Files...)
		fmt.Printf("Partial clone of %s → %s, paths: %s\n",
			res.Repo.NameWithOwner, displayDir(dir, res.Repo.NameWithOwner), strings.Join(paths, ", "))
		return gitx.SparseClone(res.Repo.NameWithOwner, res.Folders, res.Files, res.Branch, dir)
	case ui.ActionOpenEditor:
		dir := res.LocalPath
		if dir == "" {
			var err error
			if dir, err = cloneTo(res, cloneDir); err != nil {
				return err
			}
		}
		fmt.Printf("Opening %s…\n", dir)
		return sys.OpenEditor(dir)
	case ui.ActionPull:
		fmt.Printf("Pulling %s…\n", res.LocalPath)
		return gitx.Pull(res.LocalPath)
	case ui.ActionPrintPath:
		dir := res.LocalPath
		if dir == "" {
			var err error
			if dir, err = cloneTo(res, cloneDir); err != nil {
				return err
			}
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		fmt.Println(abs)
		return nil
	case ui.ActionCopyName:
		return reportCopy(sys.Copy(res.Repo.NameWithOwner), "name", res.Repo.NameWithOwner)
	case ui.ActionCopyURL:
		return reportCopy(sys.Copy(res.Repo.URL()), "URL", res.Repo.URL())
	case ui.ActionOpenWeb:
		fmt.Printf("Opening %s…\n", res.Repo.URL())
		return sys.OpenURL(res.Repo.URL())
	case ui.ActionDelete:
		return deleteRepos(res.Deletes, e)
	case ui.ActionDownload:
		fmt.Printf("Downloading from %s %s\n", res.Repo.NameWithOwner, res.Release)
		return download.Assets(e.client, res.Assets, download.Options{
			Dir:       e.downloadDir,
			Extract:   res.Extract,
			Checksums: res.Checksums,
		})
	}
	return nil
}

// deleteRepos permanently deletes the repositories confirmed in the TUI. Each
// one is reported as it goes and a failure never strands the rest, since a
// single missing scope or stale entry shouldn't abandon a reviewed batch
// halfway. Working copies on disk are deliberately left alone — only pointed
// out — so a remote deletion can never take local work with it.
func deleteRepos(repos []gh.Repo, e env) error {
	deleted := map[string]bool{}
	var failed []string
	for _, r := range repos {
		if err := e.client.DeleteRepo(r.NameWithOwner); err != nil {
			fmt.Fprintln(os.Stderr, errLine(err.Error()))
			failed = append(failed, r.NameWithOwner)
			continue
		}
		deleted[r.NameWithOwner] = true
		fmt.Printf("Deleted %s\n", r.NameWithOwner)
		if path, ok := e.locals[r.NameWithOwner]; ok {
			fmt.Printf("  local clone left in place: %s\n", path)
		}
	}
	if len(deleted) > 0 {
		pruneCache(e.cache, deleted)
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d deletions failed: %s",
			len(failed), len(repos), strings.Join(failed, ", "))
	}
	return nil
}

// pruneCache drops the deleted repos from the cached list so the next run
// doesn't offer repositories that no longer exist.
func pruneCache(c *cache.Cache, deleted map[string]bool) {
	entry, ok := c.Load()
	if !ok {
		return
	}
	kept := make([]gh.Repo, 0, len(entry.Repos))
	for _, r := range entry.Repos {
		if !deleted[r.NameWithOwner] {
			kept = append(kept, r)
		}
	}
	_ = c.Save(kept)
}

// cloneTo clones the selected repo and returns the directory it landed in.
func cloneTo(res ui.Result, cloneDir string) (string, error) {
	dir, err := destination(res.Repo.NameWithOwner, cloneDir)
	if err != nil {
		return "", err
	}
	target := displayDir(dir, res.Repo.NameWithOwner)
	if res.Branch != "" {
		fmt.Fprintf(os.Stderr, "Cloning %s (branch %s) → %s…\n", res.Repo.NameWithOwner, res.Branch, target)
	} else {
		fmt.Fprintf(os.Stderr, "Cloning %s → %s…\n", res.Repo.NameWithOwner, target)
	}
	if err := gitx.Clone(res.Repo.NameWithOwner, res.Branch, dir); err != nil {
		return "", err
	}
	return target, nil
}

// destination resolves where a clone should go, refusing to clone onto an
// existing working copy rather than letting git fail halfway through.
func destination(nameWithOwner, cloneDir string) (string, error) {
	dir := local.Destination(cloneDir, nameWithOwner)
	if local.Exists(displayDir(dir, nameWithOwner)) {
		return "", fmt.Errorf("%s already exists", displayDir(dir, nameWithOwner))
	}
	if cloneDir != "" {
		if err := os.MkdirAll(cloneDir, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// displayDir is the path a clone actually lands in: git defaults to the repo
// name in the current directory when no destination is given.
func displayDir(dir, nameWithOwner string) string {
	if dir != "" {
		return dir
	}
	return local.RepoName(nameWithOwner)
}

func reportCopy(ok bool, label, value string) error {
	if ok {
		fmt.Printf("Copied repository %s to clipboard\n", label)
	} else {
		fmt.Printf("Repository %s: %s\n(clipboard not available)\n", label, value)
	}
	return nil
}

func errLine(s string) string { return "Error: " + s }

// doctor reports whether the tools and authentication gh-select relies on are
// present, without requiring a successful login itself.
func doctor() error {
	fmt.Printf("gh-select %s: environment check\n\n", resolveVersion())

	check := func(label string, ok bool, detail string) {
		mark := "[x] "
		if ok {
			mark = "[ok]"
		}
		fmt.Printf("  %s  %-6s %s\n", mark, label, detail)
	}

	gitPath, gitErr := exec.LookPath("git")
	check("git", gitErr == nil, orMissing(gitPath, "required for cloning"))

	ghPath, ghErr := exec.LookPath("gh")
	check("gh", ghErr == nil, orMissing(ghPath, "required for authentication"))

	authed := exec.Command("gh", "auth", "status").Run() == nil
	detail := "logged in"
	if !authed {
		detail = "not authenticated, run: gh auth login"
	}
	check("auth", authed, detail)

	fmt.Println()
	return nil
}

// limits prints the viewer's API budgets with how much of each window has
// passed, so a fast burn shows before the bucket runs dry.
func limits() error {
	client, err := gh.NewClient()
	if err != nil {
		return err
	}
	buckets, err := client.RateLimits()
	if err != nil {
		return err
	}
	now := time.Now()
	fmt.Printf("GitHub API rate limits (shared by every tool using your gh login)\n\n")
	for _, b := range buckets {
		fmt.Printf("  %-12s %s  %5d/%-5d  %-3s window  %s\n",
			b.Name, b.UsageBar(20, now), b.Used, b.Limit, gh.ShortDuration(b.Window), b.Pace(now))
	}
	fmt.Println("\n  █ used  │ how far the current window has run")
	return nil
}

func orMissing(path, missingHint string) string {
	if path == "" {
		return "not found, " + missingHint
	}
	return path
}

func usage() {
	fmt.Fprint(os.Stderr, `gh-select: interactive GitHub repository selector

Usage:
  gh select [options]
  gh select doctor          check tools and authentication
  gh select limits          show GitHub API rate limit usage (or --limits)

Options:
  -n, --no-cache   bypass cache, fetch fresh data
  -r, --refresh    refresh cache and exit
  -d, --dir DIR    clone into DIR instead of the current directory
                   (or GH_SELECT_CLONE_DIR)
  --download-dir DIR
                   save release assets and browsed files into DIR instead
                   of the current directory (or GH_SELECT_DOWNLOAD_DIR)
  -p, --print-path print the selected repo's local path on stdout, cloning it
                   first if needed, for:  cd "$(gh select -p)"
  --theme NAME     color theme: tokyonight (default), catppuccin, dracula,
                   gruvbox, nord (or GH_SELECT_THEME)
  --border NAME    panel border: rounded (default), sharp, double, thick,
                   hidden (or GH_SELECT_BORDER)
  --transparent    keep the terminal's own background instead of the theme's
                   (or GH_SELECT_TRANSPARENT=1)
  -v, --version    show version
  -h, --help       show this help

Inside the TUI:
  type to filter your repos · enter to act on a repo
  repos already cloned under --dir are badged "local" and offer open/pull
  tab to search all of GitHub (a username, repo, or owner/name)
  partial clone: pick folders with space, then press c
  releases: / filters, n reads notes, o opens the release page, y copies a
  link; in a release's downloads the file for this machine is preselected,
  v previews, space marks several, enter downloads, x downloads and unpacks.
  Downloads are verified against the release's checksum file when it has one.
  ctrl+x marks repos you own, ctrl+d deletes every marked one on GitHub
  (permanent; retype the shown phrase to confirm, and note that deleting
  needs the delete_repo scope: gh auth refresh -s delete_repo)
`)
}
