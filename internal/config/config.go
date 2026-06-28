// Package config holds runtime configuration derived from the environment.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config is the resolved runtime configuration for a single invocation.
type Config struct {
	CacheDir string        // directory holding cached repo data
	CacheTTL time.Duration // how long cached data is considered fresh
	NoColor  bool          // disable ANSI styling
}

// Load resolves configuration from environment variables, applying defaults
// that mirror the original shell implementation.
func Load() Config {
	return Config{
		CacheDir: cacheDir(),
		CacheTTL: cacheTTL(),
		NoColor:  os.Getenv("NO_COLOR") != "",
	}
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

// cacheTTL reads GH_SELECT_CACHE_TTL (seconds), defaulting to 30 minutes.
func cacheTTL() time.Duration {
	if v := os.Getenv("GH_SELECT_CACHE_TTL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 30 * time.Minute
}
