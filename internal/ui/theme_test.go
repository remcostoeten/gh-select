package ui

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCustomThemeChainsAndBrokenFiles(t *testing.T) {
	defer func() {
		for _, n := range []string{"child", "zbase", "loop-a", "loop-b"} {
			delete(themes, n)
		}
		_ = SetTheme("")
	}()

	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("child.json", `{"extends": "zbase", "dark": {"red": "#00ff00"}}`)
	write("zbase.json", `{"extends": "nord", "dark": {"hl": "#ff0000"}}`)
	write("broken.json", `not json`)
	write("loop-a.json", `{"extends": "loop-b"}`)
	write("loop-b.json", `{"extends": "loop-a"}`)

	err := LoadCustomThemes(dir)
	if err == nil {
		t.Fatal("broken and looping files not reported")
	}
	for _, want := range []string{"broken.json", "loop-a.json", "loop-b.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
	child, ok := themes["child"]
	if !ok {
		t.Fatal("child extending a later custom theme did not load")
	}
	if child.hl.Dark != "#ff0000" || child.red.Dark != "#00ff00" || child.bg != themes["nord"].bg {
		t.Errorf("child did not inherit through zbase from nord: %+v", child)
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
