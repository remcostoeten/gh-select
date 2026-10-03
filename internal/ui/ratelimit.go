package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// rateRefresh is how often the help overlay refetches usage; the endpoint is
// free, the countdowns in between tick locally.
const rateRefresh = 10 * time.Second

type rateLimitsMsg struct {
	buckets []gh.RateBucket
	err     error
}

type rateTickMsg struct{}

type rateLimits struct {
	buckets   []gh.RateBucket
	err       error
	fetchedAt time.Time
	fetching  bool
}

func (a *App) openListHelp() (tea.Model, tea.Cmd) {
	a.listHelp = true
	return a, tea.Batch(a.fetchRateLimitsCmd(), rateTickCmd())
}

func (a *App) fetchRateLimitsCmd() tea.Cmd {
	if a.client == nil || a.rate.fetching {
		return nil
	}
	a.rate.fetching = true
	return func() tea.Msg {
		buckets, err := a.client.RateLimits()
		return rateLimitsMsg{buckets: buckets, err: err}
	}
}

func rateTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return rateTickMsg{} })
}

func (a *App) updateRate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case rateLimitsMsg:
		a.rate.fetching = false
		a.rate.err = msg.err
		if msg.err == nil {
			a.rate.buckets, a.rate.fetchedAt = msg.buckets, time.Now()
		}
	case rateTickMsg:
		if !a.listHelp {
			return nil
		}
		if time.Since(a.rate.fetchedAt) >= rateRefresh {
			return tea.Batch(a.fetchRateLimitsCmd(), rateTickCmd())
		}
		return rateTickCmd()
	}
	return nil
}

func (a *App) rateLimitsBody() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  API rate limits") + dimStyle.Render("  shared by every tool on your gh login") + "\n\n")
	switch {
	case a.rate.err != nil && len(a.rate.buckets) == 0:
		b.WriteString("   " + statusStyle.Render("Couldn't load: "+a.rate.err.Error()) + "\n")
		return b.String()
	case len(a.rate.buckets) == 0:
		b.WriteString("   " + dimStyle.Render("loading…") + "\n")
		return b.String()
	}
	now := time.Now()
	for _, r := range a.rate.buckets {
		pace := dimStyle.Render(r.Pace(now))
		if _, soon := r.Exhaustion(now); soon {
			pace = statusStyle.Render(r.Pace(now))
		}
		fmt.Fprintf(&b, "   %s %s %s  %s\n",
			keyStyle.Render(fmt.Sprintf("%-12s", r.Name)),
			r.UsageBar(16, now),
			fmt.Sprintf("%5d/%-5d %-3s", r.Used, r.Limit, gh.ShortDuration(r.Window)),
			pace)
	}
	b.WriteString("\n   " + dimStyle.Render("█ used  │ how far the window has run · also: gh select limits") + "\n")
	return b.String()
}
