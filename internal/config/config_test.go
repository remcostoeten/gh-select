package config

import "testing"

func TestThemePrecedence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GH_SELECT_THEME", "")

	if got := Load().Theme; got != "" {
		t.Errorf("no saved theme: got %q", got)
	}
	if err := SaveTheme("nord"); err != nil {
		t.Fatal(err)
	}
	if got := Load().Theme; got != "nord" {
		t.Errorf("saved theme: got %q, want nord", got)
	}
	t.Setenv("GH_SELECT_THEME", "dracula")
	if got := Load().Theme; got != "dracula" {
		t.Errorf("env should win over saved theme: got %q", got)
	}
}
