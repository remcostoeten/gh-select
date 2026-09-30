package gh

import (
	"testing"
	"time"
)

func TestDedupeReposKeepsFirstAndOrder(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []Repo{
		{NameWithOwner: "acme/api", UpdatedAt: base, IsOwner: false, OwnerLogin: "acme"},
		{NameWithOwner: "me/dotfiles", UpdatedAt: base.Add(-time.Hour), IsOwner: true, OwnerLogin: "me"},
		{NameWithOwner: "acme/api", UpdatedAt: base, IsOwner: false, OwnerLogin: "acme"},
		{NameWithOwner: "other/lib", UpdatedAt: base.Add(-2 * time.Hour), OwnerLogin: "other"},
	}

	got := dedupeRepos(in)
	want := []string{"acme/api", "me/dotfiles", "other/lib"}
	if len(got) != len(want) {
		t.Fatalf("dedupeRepos returned %d repos, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].NameWithOwner != name {
			t.Errorf("repo %d = %q, want %q", i, got[i].NameWithOwner, name)
		}
	}
	if !got[1].IsOwner || got[0].IsOwner {
		t.Errorf("IsOwner not preserved: %+v", got)
	}
}

func TestDedupeReposEmpty(t *testing.T) {
	if got := dedupeRepos(nil); len(got) != 0 {
		t.Fatalf("dedupeRepos(nil) = %v, want empty", got)
	}
}
