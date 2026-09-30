package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

var ownedRepos = []gh.Repo{
	{NameWithOwner: "remcostoeten/alpha", OwnerLogin: "remcostoeten", IsOwner: true},
	{NameWithOwner: "remcostoeten/beta", OwnerLogin: "remcostoeten", IsOwner: true},
	{NameWithOwner: "someorg/gamma", OwnerLogin: "someorg"},
}

func newListApp(t *testing.T, repos []gh.Repo) *App {
	t.Helper()
	a := NewApp(nil, repos, false, nil, "test")
	return send(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
}

// Marking two repos and confirming with the exact phrase yields one delete
// result carrying both, in name order.
func TestBulkDeleteConfirmed(t *testing.T) {
	a := newListApp(t, ownedRepos)

	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlX}) // mark alpha
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlX}) // mark beta
	if len(a.marked) != 2 {
		t.Fatalf("marked = %d, want 2", len(a.marked))
	}

	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlD})
	if a.screen != screenConfirmDelete {
		t.Fatalf("screen = %v, want confirm", a.screen)
	}
	if a.confirm.phrase != "DELETE 2" {
		t.Fatalf("phrase = %q, want DELETE 2", a.confirm.phrase)
	}
	if v := a.View(); !strings.Contains(v, "remcostoeten/alpha") || !strings.Contains(v, "remcostoeten/beta") {
		t.Error("confirm view does not list the repos being deleted")
	}

	// A near-miss must not delete anything.
	a = send(t, a, key("DELETE 1"))
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionNone {
		t.Fatalf("wrong phrase produced action %v", a.Result.Action)
	}

	a.confirm.typed = "DELETE 2"
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionDelete {
		t.Fatalf("action = %v, want ActionDelete", a.Result.Action)
	}
	got := []string{a.Result.Deletes[0].NameWithOwner, a.Result.Deletes[1].NameWithOwner}
	if len(a.Result.Deletes) != 2 || got[0] != "remcostoeten/alpha" || got[1] != "remcostoeten/beta" {
		t.Fatalf("deletes = %v", got)
	}
}

// Esc on the confirm screen returns to where it was opened from, deleting
// nothing and keeping the marks for another look.
func TestDeleteConfirmCancel(t *testing.T) {
	a := newListApp(t, ownedRepos)
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlX})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlD})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})

	if a.screen != screenList {
		t.Fatalf("screen = %v, want list", a.screen)
	}
	if a.Result.Action != ActionNone {
		t.Fatalf("cancel produced action %v", a.Result.Action)
	}
	if len(a.marked) != 1 {
		t.Fatalf("cancel dropped marks: %d", len(a.marked))
	}
}

// ctrl+d with nothing marked explains itself rather than opening an empty
// confirmation.
func TestBulkDeleteNeedsMarks(t *testing.T) {
	a := newListApp(t, ownedRepos)
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlD})
	if a.screen != screenList {
		t.Fatalf("screen = %v, want list", a.screen)
	}
	if a.status == "" {
		t.Error("no status explaining that nothing is marked")
	}
}

// Repos the viewer doesn't own can neither be marked nor deleted from the menu,
// since the API would refuse them.
func TestDeleteOnlyForOwnedRepos(t *testing.T) {
	a := newListApp(t, ownedRepos)
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown}) // someorg/gamma
	a = send(t, a, tea.KeyMsg{Type: tea.KeyCtrlX})
	if len(a.marked) != 0 {
		t.Fatalf("marked an unowned repo: %v", a.marked)
	}

	for _, it := range a.menuFor(ownedRepos[2]) {
		if it.act == ActionDelete {
			t.Fatal("delete offered for a repo the viewer doesn't own")
		}
	}
	var found bool
	for _, it := range a.menuFor(ownedRepos[0]) {
		found = found || it.act == ActionDelete
	}
	if !found {
		t.Fatal("delete missing from an owned repo's menu")
	}
}

// Deleting a single repo from the action menu is confirmed by its own name.
func TestSingleDeleteFromMenu(t *testing.T) {
	a := newListApp(t, ownedRepos)
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionDelete))
	if a.screen != screenConfirmDelete {
		t.Fatalf("screen = %v, want confirm", a.screen)
	}
	if a.confirm.phrase != "alpha" {
		t.Fatalf("phrase = %q, want alpha", a.confirm.phrase)
	}

	a = send(t, a, key("alpha"))
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionDelete || len(a.Result.Deletes) != 1 {
		t.Fatalf("action = %v deletes = %v", a.Result.Action, a.Result.Deletes)
	}
	if a.Result.Deletes[0].NameWithOwner != "remcostoeten/alpha" {
		t.Fatalf("deleting %q", a.Result.Deletes[0].NameWithOwner)
	}
}
