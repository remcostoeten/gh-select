# gh-select — Spec-driven Zig Migration & Feature Roadmap

This document is the **internal, Git-driven spec** for the migration and development of the GitHub CLI extension `gh-select`.

---

## Goal

- Migrate `gh-select` to Zig for performance, simplicity, and single-binary distribution
- First achieve **full functional parity**
- Only then implement new features
- Keep current codebase as reference (`v1.0/`)
- No feature creep during migration

---

## Migration Status

### ✅ Phase 0: Research Complete
- [x] API feasibility confirmed
- [x] Zig libraries evaluated
- [x] Architecture choices documented

### ✅ Phase 1: Zig Branch Created
- [x] `feature/zig` as development branch

### ✅ Phase 2: Zig Scaffold
- [x] `main.zig` entrypoint
- [x] Argument parsing
- [x] CLI command router
- [x] Styling and output helpers
- [x] XDG config and cache paths
- [x] Basic test setup

### ✅ Phase 3: Port v1.0 to Zig (Functional Parity)
- [x] Core types, errors, utilities
- [x] GitHub API calls (auth, pagination)
- [x] UI (split-pane, selection, styling)
- [x] Config (XDG loading)
- [x] Cache (TTL-based)
- [x] Unit tests

### 🚧 Phase 4: Version & Changelog
- [x] Bump to `2.0.0-alpha`
- [x] Cross-platform release workflow (Linux, macOS, Windows)
- [ ] Final testing across environments
- [ ] Tag & publish release

---

## Known Issues / Tech Debt

- [ ] Dynamic terminal resize (fallback 24x80 due to Zig 0.13 ioctl bug)
- [ ] Clipboard feedback toast ("Copied!" message)

---

## Feature Roadmap (Post-Migration)

### 1. Repository Metadata Editing
- [ ] Rename repository
- [ ] Edit description
- [ ] Manage topics
- [ ] Dry-run support
- [ ] Scope/permission checks

### 2. Repository Discovery & Search
- [ ] Search-first workflow
- [ ] Smart ranking (stars, last updated, name match)
- [ ] Filters (owner, language, topics)
- [ ] Sorting options
sssss [ ] Result caching

### 3. Repository Grouping via CLI
- [ ] Local groups/collections
- [ ] CRUD for groups
- [ ] Repo can belong to multiple groups
- [ ] Filter by group in UI

### 4. Clone Specific Branch
- [ ] Select repo
- [ ] Fetch branch list
- [ ] Select branch
- [ ] Clone with single branch / shallow depth

### 5. Clone Specific Files/Folders Only
- [ ] Inspect repo tree
- [ ] Multi-select files and folders
- [ ] Only extract selected paths
- [ ] Sparse checkout / partial clone support

---

## Test Strategy

### Unit Tests
- [x] Parsing (args)
- [x] Fuzzy scoring
- [x] Cache TTL

### Integration Tests
- [ ] Mocked GitHub API calls
- [ ] Mocked git commands

### E2E Scenarios
- [ ] Private repositories
- [ ] Missing scopes
- [ ] Large repositories
- [ ] Conflicting target directories

---

## Constraints

- No duplication
- Explicit choices
- Performance-first
- Long-term maintainability
- Extensible without restructuring
