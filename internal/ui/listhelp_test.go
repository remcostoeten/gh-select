package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/remcostoeten/gh-select/internal/gh"
)

func TestListHelpScrollsOnShortTerminal(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 24})
	a = send(t, a, key("?"))
	now := time.Now()
	for _, n := range []string{"search", "core", "graphql", "code_search"} {
		a.rate.buckets = append(a.rate.buckets, gh.RateBucket{Name: n, Limit: 30, Used: 5, Reset: now.Add(time.Minute), Window: time.Minute})
	}

	if v := a.View(); strings.Contains(v, "code_search") || !strings.Contains(v, "scroll") {
		t.Fatal("expected the last bucket cut off and a scroll hint on a 24-row terminal")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyPgDown})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyPgDown})
	if !strings.Contains(a.View(), "code_search") {
		t.Fatal("scrolling down did not reveal the last bucket")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyPgUp})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyPgUp})
	if a.helpScroll != 0 {
		t.Fatalf("scroll did not clamp at the top: %d", a.helpScroll)
	}
	a = send(t, a, key("?"))
	a = send(t, a, key("?"))
	if a.helpScroll != 0 || !a.listHelp {
		t.Fatal("reopening the overlay did not reset the scroll")
	}
}

func TestFooterShowsBuildAndSource(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "v1.2.0")
	a.SetBuild("270fe07", time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC))
	a = send(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})

	v := a.View()
	for _, want := range []string{"v1.2.0", "270fe07", "updated 3 Oct 2026", "Remco Stoeten", "github.com/remcostoeten/gh-select"} {
		if !strings.Contains(v, want) {
			t.Errorf("footer is missing %q", want)
		}
	}
	if got := lipgloss.Height(v); got != 30 {
		t.Errorf("view is %d rows, want 30", got)
	}

	a = send(t, a, tea.WindowSizeMsg{Width: 60, Height: 30})
	if v := a.View(); !strings.Contains(v, "v1.2.0") || strings.Contains(v, "Remco Stoeten") {
		t.Error("narrow footer should keep the version and drop the author")
	}
}
