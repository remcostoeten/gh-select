package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Persistent app chrome: a boxed header (app name + version + context) and a
// boxed footer (contextual key hints) drawn around every screen so the TUI
// always shows where you are and what you can press. Each box is pinned to a
// single content line, so the chrome always occupies a fixed number of rows —
// which lets every screen compute its remaining body height.
const chromeLines = 6 // header box (3) + footer box (3)

// box wraps a single content line in a full-width rounded border with one cell
// of horizontal padding. Content is truncated (never wrapped) so the box stays
// exactly three rows tall regardless of terminal width.
func box(content string, width int) string {
	if width < 4 {
		width = 4
	}
	inner := width - 4 // 2 border cells + 2 padding cells
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colDim).
		Padding(0, 1).
		Width(width - 2). // total width including the left/right border cells
		MaxHeight(3).
		Render(truncate(content, inner))
}

// truncate clips an ANSI-styled string to w display cells, adding an ellipsis
// when it overflows. ANSI-aware so style escapes aren't miscounted or severed.
func truncate(s string, w int) string {
	if w < 1 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// headerLine lays the app identity on the left and screen context on the right,
// justified to fill innerWidth cells.
func headerLine(version, context string, innerWidth int) string {
	left := appStyle.Render("gh-select") + " " + dimStyle.Render(version)
	right := contextStyle.Render(context)
	gap := innerWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// Too narrow for both — keep the app name; truncate() trims the rest.
		return left + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

// compose stacks the header, a body padded to exactly innerHeight rows, and the
// footer — so the footer is always pinned to the bottom of the screen.
func compose(width, height int, version, context, body, keys string) string {
	header := box(headerLine(version, context, width-4), width)
	footer := box(keys, width)
	innerH := height - chromeLines
	if innerH < 1 {
		innerH = 1
	}
	return header + "\n" + fitHeight(body, innerH) + "\n" + footer
}

// fitHeight truncates or blank-pads body to exactly n rows.
func fitHeight(body string, n int) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// keyHint renders a "key action" pair, then joins pairs with a dim separator —
// used to build the footer strings.
func keyHint(pairs ...[2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p[0])+" "+dimStyle.Render(p[1]))
	}
	return strings.Join(parts, dimStyle.Render("  ·  "))
}
