# Changelog

All notable changes to gh-select are documented here.

## [Unreleased]

### Changed
- **Rewritten in Go** as a precompiled `gh` extension (was a Bash script).
  Authentication, REST + GraphQL access via `go-gh`; TUI via Bubble Tea.
  `fzf` and `jq` are no longer required. The previous shell implementation is
  kept under `legacy/`.

### Added
- **Codebase tree browser** — navigate a repository's full file tree and preview
  file contents without cloning (single recursive tree API call).
- **Partial / sparse clone** — multi-select folders and clone only those via
  `git clone --filter=blob:none --sparse` + cone-mode sparse-checkout.
- Type-to-filter (`/`) within the tree browser for large repositories.
- Stale-while-revalidate cache: the list renders instantly and refreshes in the
  background, with a loading spinner during fetches.
- `gh select doctor` — checks for git, gh, and authentication.

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
