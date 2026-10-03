package config

import "testing"

func TestSavedTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GH_SELECT_THEME", "")

	if got := Load().SavedTheme; got != "" {
		t.Errorf("no saved theme: got %q", got)
	}
	if err := SaveTheme("nord"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GH_SELECT_THEME", "dracula")
	cfg := Load()
	if cfg.SavedTheme != "nord" || cfg.Theme != "dracula" {
		t.Errorf("got Theme=%q SavedTheme=%q, want dracula and nord", cfg.Theme, cfg.SavedTheme)
	}
}
