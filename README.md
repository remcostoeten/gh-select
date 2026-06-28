# gh-select

A fast `gh` CLI extension to **fuzzy-search your GitHub repositories**, **browse any repo's file tree without cloning**, and **partial (sparse) clone only the folders you want**. Built in Go with a self-contained terminal UI — no `fzf` or `jq` required.

![gh-select Demo](./assets/gh-select-demo.gif)

`gh select` turns finding, previewing, and cloning GitHub repos into a single keyboard-driven flow in your terminal. Type to filter your repositories from cache (instant), open a repo's codebase tree to preview files via the GitHub API, mark the folders you actually need, and clone just those with a sparse `git` checkout.

## Features

- **Fuzzy repository search** — always-on search box filters your repos as you type, served instantly from cache with background refresh.
- **Browse before you clone** — navigate any repository's file tree and preview file contents straight from the GitHub API, no download needed.
- **Partial / sparse clone** — select individual folders and clone only those with `git clone --filter=blob:none --sparse`, saving bandwidth and disk on large monorepos.
- **Zero extra dependencies** — the TUI and GitHub API client are built in; no `fzf`, no `jq`.
- **One precompiled binary** — install as a `gh` extension or a standalone binary; nothing to compile.
- **Quick actions** — clone, copy repo name or URL, and open in browser without leaving the terminal.

## Installation

### As a `gh` extension (recommended)

```bash
gh extension install remcostoeten/gh-select
gh extension upgrade gh-select   # later, to update
```

Precompiled binaries are published per release, so there is nothing to build.
`gh` downloads the binary matching your OS/arch and runs it as `gh select`.

### Standalone binary (without `gh`)

Every release also ships plain executables. Grab the one for your platform from
the [Releases page](https://github.com/remcostoeten/gh-select/releases), then:

```bash
chmod +x gh-select_*        # macOS/Linux
mv gh-select_* /usr/local/bin/gh-select
gh-select                   # run directly
```

It still uses `gh` for authentication under the hood, so `gh auth login` must
have been run once.

### From source (Go)

```bash
go install github.com/remcostoeten/gh-select@latest
```

Installs the latest tagged version to `$(go env GOPATH)/bin` as `gh-select`.

### Requirements

- [GitHub CLI](https://cli.github.com/) — provides authentication (`gh auth login`)
- `git` — for cloning

No `fzf` or `jq` needed; the TUI and API client are built in.

## Usage

```bash
gh select
```

1. **Find a repo** — type to fuzzy-filter your repositories. The list loads
   instantly from cache and refreshes in the background.
2. **Choose an action:**
   - Clone repository (full)
   - **Browse & partial clone** — open the codebase tree
   - Copy repository name / URL
   - Open in browser

### Browse & partial clone

Inside the tree browser you can navigate the entire repository **without
cloning it**:

- `↑/↓` move · `→` open a folder or preview a file · `←` go up
- `/` filter the current directory · `space` mark a folder (multi-select)
- `c` clone — performs a partial clone (`--filter=blob:none --sparse`) that
  downloads only the folders you selected

### Options

```bash
gh select -n, --no-cache   # bypass cache, fetch fresh data
gh select -r, --refresh    # refresh cache and exit
gh select -v, --version    # show version
gh select -h, --help       # show help
gh select doctor           # check tools + authentication
```

`GH_SELECT_CACHE_TTL` (seconds) controls cache freshness (default 1800).

## Performance

- Cached list renders instantly (stale-while-revalidate; refreshed in the
  background).
- Repository fetch uses a trimmed GraphQL query (~1.4s/page) instead of the
  expensive default-branch field.
- Codebase preview is a single tree API call; file contents load on demand.

## License

MIT
