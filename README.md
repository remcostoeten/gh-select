<h1 align="center">gh-select</h1>

<p align="center">A <code>gh</code> extension to find, browse and clone GitHub repositories from one terminal screen.</p>

<p align="center">
  <a href="https://github.com/remcostoeten/gh-select/releases/latest"><img src="https://shieldcn.dev/github/remcostoeten/gh-select/release.svg?font=jetbrains-mono" alt="release" /></a>
  <a href="https://github.com/remcostoeten/gh-select/actions/workflows/ci.yml"><img src="https://shieldcn.dev/github/remcostoeten/gh-select/ci.svg?font=jetbrains-mono" alt="CI" /></a>
  <a href="LICENSE"><img src="https://shieldcn.dev/github/remcostoeten/gh-select/license.svg?font=jetbrains-mono" alt="license" /></a>
  <img src="https://shieldcn.dev/badge/runs%20as-gh%20extension-black.svg?font=jetbrains-mono&logo=github" alt="runs as: gh extension" />
  <img src="https://shieldcn.dev/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows%20%7C%20FreeBSD-black.svg?font=jetbrains-mono" alt="platforms: macOS, Linux, Windows, FreeBSD" />
</p>

<p align="center">
  <img src="assets/gh-select-demo.gif" width="100%" alt="Filtering your own repos with is:public, tabbing to a GitHub search for charmbracelet bubbletea, browsing its file tree, marking the examples and tutorials folders, opening a folder, previewing the rendered README, then cycling color themes with ctrl+t" />
</p>
<p align="center">
  <sub>Recorded with vhs in bash, catppuccin theme. Re-record with <code>scripts/dev.sh demo</code>.</sub>
</p>

`gh select` opens a searchable list of your repositories, your starred repos or all of GitHub. Pick one to clone it, open it, read its releases or browse its files without cloning, then clone only the folders you marked.

- **Search** your own, organization and collaborator repos from a local cache, your starred repos, or all of GitHub by name, owner or pasted URL.
- **Browse** any repository's file tree and preview files, with markdown rendered, before cloning anything.
- **Partial clone** only the folders you mark, with `git clone --filter=blob:none --sparse`.
- **Local clones** anywhere under your home directory are tagged, and their menu offers open in editor and pull.
- **Releases** list their notes and assets, preselect the build for your machine, verify checksums and unpack archives.
- **Delete** repositories you own, one at a time or in bulk, behind a retyped confirmation.
- **Jump** into a picked repo with `--print-path` and a one-line shell function.

The full changelog is in [CHANGELOG.md](CHANGELOG.md).

## Usage

```bash
gh select
```

Type to filter the list. `tab` and `shift+tab` switch between your repos, your starred repos and GitHub search, `ctrl+s` changes the sort and `ctrl+t` cycles themes. `?` on an empty search shows every key, your GitHub API rate limits and links to the source and issues. `enter` opens the action menu for the highlighted repo.

### Search and filters

GitHub search takes a name, an owner, `owner/repo`, `owner repo`, `owner-repo`, a URL, an SSH remote or a whole `git clone` command. A partial owner such as `heygen liveav` still finds `heygen-com/liveavatar-*`. Pasting a URL into any list switches to GitHub search.

Filter tokens work in every list and combine with text, as in `cli lang:go is:mine`:

| Token | Keeps |
| --- | --- |
| `lang:go` | repos whose main language is Go |
| `is:public`, `is:private` | public or private repos |
| `is:fork`, `is:source` | forks or original repos |
| `is:local` | repos you have cloned |
| `is:mine` | repos you own |

### Browse and partial clone

Choose **Browse files** from the action menu to open the tree. It works for any public repository, so `tab` to GitHub search first to browse someone else's code.

| Key | Action |
| --- | --- |
| `↑` `↓` `→` `←` | move, open a folder or preview a file, go up |
| `/` | filter the current folder |
| `space` | mark a file or folder |
| `c` | partial clone of what you marked |
| `s` | save the marked or highlighted files without git |
| `y` | copy the file's contents |
| `b` | switch to another branch or tag |
| `o` / `Y` / `r` | open on github.com, copy a permalink, copy the raw URL |

Saved files never overwrite existing ones; the new copy gets a `-1` suffix.

### Releases

**Releases** in the action menu lists the last 100 releases with their notes. `enter` opens the downloads with the file for your OS and CPU preselected. `space` marks several, `enter` downloads and `x` downloads and unpacks `.tar.gz`, `.tgz`, `.tar.bz2` and `.zip` archives. Downloads are checked against the release's checksum files and land in `--download-dir`, `GH_SELECT_DOWNLOAD_DIR` or the current directory.

### Deleting repositories

`ctrl+x` marks a repository you own and `ctrl+d` opens a confirmation screen for everything marked. You retype the repository's name, or `DELETE <n>` for a batch. Local clones are listed but never removed. GitHub needs an extra scope for this:

```bash
gh auth refresh -s delete_repo
```

### Clone directory and local clones

Set a clone root and gh-select clones there instead of the current directory:

```bash
export GH_SELECT_CLONE_DIR=~/dev
```

Existing clones are found anywhere under your home directory, including git worktrees and submodules, and matched by their git remote. `GH_SELECT_SCAN_DIRS` lists directories to search first, separated like `PATH`.

### Jumping into a repo

`--print-path` prints the picked repo's path on stdout, cloning it first if needed. Quitting without a pick exits with status 130, so these wrappers leave your directory alone:

```fish
function ghcd
    set -l dir (gh select --print-path); and cd $dir
end
```

```bash
ghcd() { local dir; dir=$(gh select --print-path) && cd "$dir"; }
```

### Themes

Built in: `catppuccin` (default), `dracula`, `everforest`, `graphite`, `gruvbox`, `kanagawa`, `nord`, `rose-pine`, `solarized` and `tokyonight`, each with a light and dark variant picked from the terminal background. Set one with `--theme NAME` or `GH_SELECT_THEME`, or press `ctrl+t`, which saves the choice to `~/.config/gh-select/theme`.

A custom theme is a JSON file in `~/.config/gh-select/themes/`, named after the theme. Colors left out come from the theme in `extends`, which can be built in or custom:

```json
{
  "extends": "nord",
  "dark": { "hl": "#c792ea", "pink": "#ff5370" },
  "light": { "hl": "#7c4dff" }
}
```

The colors are `bg`, `fg`, `dim`, `hl`, `cyan`, `blue`, `green`, `pink`, `yellow` and `red`, each as `#rrggbb`. `--border` adds boxes around panels, `--transparent` keeps the terminal background, and `--icons none` replaces the [Nerd Font](https://www.nerdfonts.com) icons with words.

### Commands and options

Each command works as a word, a short flag or a long flag, and takes `-h` for its own help.

```bash
gh select doctor             # check git, gh and your GitHub login (--doctor)
gh select limits             # show API rate limit usage (-l, --limits)
gh select refresh            # refetch the repo cache and exit (-r, --refresh)
gh select version            # print the version (-v, --version)
gh select help [command]     # overview, or one command's help (-h, --help)

gh select -n, --no-cache     # skip the cache and fetch fresh data
gh select -d, --dir DIR      # clone into DIR
gh select -p, --print-path   # print the picked repo's path on stdout
```

`gh select help` lists every option and environment variable, including `GH_SELECT_CACHE_TTL` (seconds, default 1800). Unknown commands exit with status 2 and suggest the closest match.

## Install

Needs [GitHub CLI](https://cli.github.com/) logged in with `gh auth login`, and `git`.

### gh extension

```bash
gh extension install remcostoeten/gh-select
gh extension upgrade gh-select
```

### Standalone binary

Each release ships a binary per platform, named like `linux-amd64` or `darwin-arm64`:

```bash
curl -L -o gh-select https://github.com/remcostoeten/gh-select/releases/latest/download/linux-amd64
chmod +x gh-select && mv gh-select ~/.local/bin/
```

### Go

```bash
go install github.com/remcostoeten/gh-select@latest
```

## Development

Requires Go 1.26 or newer.

```bash
scripts/dev.sh check     # vet, test and build
scripts/dev.sh run       # build and run ./gh-select
scripts/dev.sh install   # install the local build as gh select
scripts/dev.sh demo      # re-record the demo gif with vhs
```

<br/>

xxx,<br/>
[Remco Stoeten](https://remcostoeten.com)<br/>
<small>MIT</small>
