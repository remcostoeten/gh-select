package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Slim persistent chrome: a one-line header (app name + screen context), a
// blank breathing row, a one-line status row, and a one-line key-hint footer.
// All framing comes from titled panels, giving the multi-pane dashboard feel
// of tools like lazygit — the focused panel gets a highlight border, the rest
// stay dim. The whole layout is centered and capped at maxContentWidth so it
// doesn't stretch thin across very wide terminals.
const chromeLines = 4 // header (1) + blank (1) + status line (1) + footer (1)

// maxContentWidth caps how wide the layout grows; anything wider is margin.
const maxContentWidth = 118

// contentWidth is the width the layout actually occupies.
func contentWidth(total int) int {
	w := total - 4 // always keep a little side margin
	if w > maxContentWidth {
		w = maxContentWidth
	}
	if w < 20 {
		w = total
	}
	return w
}

// contentPad is the left margin that centers the content column.
func contentPad(total int) int {
	pad := (total - contentWidth(total)) / 2
	if pad < 0 {
		pad = 0
	}
	return pad
}

// splitWidths divides the content column into the main panel and the side
// column (details / selection), leaving a one-cell gap between them. side is 0
// when the terminal is too narrow for a second column.
func splitWidths(total int) (main, side int) {
	side = sideWidth(total)
	main = total
	if side > 0 {
		main = total - side - 1
	}
	return main, side
}

// searchBoxLines is the height of the search panel (border + input + border).
const searchBoxLines = 3

// chromeBorder is the rune set every panel is drawn with; selectable via
// SetBorder.
var chromeBorder = lipgloss.RoundedBorder()

// sideWidth is the width of the right-hand companion column (details,
// selection); 0 when the terminal is too narrow for a second column.
func sideWidth(total int) int {
	if total < 80 {
		return 0
	}
	w := total * 2 / 5
	if w > 56 {
		w = 56
	}
	return w
}

// panel draws content inside a full border with the title embedded in the top
// edge — the building block of the layout. width and height are outer sizes;
// content lines are clipped (never wrapped) and padded so the box is always
// exact, which keeps side-by-side panels aligned.
func panel(title, content string, width, height int, focused bool) string {
	if width < 6 {
		width = 6
	}
	if height < 2 {
		height = 2
	}
	edge := dimStyle
	label := dimStyle
	if focused {
		edge = lipgloss.NewStyle().Foreground(colHL)
		label = lipgloss.NewStyle().Foreground(colHL).Bold(true)
	}
	inner := width - 4 // 2 border cells + 2 padding cells
	b := chromeBorder

	var out strings.Builder

	if title == "" {
		out.WriteString(edge.Render(b.TopLeft + strings.Repeat(b.Top, width-2) + b.TopRight))
	} else {
		t := truncate(title, inner-2)
		rest := width - lipgloss.Width(t) - 5 // TL + top rune + 2 spaces + TR
		if rest < 0 {
			rest = 0
		}
		out.WriteString(edge.Render(b.TopLeft+b.Top) + " " + label.Render(t) + " " +
			edge.Render(strings.Repeat(b.Top, rest)+b.TopRight))
	}
	out.WriteString("\n")

	left, right := edge.Render(b.Left), edge.Render(b.Right)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(lines) {
			line = truncate(lines[i], inner)
		}
		pad := inner - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		out.WriteString(left + " " + line + strings.Repeat(" ", pad) + " " + right + "\n")
	}

	out.WriteString(edge.Render(b.BottomLeft + strings.Repeat(b.Bottom, width-2) + b.BottomRight))
	return out.String()
}

// hsplit places two same-height panels side by side with a one-cell gap.
func hsplit(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

// indent prefixes every line with n spaces — used to center the content column.
func indent(s string, n int) string {
	if n < 1 {
		return s
	}
	pad := strings.Repeat(" ", n)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

// searchField builds the search input's content line: a prompt, then either
// the typed query (with a block cursor) or a dim placeholder when empty.
func searchField(query, placeholder string) string {
	prompt := keyStyle.Render("> ")
	cursor := selectedStyle.Render("▏")
	if query == "" {
		return prompt + dimStyle.Render(placeholder) + cursor
	}
	return prompt + lipgloss.NewStyle().Foreground(colFg).Render(query) + cursor
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

// compose stacks the slim header, a breathing row, a body padded to exactly
// innerHeight rows, the status line, and the key-hint footer — the footer is
// always pinned to the bottom of the screen with status visible above it, and
// the whole column is centered within the terminal.
func compose(width, height int, version, context, body, status, keys string) string {
	cw := contentWidth(width)
	innerH := height - chromeLines
	if innerH < 1 {
		innerH = 1
	}
	statusLine := ""
	if status != "" {
		statusLine = truncate(" "+status, cw)
	}
	out := headerLine(version, context, cw) + "\n\n" +
		fitHeight(body, innerH) + "\n" +
		statusLine + "\n" +
		" " + fitHints(keys, cw-1)
	return indent(out, contentPad(width))
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

const hintSep = " · "

// keyHint renders a "key action" pair, then joins pairs with a dim separator —
// used to build the footer strings.
func keyHint(pairs ...[2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p[0])+" "+dimStyle.Render(p[1]))
	}
	return strings.Join(parts, dimStyle.Render(hintSep))
}

func dangerHint(key, label string) string {
	return errStyle.Render(key) + " " + errStyle.UnsetBold().Render(label)
}

func fitHints(keys string, w int) string {
	if lipgloss.Width(keys) <= w {
		return keys
	}
	sep := dimStyle.Render(hintSep)
	parts := strings.Split(keys, sep)
	for len(parts) > 1 && lipgloss.Width(strings.Join(parts, sep)) > w {
		parts = append(parts[:len(parts)-2], parts[len(parts)-1])
	}
	return truncate(strings.Join(parts, sep), w)
}

func helpBody(title string, rows [][2]string) string {
	keyW := 0
	for _, r := range rows {
		keyW = max(keyW, lipgloss.Width(r[0]))
	}
	var b strings.Builder
	b.WriteString("\n" + headerStyle.Render("  "+title) + "\n\n")
	for _, r := range rows {
		if r[0] == "" {
			b.WriteString("\n")
			continue
		}
		pad := strings.Repeat(" ", keyW-lipgloss.Width(r[0])+3)
		b.WriteString("   " + keyStyle.Render(r[0]) + pad + dimStyle.Render(r[1]) + "\n")
	}
	return b.String()
}
