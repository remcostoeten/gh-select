package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pressCtrlC(t *testing.T, a *App) (*App, bool) {
	t.Helper()
	m, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		return m.(*App), false
	}
	_, quit := cmd().(tea.QuitMsg)
	return m.(*App), quit
}

func TestCtrlCClearsListSearchBeforeQuitting(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a = send(t, a, key("beta"))

	a, quit := pressCtrlC(t, a)
	if quit || a.query != "" || len(a.list.Items()) != 2 {
		t.Fatalf("first ctrl+c: quit=%v query=%q items=%d", quit, a.query, len(a.list.Items()))
	}
	if _, quit = pressCtrlC(t, a); !quit {
		t.Fatal("second ctrl+c did not quit")
	}
}

func TestCtrlCClearsPickerQueryBeforeQuitting(t *testing.T) {
	p := newPicker([]string{"main", "dev"})
	p.handleKey("d", []rune("d"), true)
	if got := p.handleKey("ctrl+c", nil, false); got != pickerContinue || p.query != "" {
		t.Fatalf("first ctrl+c: outcome=%v query=%q", got, p.query)
	}
	if got := p.handleKey("ctrl+c", nil, false); got != pickerQuit {
		t.Fatalf("second ctrl+c: outcome=%v, want quit", got)
	}
}

func TestCtrlCClearsTreeFilterBeforeQuitting(t *testing.T) {
	a := openTree(t)
	a = send(t, a, key("/"))
	a = send(t, a, key("doc"))

	a, quit := pressCtrlC(t, a)
	if quit || a.tree.filter != "" || !a.tree.filtering {
		t.Fatalf("first ctrl+c: quit=%v filter=%q filtering=%v", quit, a.tree.filter, a.tree.filtering)
	}
	if _, quit = pressCtrlC(t, a); !quit {
		t.Fatal("second ctrl+c did not quit")
	}
}

func TestCtrlCClearsReleasesFilterBeforeQuitting(t *testing.T) {
	a := openReleases(t)
	a = send(t, a, key("/"))
	a = send(t, a, key("rc1"))

	a, quit := pressCtrlC(t, a)
	if quit || a.releases.filter != "" {
		t.Fatalf("first ctrl+c: quit=%v filter=%q", quit, a.releases.filter)
	}
	if _, quit = pressCtrlC(t, a); !quit {
		t.Fatal("second ctrl+c did not quit")
	}
}

func TestCtrlCClearsDeleteConfirmBeforeQuitting(t *testing.T) {
	a := newListApp(t, ownedRepos)
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlX})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlD})
	a = send(t, a, key("delete"))

	a, quit := pressCtrlC(t, a)
	if quit || a.confirm.typed != "" {
		t.Fatalf("first ctrl+c: quit=%v typed=%q", quit, a.confirm.typed)
	}
	if _, quit = pressCtrlC(t, a); !quit {
		t.Fatal("second ctrl+c did not quit")
	}
}
