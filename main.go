// Command gh-select is a GitHub CLI extension: an interactive TUI for finding
// your repositories, previewing their codebase, and cloning them — fully or by
// selecting only the folders you want (partial/sparse clone).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/cache"
	"github.com/remcostoeten/gh-select/internal/config"
	"github.com/remcostoeten/gh-select/internal/gh"
	"github.com/remcostoeten/gh-select/internal/gitx"
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
	)
	flag.BoolVar(noCache, "n", false, "bypass cache (shorthand)")
	flag.BoolVar(refreshOnly, "r", false, "refresh cache and exit (shorthand)")
	flag.BoolVar(showVer, "v", false, "show version (shorthand)")
	flag.Usage = usage
	flag.Parse()

	if *showVer {
		fmt.Printf("gh-select %s\n", resolveVersion())
		return
	}

	if err := run(*noCache, *refreshOnly); err != nil {
		fmt.Fprintln(os.Stderr, errLine(err.Error()))
		os.Exit(1)
	}
}

func run(noCache, refreshOnly bool) error {
	if flag.Arg(0) == "doctor" {
		return doctor()
	}

	cfg := config.Load()

	client, err := gh.NewClient()
	if err != nil {
		return err
	}

	c := cache.New(cfg.CacheDir, cfg.CacheTTL)
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

	saveFn := func(repos []gh.Repo) { _ = c.Save(repos) }
	app := ui.NewApp(client, initial, needRefresh, saveFn, resolveVersion())

	final, err := tea.NewProgram(app, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}

	res := final.(*ui.App).Result
	return execute(res)
}

// execute performs the side-effecting action chosen in the TUI, after the
// alternate screen has been restored.
func execute(res ui.Result) error {
	switch res.Action {
	case ui.ActionClone:
		if res.Branch != "" {
			fmt.Printf("Cloning %s (branch %s)…\n", res.Repo.NameWithOwner, res.Branch)
		} else {
			fmt.Printf("Cloning %s…\n", res.Repo.NameWithOwner)
		}
		return gitx.Clone(res.Repo.NameWithOwner, res.Branch, "")
	case ui.ActionSparseClone:
		paths := append(append([]string{}, res.Folders...), res.Files...)
		fmt.Printf("Partial clone of %s — paths: %s\n",
			res.Repo.NameWithOwner, strings.Join(paths, ", "))
		return gitx.SparseClone(res.Repo.NameWithOwner, res.Folders, res.Files, res.Branch, "")
	case ui.ActionCopyName:
		return reportCopy(sys.Copy(res.Repo.NameWithOwner), "name", res.Repo.NameWithOwner)
	case ui.ActionCopyURL:
		return reportCopy(sys.Copy(res.Repo.URL()), "URL", res.Repo.URL())
	case ui.ActionOpenWeb:
		fmt.Printf("Opening %s…\n", res.Repo.URL())
		return sys.OpenURL(res.Repo.URL())
	}
	return nil
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
	fmt.Printf("gh-select %s — environment check\n\n", resolveVersion())

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
		detail = "not authenticated — run: gh auth login"
	}
	check("auth", authed, detail)

	fmt.Println()
	return nil
}

func orMissing(path, missingHint string) string {
	if path == "" {
		return "not found — " + missingHint
	}
	return path
}

func usage() {
	fmt.Fprint(os.Stderr, `gh-select — interactive GitHub repository selector

Usage:
  gh select [options]
  gh select doctor          check tools and authentication

Options:
  -n, --no-cache   bypass cache, fetch fresh data
  -r, --refresh    refresh cache and exit
  -v, --version    show version
  -h, --help       show this help

Inside the TUI:
  type to filter your repos · enter to act on a repo
  tab to search all of GitHub (a username, repo, or owner/name)
  partial clone: pick folders with space, then press c
`)
}
