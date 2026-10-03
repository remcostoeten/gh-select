package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	appAuthor = "Remco Stoeten"
	appSource = "github.com/remcostoeten/gh-select"
)

// footerMeta is what the bottom chrome line says about the running build.
type footerMeta struct {
	version string
	commit  string
	updated time.Time
}

// SetBuild records the commit and commit time the binary was built from, shown
// in the footer; either may be empty.
func (a *App) SetBuild(commit string, updated time.Time) {
	a.meta.commit = commit
	a.meta.updated = updated
}

// line lays the version, commit and last update on the left and author and
// source on the right, dropping the author and then the source when w is too
// narrow for both sides.
func (m footerMeta) line(w int) string {
	parts := []string{m.version}
	if m.commit != "" && !strings.Contains(m.version, m.commit) {
		parts = append(parts, m.commit)
	}
	if !m.updated.IsZero() {
		parts = append(parts, "updated "+m.updated.Local().Format("2 Jan 2006"))
	}
	left := dimStyle.Render(strings.Join(parts, hintSep))
	for _, right := range []string{"by " + appAuthor + hintSep + appSource, appSource} {
		r := dimStyle.Render(right)
		if lipgloss.Width(left)+lipgloss.Width(r)+2 <= w {
			return justify(left, r, w)
		}
	}
	return truncate(left, w)
}
