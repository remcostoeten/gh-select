package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Slim persistent chrome: a one-line header (app name, build and source on the
// left, screen context on the right), a blank breathing row, a one-line status
// row, a hairline, and a one-line key-hint footer.
// The body is built from titled panels, frameless by default or boxed via
// SetBorder, with the focused panel's title highlighted. The whole layout is
// centered and capped at maxContentWidth so it doesn't stretch thin across
// very wide terminals.
const chromeLines = 5 // header (1) + blank (1) + status line (1) + rule (1) + footer (1)

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
var chromeBorder lipgloss.Border

// frameless drops panel borders: a title row, the content, and a hairline
// between side-by-side panels instead of a box around each.
var frameless = true

// rowMark is a zero-width APC sequence a row starts with to ask panel to draw
// it as the highlighted row. panel strips it, so it never reaches the terminal.
const rowMark = "\x1b_sel\x1b\\"

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

// selectRow marks a content row as the highlighted one; prefix it to the row.
func selectRow(pointer string) string { return rowMark + pointer }

// tintRow pads a row to w cells on the selection tint, re-arming the tint
// after every reset so styled spans inside the row keep it.
func tintRow(row string, w int) string {
	if pad := w - lipgloss.Width(row); pad > 0 {
		row += strings.Repeat(" ", pad)
	}
	hex := colSel.Dark
	if !lipgloss.HasDarkBackground() {
		hex = colSel.Light
	}
	seq := lipgloss.ColorProfile().Color(hex).Sequence(true)
	if seq == "" {
		return row
	}
	bg := "\x1b[" + seq + "m"
	const reset = "\x1b[0m"
	return bg + strings.ReplaceAll(row, reset, reset+bg) + reset
}

// panelRow lays one content line into a row of inner cells with lead blank
// cells before it and one after, drawing the selection bar in the first lead
// cell and tinting the row for marked lines.
func panelRow(line string, inner, lead int) string {
	selected := strings.HasPrefix(line, rowMark)
	line = truncate(strings.TrimPrefix(line, rowMark), inner)
	pad := max(inner-lipgloss.Width(line), 0)
	if !selected {
		return strings.Repeat(" ", lead) + line + strings.Repeat(" ", pad) + " "
	}
	bar := lipgloss.NewStyle().Foreground(colHL).Render("▌")
	return tintRow(bar+strings.Repeat(" ", lead-1)+line, inner+lead+1)
}

// panel draws content under a title, either boxed (the title embedded in the
// top border) or frameless (a title row and a blank closing row). width and
// height are outer sizes and content gets height-2 rows of width-4 cells
// either way; lines are clipped (never wrapped) and padded so the box is
// always exact, which keeps side-by-side panels aligned.
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
	inner := width - 4
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	var out strings.Builder
	if frameless {
		out.WriteString(padRight("  "+label.Render(truncate(title, inner)), width) + "\n")
		for i := 0; i < height-2; i++ {
			line := ""
			if i < len(lines) {
				line = lines[i]
			}
			out.WriteString(panelRow(line, inner, 2) + " \n")
		}
		out.WriteString(strings.Repeat(" ", width))
		return out.String()
	}

	b := chromeBorder
	if title == "" {
		out.WriteString(edge.Render(b.TopLeft + strings.Repeat(b.Top, width-2) + b.TopRight))
	} else {
		t := truncate(title, inner-2)
		rest := max(width-lipgloss.Width(t)-5, 0) // TL + top rune + 2 spaces + TR
		out.WriteString(edge.Render(b.TopLeft+b.Top) + " " + label.Render(t) + " " +
			edge.Render(strings.Repeat(b.Top, rest)+b.TopRight))
	}
	out.WriteString("\n")

	left, right := edge.Render(b.Left), edge.Render(b.Right)
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out.WriteString(left + panelRow(line, inner, 1) + right + "\n")
	}

	out.WriteString(edge.Render(b.BottomLeft + strings.Repeat(b.Bottom, width-2) + b.BottomRight))
	return out.String()
}

// searchPanel is the one-line input at the top of a screen: boxed like any
// panel, or frameless as the field over a hairline rule.
func searchPanel(title, field string, width int, focused bool) string {
	if !frameless {
		return panel(title, field, width, searchBoxLines, focused)
	}
	rule := dimStyle.Render(strings.Repeat("─", max(width-2, 0)))
	return padRight("  "+truncate(field, width-4), width) + "\n " + rule + " \n"
}

// hsplit places two same-height panels side by side, split by a one-cell gap
// or, when frameless, a hairline.
func hsplit(left, right string) string {
	gap := " "
	if frameless {
		h := max(lipgloss.Height(left), lipgloss.Height(right))
		rows := make([]string, h)
		rows[0] = " "
		for i := 1; i < h-1; i++ {
			rows[i] = dimStyle.Render("│")
		}
		if h > 1 {
			rows[h-1] = " "
		}
		gap = strings.Join(rows, "\n")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
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
	prompt := keyStyle.Render("› ")
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

// justify places left and right at either end of w cells; when both don't
// fit they're joined by a space and left for truncate() to trim.
func justify(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

// compose stacks the slim header, a breathing row, a body padded to exactly
// innerHeight rows, the status line, a hairline and the key hints, so the
// footer is always pinned to the bottom of the screen with status visible
// above it, and the whole column is centered within the terminal.
func compose(width, height int, meta buildMeta, context, body, status, keys string) string {
	cw := contentWidth(width)
	innerH := height - chromeLines
	if innerH < 1 {
		innerH = 1
	}
	const edge = "  " // panels start their content two cells in
	statusLine := ""
	if status != "" {
		statusLine = edge + truncate(status, cw-4)
	}
	out := edge + truncate(meta.header(context, cw-4), cw-4) + "\n\n" +
		fitHeight(body, innerH) + "\n" +
		statusLine + "\n" +
		" " + dimStyle.Render(strings.Repeat("─", max(cw-2, 0))) + "\n" +
		edge + fitHints(keys, cw-4)
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
