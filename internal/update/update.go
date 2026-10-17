// Package update notices when a newer gh-select release exists, asking GitHub
// at most once a day so the app never slows down a start for it.
package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// interval is how long a fetched answer is trusted before asking again.
const interval = 24 * time.Hour

const stateFile = "latest-release.json"

// state is the cached answer: the newest tag seen and when it was fetched.
type state struct {
	Tag     string    `json:"tag"`
	Checked time.Time `json:"checked"`
}

// Check returns the newest release tag when it is newer than current, else "".
// fetch runs only when the cached answer is older than a day; when it fails
// the stale answer is kept. Builds that aren't a release never check.
func Check(dir, current string, fetch func() (string, error)) string {
	if _, ok := parse(current); !ok {
		return ""
	}
	s, cached := load(dir)
	if !cached || time.Since(s.Checked) > interval {
		if tag, err := fetch(); err == nil {
			s = state{Tag: tag, Checked: time.Now()}
			_ = save(dir, s)
		}
	}
	return newerOrEmpty(current, s.Tag)
}

// Cached answers from the last check alone, without touching the network, for
// commands that must exit at once.
func Cached(dir, current string) string {
	s, ok := load(dir)
	if !ok {
		return ""
	}
	return newerOrEmpty(current, s.Tag)
}

func newerOrEmpty(current, latest string) string {
	if Newer(current, latest) {
		return latest
	}
	return ""
}

// Newer reports whether latest is a higher release than current. Both must be
// "v"-prefixed semantic versions; anything after a "-" or "+" is ignored.
func Newer(current, latest string) bool {
	cur, ok := parse(current)
	if !ok {
		return false
	}
	next, ok := parse(latest)
	if !ok {
		return false
	}
	for i := range cur {
		if next[i] != cur[i] {
			return next[i] > cur[i]
		}
	}
	return false
}

// parse reads "vMAJOR.MINOR.PATCH", dropping a prerelease or build suffix.
func parse(v string) ([3]int, bool) {
	var out [3]int
	if !strings.HasPrefix(v, "v") {
		return out, false
	}
	v = v[1:]
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Hint is the upgrade instruction for the way this binary was installed: the
// gh extension command when gh manages it, otherwise the releases page.
func Hint(executable string) string {
	path := strings.ReplaceAll(executable, `\`, "/")
	if strings.Contains(path, "/gh/extensions/") || strings.Contains(path, "/GitHub CLI/extensions/") {
		return "gh extension upgrade gh-select"
	}
	return "github.com/remcostoeten/gh-select/releases/latest"
}

func load(dir string) (state, bool) {
	data, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		return state{}, false
	}
	var s state
	if err := json.Unmarshal(data, &s); err != nil || s.Tag == "" {
		return state{}, false
	}
	return s, true
}

func save(dir string, s state) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	file := filepath.Join(dir, stateFile)
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}
