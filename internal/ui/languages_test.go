package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestLangListFitsWidth(t *testing.T) {
	langs := []string{"Go", "TypeScript", "Shell", "Dockerfile", "Makefile"}
	for _, w := range []int{4, 12, 20, 30, 80} {
		got := langList(langs, w)
		if lipgloss.Width(got) > w {
			t.Errorf("width %d: %q is %d cells", w, ansi.Strip(got), lipgloss.Width(got))
		}
	}
	if got := ansi.Strip(langList(langs, 80)); got != "● Go  ● TypeScript  ● Shell  ● Dockerfile  ● Makefile" {
		t.Errorf("wide: got %q", got)
	}
	if got := ansi.Strip(langList(langs, 30)); got != "● Go  ● TypeScript  +3" {
		t.Errorf("narrow: got %q", got)
	}
}
