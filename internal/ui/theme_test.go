package ui

import "testing"

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
