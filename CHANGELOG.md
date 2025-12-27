# Changelog

All notable changes to gh-select are documented here.

## [Unreleased] - 2025-12-27

### Added
- **Cache system** with 30-minute TTL (12x faster startup)
- **Interactive spellcheck** - typos prompt "Did you mean X? [Y/n]"
- `--no-cache` flag to bypass cache and fetch fresh data
- Auto-generated argument system (define once, get --long, -short, bare forms)

### Fixed
- Fixed pagination bug that fetched 2000 repos instead of actual count

### Performance
- Cold start: ~3s (API fetch)
- Cached: ~0.24s (12x faster)

---

## [2.2.0] - 2025-07-18

### Changed
- Clean up repository - remove bloat and organize structure

---

## [2.1.0] - 2025-07-18

### Changed
- Clean up README - remove bloat and make it concise

---

## [2.0.0] - 2025-07-18

### Changed
- Refactor to meet GitHub CLI extension requirements

---

## [1.1.0] - 2025-07-05

### Added
- Visual specifications
- GitHub Action to auto-regenerate OG image
- Pillow script and Open Graph image

---

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
