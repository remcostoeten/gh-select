package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// Deleting a repository on GitHub is irreversible, so the UI only ever hands
// back an intent: the repos are listed on a dedicated screen and the user must
// retype a phrase that names what is about to happen before Result is set.
// Repos the viewer doesn't own are never offered, since the API would refuse
// them anyway and a "delete" that silently fails is worse than no delete.

// deletable reports whether delete may be offered for a repo. Ownership is the
// bar: organization repositories the viewer merely administers are excluded,
// because the repo list only knows the owner login, not the viewer's role.
func deletable(r gh.Repo) bool { return r.IsOwner }

// confirmModel is the retype-to-confirm screen guarding a delete.
type confirmModel struct {
	repos  []gh.Repo
	phrase string
	typed  string
	status string
}

// confirmPhrase is what the user must retype: a single repo is confirmed by its
// own name (the convention GitHub itself uses), a bulk delete by a phrase
// carrying the count, so the number can't go unread.
func confirmPhrase(repos []gh.Repo) string {
	if len(repos) == 1 {
		return repos[0].Name()
	}
	return fmt.Sprintf("DELETE %d", len(repos))
}

func newConfirm(repos []gh.Repo) *confirmModel {
	return &confirmModel{repos: repos, phrase: confirmPhrase(repos)}
}

// markedRepos returns the marked repos in a stable, name-sorted order.
func (a *App) markedRepos() []gh.Repo {
	out := make([]gh.Repo, 0, len(a.marked))
	for _, r := range a.marked {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NameWithOwner < out[j].NameWithOwner })
	return out
}

// toggleMark adds or removes the highlighted repo from the bulk selection.
func (a *App) toggleMark() (tea.Model, tea.Cmd) {
	it, ok := a.list.SelectedItem().(repoItem)
	if !ok {
		return a, nil
	}
	if !deletable(it.repo) {
		a.status = it.repo.NameWithOwner + " isn't yours, so it can't be marked for deletion"
		return a, nil
	}
	name := it.repo.NameWithOwner
	if _, already := a.marked[name]; already {
		delete(a.marked, name)
		a.markNote = dimStyle.Render("unmarked " + name)
	} else {
		a.marked[name] = it.repo
		a.markNote = errStyle.Render("✗ ") + name + dimStyle.Render(" marked for deletion")
	}
	if n := len(a.marked); n > 0 {
		a.markNote += dimStyle.Render(" · ") + errStyle.UnsetBold().Render(plural(n, "repo", "repos")+" queued") +
			dimStyle.Render(" · press ") + keyStyle.Render("^d") + dimStyle.Render(" to review")
	}
	a.status = ""
	return a, nil
}

// enterConfirmDelete opens the confirmation screen for repos, refusing to open
// it with nothing to delete.
func (a *App) enterConfirmDelete(repos []gh.Repo) (tea.Model, tea.Cmd) {
	if len(repos) == 0 {
		a.status = "nothing marked for deletion yet: press ctrl+x on a repo you own to mark it"
		return a, nil
	}
	a.markNote = ""
	a.confirm = newConfirm(repos)
	a.screen = screenConfirmDelete
	a.status = ""
	return a, nil
}

func (a *App) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	c := a.confirm
	switch key.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "esc":
		a.confirm = nil
		a.screen = a.confirmReturn
		return a, nil
	case "backspace":
		if c.typed != "" {
			r := []rune(c.typed)
			c.typed = string(r[:len(r)-1])
		}
		c.status = ""
		return a, nil
	case "enter":
		if c.typed != c.phrase {
			c.status = "type " + c.phrase + " exactly to confirm"
			return a, nil
		}
		a.Result = Result{Action: ActionDelete, Repo: c.repos[0], Deletes: c.repos}
		return a, tea.Quit
	}
	if key.Type == tea.KeyRunes || key.Type == tea.KeySpace {
		if key.Type == tea.KeySpace {
			c.typed += " "
		} else {
			c.typed += string(key.Runes)
		}
		c.status = ""
	}
	return a, nil
}

func (a *App) viewConfirmDelete() string {
	c := a.confirm
	h := a.height - chromeLines
	cw := contentWidth(a.width)

	var b strings.Builder
	b.WriteString("\n " + errStyle.Render(fmt.Sprintf(
		"Permanently deleting %s on GitHub. This cannot be undone.", plural(len(c.repos), "repository", "repositories"))))
	b.WriteString("\n\n")

	// Leave room for the warning, the blank rows, and the confirmation prompt.
	room := h - 9
	if room < 1 {
		room = 1
	}
	for i, r := range c.repos {
		if i == room && len(c.repos) > room {
			fmt.Fprintf(&b, "   %s\n", dimStyle.Render(fmt.Sprintf("… and %d more", len(c.repos)-room)))
			break
		}
		b.WriteString("   " + errStyle.Render("✗ ") + r.NameWithOwner)
		if r.IsPrivate {
			b.WriteString(dimStyle.Render(" · ") + privateBadge.Render("private"))
		}
		if r.StargazerCount > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" · ★%d", r.StargazerCount)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n " + dimStyle.Render("type ") + keyStyle.Render(c.phrase) +
		dimStyle.Render(" to confirm, esc to cancel") + "\n\n")
	b.WriteString(" " + searchField(c.typed, ""))

	status := ""
	if c.status != "" {
		status = errStyle.Render(c.status)
	}
	body := panel("confirm deletion", b.String(), cw, h, true)
	return compose(a.width, a.height, a.version, "delete", body, status, keyHint(
		[2]string{"type", "confirmation"},
		[2]string{"enter", "delete"},
		[2]string{"esc", "cancel"},
		[2]string{"^C", "quit"},
	))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
