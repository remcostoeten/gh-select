package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/remcostoeten/gh-select/internal/gh"
)

const (
	DirName      = "gh-select"
	FileName     = "repos.json"
	CacheTTL     = 30 * time.Minute
)

func GetCachePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, DirName, FileName), nil
}

func Save(repos []gh.Repository) error {
	path, err := GetCachePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(repos)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Load() ([]gh.Repository, error) {
	path, err := GetCachePath()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > CacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var repos []gh.Repository
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}
