// Package config holds runtime configuration derived from the environment.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved runtime configuration for a single invocation.
type Config struct {
	CacheDir    string        // directory holding cached repo data
	CacheTTL    time.Duration // how long cached data is considered fresh
	CloneDir    string        // where clones land (empty = current directory)
	DownloadDir string        // where release assets land (empty = current directory)
	NoColor     bool          // disable ANSI styling
	Theme       string        // UI color theme name (empty = default)
	Border      string        // panel border style name (empty = default)
	Transparent bool          // don't paint the app background
}

// Load resolves configuration from environment variables, applying defaults
// that mirror the original shell implementation.
func Load() Config {
	return Config{
		CacheDir:    cacheDir(),
		CacheTTL:    cacheTTL(),
		CloneDir:    ExpandHome(os.Getenv("GH_SELECT_CLONE_DIR")),
		DownloadDir: ExpandHome(os.Getenv("GH_SELECT_DOWNLOAD_DIR")),
		NoColor:     os.Getenv("NO_COLOR") != "",
		Theme:       envOr("GH_SELECT_THEME", savedTheme()),
		Border:      os.Getenv("GH_SELECT_BORDER"),
		Transparent: os.Getenv("GH_SELECT_TRANSPARENT") != "",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ConfigDir resolves $XDG_CONFIG_HOME/gh-select, falling back to ~/.config.
func ConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "gh-select")
}

// ThemesDir holds custom theme files, one <name>.json per theme.
func ThemesDir() string { return filepath.Join(ConfigDir(), "themes") }

func themeStatePath() string { return filepath.Join(ConfigDir(), "theme") }

func savedTheme() string {
	data, err := os.ReadFile(themeStatePath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveTheme remembers the theme picked inside the TUI for later runs.
func SaveTheme(name string) error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(themeStatePath(), []byte(name+"\n"), 0o644)
}

// cacheDir resolves $XDG_CACHE_HOME/gh-select, falling back to ~/.cache.
func cacheDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "gh-select")
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "gh-select")
}

// ExpandHome resolves a leading "~" in a path, which shells leave untouched
// inside quoted environment variables.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// cacheTTL reads GH_SELECT_CACHE_TTL (seconds), defaulting to 30 minutes.
func cacheTTL() time.Duration {
	if v := os.Getenv("GH_SELECT_CACHE_TTL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 30 * time.Minute
}
