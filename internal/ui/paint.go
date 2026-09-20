package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var transparentBG bool

// SetTransparent disables the painted app background, letting the terminal's
// own background (including transparency effects) show through.
func SetTransparent(v bool) { transparentBG = v }

// paintBackground fills every cell of the frame with the theme's background
// color: each line is padded to the full terminal width and the background
// sequence is re-armed after every ANSI reset, so styled and plain text alike
// sit on the app's own canvas. Terminals apply transparency only to cells on
// the default background, which is why apps that paint theirs read as a solid
// application window instead of text in a terminal.
func paintBackground(view string, width int) string {
	if transparentBG {
		return view
	}
	hex := colBg.Dark
	if !lipgloss.HasDarkBackground() {
		hex = colBg.Light
	}
	seq := lipgloss.ColorProfile().Color(hex).Sequence(true)
	if seq == "" {
		return view // NO_COLOR or a colorless terminal
	}
	bg := "\x1b[" + seq + "m"
	const reset = "\x1b[0m"

	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if pad := width - lipgloss.Width(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		lines[i] = bg + strings.ReplaceAll(line, reset, reset+bg) + reset
	}
	return strings.Join(lines, "\n")
}
