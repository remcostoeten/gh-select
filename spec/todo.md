# gh-select v2 - Roadmap / TODO

## ✅ Completed (This Session)
- [x] Split-pane UI (List | Preview)
- [x] Buffered I/O for 3x speedup
- [x] Quick action hotkeys (`w`/`r`/`o`)
- [x] Action menu polish (readable titles, alignment)
- [x] Unit tests (fuzzyScore, cache)
- [x] `homepageUrl` support

## 🚧 Known Issues
- [ ] Dynamic terminal resize (fallback to 24x80 due to POSIX ioctl bug in Zig 0.13)

## 🔮 Future Features
- [ ] Clipboard feedback toast ("Copied!" message with timeout)
- [ ] README preview in right pane (fetch via `gh api`)
- [ ] Star count display
- [ ] Language/topic tags
- [ ] Configurable keybindings
- [ ] `--filter` flag for scripting (non-interactive)
- [ ] Org/user filter (`--org`, `--user`)
- [ ] Themes (Tokyo Night variants, Catppuccin, etc.)
