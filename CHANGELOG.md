# Changelog

All notable changes to gh-select are documented here.

## [Unreleased]

## [2.5.0] - 2026-10-03

### Added
- **GitHub search understands more input.** It takes `owner/repo`,
  `owner repo`, `owner-repo`, a bare owner or name, URLs with any scheme,
  `www.` or TLD, SSH remotes, `.git` suffixes and whole `git clone`
  commands. An exact repo is pinned on top, an owner's repos match by
  partial name, and a misspelled owner falls back to similar accounts.
  Pasting a URL into My repos or Starred switches to GitHub search.
- **API rate limits.** `gh select limits` (`-l`, `--limits`) and the `?`
  overlay show the search, core, graphql and code search budgets: usage,
  how far each window has run, time to reset, and how soon the current pace
  would run a bucket dry.
- **Clones are found anywhere under your home directory**, not only under
  the clone root. The scan runs in the background, skips dependency and
  cache folders, and is cached for the next start. Worktrees and submodules
  count, and when a repo is cloned twice the most recently used copy wins.
  `GH_SELECT_SCAN_DIRS` lists directories to search first.
- **Fork details.** Forks are tagged in the list, the details show the
  parent repo, and `is:fork` and `is:source` filter on it.
- **Every language** a repo uses is listed in the details, largest first.
- **Themes.** `everforest`, `graphite`, `kanagawa`, `rose-pine` and
  `solarized` join the built-in set. Custom themes load from
  `~/.config/gh-select/themes/NAME.json` and inherit unset colors through
  `extends`. `ctrl+t` cycles themes and saves the choice.
- **Credits in the `?` overlay**, with one-key links to the author's GitHub
  and website, the source, a new issue and the release notes.
- **`shift+tab`** switches scope backwards.

### Changed
- **Command line.** `doctor`, `limits`, `refresh`, `version` and `help`
  each work as a word, a short flag and a long flag, and take `-h` for
  their own help page. Help is grouped into commands, options, environment
  and examples, and is colored unless `NO_COLOR` is set. Unknown commands
  and options exit with status 2 and suggest the closest match instead of
  opening the picker.
- **`--print-path` exits 130** when you quit without a pick, so
  `dir=$(gh select -p) && cd "$dir"` no longer runs `cd ""`.
- **Redesigned screen.** Panels are frameless by default (`--border
  rounded` brings the boxes back), scope tabs sit above the search, the
  list has aligned name, language and star columns, and a details block
  below it shows the repo's URL and local path. The header shows the
  version and build, and a hairline separates the key hints.
- **Nerd Font icons** mark private repos, forks, local clones and stars.
  `--icons none` or `GH_SELECT_ICONS=none` shows words instead.
- **The default theme is now `catppuccin`.** `--theme tokyonight` keeps the
  old look.
- **`ctrl+c` clears a search or filter box first** and quits on the second
  press.
- The details block drops your own username from repo names.
- The repo cache is refetched once after upgrading to pick up fork data.

### Fixed
- `space` types a space in the repo list search, so `lang:go alpha` works.
- Backspace goes back from the action menu, and from the branch picker once
  its filter is empty.

## [2.4.0] - 2026-09-30

### Added
- **Starred repositories**: `tab` now cycles my repos → starred → GitHub
  search. Starred repos show from cache instantly and refresh once per
  session.
- **Sort and filter the list**: `ctrl+s` cycles recent / stars / name (plus
  updated for starred and search results), shown in the panel title. Filter
  tokens in the search box: `lang:go`, `is:private`, `is:public`, `is:local`,
  `is:mine`; in GitHub search they become GitHub's own qualifiers where one
  exists.
- **Branches and tags in the tree browser**: `b` lists both and reloads the
  tree at the one you pick. Preview, save, copy and partial clone all follow
  it.
- **File links in the tree browser**: `o` opens the file or folder on
  github.com, `Y` copies a permalink pinned to the current commit, `r` copies
  the raw download URL.
- **Save and copy from the tree browser**: `s` saves the marked files and
  folders (or the highlighted one) straight from the GitHub API, no git
  needed, into `--download-dir` / `GH_SELECT_DOWNLOAD_DIR` or the current
  directory. Existing files are kept and the new copy gets a `-1` suffix. `y`
  copies a file's contents to the clipboard, from the listing or the preview.
- **Releases**: a **Releases…** action lists a repository's last 100
  releases with their notes rendered alongside (`n` reads them full-screen,
  `o` opens the release page, `y` copies its link, `/` filters). `enter` opens
  a release's downloads, the uploaded assets plus the generated source archives,
  with the file for this OS and CPU preselected and tagged `◆ this machine`.
  `space` marks several, `a` marks all, `v` previews a text asset, `y` copies
  a download link, `enter` downloads, and `x` downloads and unpacks
  `.tar.gz` / `.tgz` / `.tar.bz2` / `.zip` archives (refusing entries that
  would escape the destination; links are skipped). Downloads are verified
  against the release's checksum files (GNU, BSD and per-file `.sha256`
  formats) and a mismatch discards the file. Files land in the current
  directory, or `--download-dir` / `GH_SELECT_DOWNLOAD_DIR`; existing files are
  never overwritten, a download only takes its real name once complete, and
  Ctrl+C removes the partial file. Works for private repositories.
- **Rendered markdown**: `.md` files opened in the tree browser are rendered as
  prose (headings, lists, links, highlighted code blocks) instead of shown as
  raw source, themed from the active palette. A **View README** action opens the
  repository's readme the same way, without hunting for it in the tree; GitHub
  resolves which file that is, so `docs/`, `.github/`, and odd casing all work.
- **Delete repositories, single or in bulk**: `ctrl+x` marks a repository in
  the list (marked rows get a `✗` and the panel title counts them) and `ctrl+d`
  opens a confirmation screen for every marked one; a single repository can
  also be deleted from its action menu. Deletion is permanent, so it is gated
  behind retyping a phrase (the repository's name for one, `DELETE <n>` for a
  batch) and is only offered for repositories you own. Local working copies
  are never touched, only pointed out. Requires the `delete_repo` scope:
  `gh auth refresh -s delete_repo`.
- **Clone directory**: `-d, --dir DIR` / `GH_SELECT_CLONE_DIR` sets a clone
  root so repositories land there instead of the current working directory.
  Cloning refuses to overwrite an existing working copy.
- **Local clone awareness**: repositories already cloned under the clone root
  are badged `local` in the list, show their path in the info panel, and their
  action menu leads with **Open in editor** (`$VISUAL`/`$EDITOR`) and **Pull**
  instead of a redundant clone. Detection scans one level deep plus
  `owner/repo` layouts and matches on the git remote, so renamed directories
  are still recognised.
- **`-p, --print-path`**: prints the selected repository's absolute path on
  stdout, cloning it first if needed, so a shell wrapper can `cd` into it:
  `ghcd() { local dir; dir=$(gh select --print-path) && cd "$dir"; }`.
  The TUI and git's progress move to stderr in this mode.
- **Open in editor** as an action for any repository (`Clone & open` when it
  isn't cloned yet).

### Changed
- Files over 1 MB now load in the tree browser (preview, save and copy) via the
  git blob API instead of coming back empty.
- Repository rows line up in columns: name, `local` / `private` tags, a
  language with its GitHub color dot, and a right-aligned star count shortened
  to `★1.2k`. Repos you don't own are dimmed in "my repos".
- The details column is one card titled with the repository: description
  first, then access, language, stars, last update and local path.
- Footers drop whole hints on narrow terminals instead of cutting one off, and
  `?` on an empty search opens the full repo-list keymap. Footer key colors now
  follow the theme on every screen.
- Marking for deletion explains itself: `^x mark for deletion` / `unmark` in
  the footer, a confirmation line on each mark, a red `^d delete N repos` hint,
  and marked rows struck through.
- Action menu numbers are right-aligned so `10` stays in line, and its hints
  are short enough to fit beside the details column.
- Repository fetch now includes organization and collaborator repositories, not
  just owned ones (`affiliations` *and* `ownerAffiliations` set to
  `[OWNER, ORGANIZATION_MEMBER, COLLABORATOR]`; the latter defaults to
  `[OWNER, COLLABORATOR]` and would otherwise drop every org-owned repo).
  Results are de-duplicated, preserving recency order.
- The action menu is built per repository, so its number shortcuts vary with
  the entries on offer.
- **Rewritten in Go** as a precompiled `gh` extension (was a Bash script).
  Authentication, REST + GraphQL access via `go-gh`; TUI via Bubble Tea.
  `fzf` and `jq` are no longer required.

### Removed
- The legacy Bash implementation (`legacy/gh-select.sh`) and its shell
  installer scripts (`scripts/install.sh`, `scripts/uninstall.sh`), superseded
  by the Go binary distributed via `gh extension install`.

### Added (Go rewrite)
- **Codebase tree browser**: navigate a repository's full file tree and preview
  file contents without cloning (single recursive tree API call).
- **Partial / sparse clone**: multi-select folders and clone only those via
  `git clone --filter=blob:none --sparse` + cone-mode sparse-checkout.
- Type-to-filter (`/`) within the tree browser for large repositories.
- Stale-while-revalidate cache: the list renders instantly and refreshes in the
  background, with a loading spinner during fetches.
- `gh select doctor`: checks for git, gh, and authentication.

### Performance
- Repository fetch drops the expensive `defaultBranchRef` GraphQL field
  (~3.5s/page → ~1.4s/page); tree/clone default to HEAD instead.

## [1.0.2] - 2025-07-05

### Added
- Comprehensive uninstallation support
- Global installation support

### Fixed
- Extension name must start with gh- prefix
- Remove tag from extension.yml for proper installation

---

## [1.0.1] - 2025-07-03

### Fixed
- Extension.yml configuration for proper installation

---

## [1.0.0] - 2025-07-01

### Added
- Initial release
- Interactive repository selection with fuzzy search
- Clone repositories to any location
- Copy repository names or URLs to clipboard
- Open repositories in browser
- Beautiful fzf interface with live preview
