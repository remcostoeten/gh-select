package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// isMarkdown reports whether a path should be rendered as prose rather than
// syntax-highlighted as source.
func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".mkdn":
		return true
	}
	return false
}

// renderMarkdown renders markdown to ANSI, wrapped to width cells and colored
// from the active theme. On any failure it returns the source unchanged, so a
// preview never disappears because of a renderer error.
func renderMarkdown(source string, width int) string {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle()),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return source
	}
	out, err := r.Render(source)
	if err != nil {
		return source
	}
	return strings.Trim(out, "\n")
}

// markdownStyle adapts glamour's stock style to the active palette. Margins are
// dropped because the surrounding panel already provides the padding, and its
// lines are clipped rather than wrapped.
func markdownStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	if !lipgloss.HasDarkBackground() {
		s = styles.LightStyleConfig
	}
	p := activePalette

	zero := uint(0)
	s.Document.Margin = &zero
	s.Document.Color = hexPtr(p.fg)

	s.Heading.Color = hexPtr(p.hl)
	s.H1.Color = hexPtr(p.bg)
	s.H1.BackgroundColor = hexPtr(p.hl)
	s.H2.Color = hexPtr(p.hl)
	s.H3.Color = hexPtr(p.cyan)
	s.H4.Color, s.H5.Color, s.H6.Color = hexPtr(p.cyan), hexPtr(p.cyan), hexPtr(p.dim)

	s.Link.Color = hexPtr(p.blue)
	s.LinkText.Color = hexPtr(p.cyan)
	s.Image.Color = hexPtr(p.pink)
	s.ImageText.Color = hexPtr(p.dim)
	s.Code.Color = hexPtr(p.pink)
	s.Code.BackgroundColor = nil
	s.Item.Color = hexPtr(p.green)
	s.Enumeration.Color = hexPtr(p.green)
	s.HorizontalRule.Color = hexPtr(p.dim)
	s.BlockQuote.Color = hexPtr(p.dim)

	return s
}

// hexPtr resolves an adaptive color for the terminal's background into the hex
// string glamour's style config expects.
func hexPtr(c lipgloss.AdaptiveColor) *string {
	hex := c.Light
	if lipgloss.HasDarkBackground() {
		hex = c.Dark
	}
	return &hex
}
