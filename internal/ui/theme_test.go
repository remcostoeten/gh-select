package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestThemeAndBorderSelection(t *testing.T) {
	defer func() {
		_ = SetTheme("")
		_ = SetBorder("")
	}()

	if err := SetTheme("gruvbox"); err != nil {
		t.Fatal(err)
	}
	if colHL != ac("#8f3f71", "#d3869b") {
		t.Errorf("gruvbox highlight not applied: %+v", colHL)
	}
	if err := SetTheme("Catppuccin"); err != nil { // case-insensitive
		t.Errorf("case-insensitive theme lookup failed: %v", err)
	}
	if err := SetTheme("nope"); err == nil {
		t.Error("unknown theme accepted")
	}

	if err := SetBorder("double"); err != nil {
		t.Fatal(err)
	}
	if err := SetBorder("nope"); err == nil {
		t.Error("unknown border accepted")
	}
}

func TestCustomThemes(t *testing.T) {
	defer func() {
		delete(themes, "mine")
		_ = SetTheme("")
	}()

	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Mine.json", `{"extends": "nord", "dark": {"hl": "#ff0000"}}`)
	if err := LoadCustomThemes(dir); err != nil {
		t.Fatal(err)
	}
	if err := SetTheme("mine"); err != nil {
		t.Fatal(err)
	}
	if colHL != ac("#8f5f80", "#ff0000") {
		t.Errorf("override or inherited light color wrong: %+v", colHL)
	}
	if colRed != themes["nord"].red {
		t.Errorf("unset color not inherited from nord: %+v", colRed)
	}

	for _, body := range []string{
		`{"dark": {"hl": "red"}}`,
		`{"dark": {"purple": "#ff0000"}}`,
		`{"extends": "nope"}`,
		`not json`,
	} {
		write("bad.json", body)
		if err := LoadCustomThemes(dir); err == nil {
			t.Errorf("accepted %s", body)
		}
	}

	if err := LoadCustomThemes(filepath.Join(dir, "missing")); err != nil {
		t.Errorf("missing dir: %v", err)
	}
}

func TestCycleTheme(t *testing.T) {
	defer func() { _ = SetTheme("") }()

	names := ThemeNames()
	_ = SetTheme(names[len(names)-1])
	if got := cycleTheme(); got != names[0] || CurrentTheme() != names[0] {
		t.Errorf("cycle from last = %q, want wrap to %q", got, names[0])
	}
}
