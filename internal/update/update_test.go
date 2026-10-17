package update

import (
	"errors"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v2.5.0", "v2.6.0", true},
		{"v2.5.0", "v3.0.0", true},
		{"v2.5.0", "v2.5.1", true},
		{"v2.5.0", "v2.5.0", false},
		{"v2.5.0", "v2.4.9", false},
		{"v2.5.0", "v10.0.0", true},
		{"v2.5.0-rc1", "v2.5.0", false},
		{"v2.5.0", "v2.6.0-rc1", true},
		{"dev+270fe07", "v2.6.0", false},
		{"v2.5.0", "latest", false},
		{"v2.5.0", "", false},
		{"v2.5", "v2.6.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestCheckFetchesOnceADay(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	fetch := func() (string, error) {
		calls++
		return "v2.6.0", nil
	}
	if got := Check(dir, "v2.5.0", fetch); got != "v2.6.0" {
		t.Fatalf("first check = %q, want v2.6.0", got)
	}
	if got := Check(dir, "v2.5.0", fetch); got != "v2.6.0" {
		t.Fatalf("second check = %q, want v2.6.0", got)
	}
	if calls != 1 {
		t.Fatalf("fetch ran %d times, want 1", calls)
	}
	if got := Cached(dir, "v2.6.0"); got != "" {
		t.Fatalf("Cached on the latest version = %q, want empty", got)
	}
	if got := Cached(dir, "v2.5.0"); got != "v2.6.0" {
		t.Fatalf("Cached = %q, want v2.6.0", got)
	}
}

func TestCheckSkipsNonReleaseBuilds(t *testing.T) {
	fetch := func() (string, error) {
		t.Fatal("fetch must not run for a dev build")
		return "", nil
	}
	if got := Check(t.TempDir(), "dev+270fe07", fetch); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestCheckSwallowsFetchErrors(t *testing.T) {
	dir := t.TempDir()
	fetch := func() (string, error) { return "", errors.New("offline") }
	if got := Check(dir, "v2.5.0", fetch); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
	if got := Cached(dir, "v2.5.0"); got != "" {
		t.Fatalf("a failed fetch was cached: %q", got)
	}
}

func TestHint(t *testing.T) {
	cases := map[string]string{
		"/home/me/.local/share/gh/extensions/gh-select/gh-select":                 "gh extension upgrade gh-select",
		`C:\Users\me\AppData\Local\GitHub CLI\extensions\gh-select\gh-select.exe`: "gh extension upgrade gh-select",
		"/usr/local/bin/gh-select":                                                "github.com/remcostoeten/gh-select/releases/latest",
		"/home/me/go/bin/gh-select":                                               "github.com/remcostoeten/gh-select/releases/latest",
	}
	for exe, want := range cases {
		if got := Hint(exe); got != want {
			t.Errorf("Hint(%q) = %q, want %q", exe, got, want)
		}
	}
}
