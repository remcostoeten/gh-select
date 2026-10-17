package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/sys"
)

const (
	appAuthor = "Remco Stoeten"
	appSource = "github.com/remcostoeten/gh-select"
)

// buildMeta is what the header says about the running build.
type buildMeta struct {
	version string
	commit  string
	updated time.Time
	latest  string // a newer release tag, once the update check found one
	hint    string // how to install it
}

// SetBuild records the commit and commit time the binary was built from, shown
// in the header; either may be empty.
func (a *App) SetBuild(commit string, updated time.Time) {
	a.meta.commit = commit
	a.meta.updated = updated
}

// SetUpdateCheck installs the check run in the background on start. It returns
// the newer release tag and the upgrade instruction, or "" when up to date.
func (a *App) SetUpdateCheck(check func() (tag, hint string)) {
	a.updateCheck = check
}

type updateAvailableMsg struct{ tag, hint string }

func (a *App) updateCheckCmd() tea.Cmd {
	if a.updateCheck == nil {
		return nil
	}
	check := a.updateCheck
	return func() tea.Msg {
		tag, hint := check()
		return updateAvailableMsg{tag: tag, hint: hint}
	}
}

// updateNotice is the "v2.6.0 available · gh extension upgrade gh-select"
// fragment, empty while nothing newer is known.
func (m buildMeta) updateNotice() string {
	if m.latest == "" {
		return ""
	}
	return markedStyle.Render(m.latest+" available") + dimStyle.Render(hintSep+m.hint)
}

// header lays the app name, version and any update notice on the left and the
// screen context on the right; the credit row at the bottom carries the
// author, source and build. The notice drops out when w is too narrow for it.
func (m buildMeta) header(context string, w int) string {
	name := appStyle.Render("gh-select") + " " + dimStyle.Render(m.version)
	right := contextStyle.Render(context)
	if notice := m.updateNotice(); notice != "" {
		withNotice := name + dimStyle.Render(hintSep) + notice
		if lipgloss.Width(withNotice) <= w-lipgloss.Width(right)-2 {
			name = withNotice
		}
	}
	return justify(name, right, w)
}

type creditLink struct {
	key, label, url string
}

type creditsStatusMsg struct{ text string }

// creditLinks are the help overlay's credits, each opened in the browser with
// its key while the overlay is up.
func (m buildMeta) creditLinks() []creditLink {
	release := "https://" + appSource + "/releases"
	if strings.HasPrefix(m.version, "v") {
		release += "/tag/" + m.version
	}
	return []creditLink{
		{"g", "author on GitHub", "https://github.com/remcostoeten"},
		{"w", "website", "https://remcostoeten.nl"},
		{"s", "source code", "https://" + appSource},
		{"i", "report a bug or request a feature", "https://" + appSource + "/issues/new"},
		{"r", "release notes", release},
	}
}

// creditsBody is the help overlay's credits section: what the app is, who
// made it, the build, and the links with the key that opens each.
func (m buildMeta) creditsBody() string {
	var b strings.Builder
	b.WriteString("\n" + headerStyle.Render("  Credits") + "\n\n")
	b.WriteString("   " + appStyle.Render("gh-select") + " " + dimStyle.Render(m.version) + "\n")
	b.WriteString("   " + lipgloss.NewStyle().Foreground(colFg).Render("Find, preview and clone your GitHub repositories from the terminal.") + "\n")
	build := []string{"made by " + appAuthor, "MIT license"}
	if m.commit != "" && !strings.Contains(m.version, m.commit) {
		build = append(build, "commit "+m.commit)
	}
	if !m.updated.IsZero() {
		build = append(build, "updated "+m.updated.Local().Format("2 Jan 2006"))
	}
	b.WriteString("   " + dimStyle.Render(strings.Join(build, hintSep)) + "\n")
	if notice := m.updateNotice(); notice != "" {
		b.WriteString("   " + notice + "\n")
	}
	b.WriteString("\n")
	links := m.creditLinks()
	labelW := 0
	for _, l := range links {
		labelW = max(labelW, lipgloss.Width(l.label))
	}
	for _, l := range links {
		pad := strings.Repeat(" ", labelW-lipgloss.Width(l.label)+3)
		b.WriteString("   " + keycapStyle.Render(l.key) + "   " + dimStyle.Render(l.label) + pad + statusStyle.Render(strings.TrimPrefix(l.url, "https://")) + "\n")
	}
	b.WriteString("\n   " + dimStyle.Render("press a key to open the link in your browser") + "\n")
	return b.String()
}

// openCredit opens the credit link bound to key, if any.
func (a *App) openCredit(key string) tea.Cmd {
	for _, l := range a.meta.creditLinks() {
		if l.key != key {
			continue
		}
		url := l.url
		return func() tea.Msg {
			if err := sys.OpenURL(url); err != nil {
				return creditsStatusMsg{"Couldn't open browser: " + err.Error()}
			}
			return creditsStatusMsg{"Opened " + strings.TrimPrefix(url, "https://")}
		}
	}
	return nil
}
