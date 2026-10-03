package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
