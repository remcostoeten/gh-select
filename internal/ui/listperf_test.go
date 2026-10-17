package ui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

func manyRepos(n int) []gh.Repo {
	langs := []string{"Go", "TypeScript", "Rust", "Shell", ""}
	repos := make([]gh.Repo, n)
	for i := range repos {
		repos[i] = gh.Repo{
			NameWithOwner:  fmt.Sprintf("me/repo-%03d", i),
			Description:    "a repository",
			Language:       langs[i%len(langs)],
			StargazerCount: i * 7,
			PushedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
			IsOwner:        true,
			IsFork:         i%9 == 0,
			IsArchived:     i%13 == 0,
			IsPrivate:      i%4 == 0,
		}
	}
	return repos
}

func BenchmarkListScroll(b *testing.B) {
	a := NewApp(nil, manyRepos(300), false, nil, "test")
	m, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = m.(*App)
	down := tea.KeyMsg{Type: tea.KeyDown}
	b.ResetTimer()
	for b.Loop() {
		m, _ = a.Update(down)
		a = m.(*App)
		_ = a.View()
	}
}

func TestColumnsMemoFollowsItems(t *testing.T) {
	d := compactDelegate{bare: true, memo: &columnsMemo{}}
	short := repoItems([]gh.Repo{{NameWithOwner: "me/a"}})
	long := repoItems([]gh.Repo{{NameWithOwner: "me/a-much-longer-name"}})
	if got := d.cachedColumns(short).name; got != 1 {
		t.Fatalf("short name width = %d", got)
	}
	if got := d.cachedColumns(long).name; got != len("a-much-longer-name") {
		t.Fatalf("memo kept stale widths: %d", got)
	}
	icons := nerdIcons
	t.Cleanup(func() { nerdIcons = icons })
	forks := repoItems([]gh.Repo{{NameWithOwner: "me/a", IsFork: true}})
	nerdIcons = true
	withIcons := d.cachedColumns(forks).tags
	nerdIcons = false
	if got := d.cachedColumns(forks).tags; got == withIcons {
		t.Fatalf("memo ignored the icon toggle: %d", got)
	}
}
