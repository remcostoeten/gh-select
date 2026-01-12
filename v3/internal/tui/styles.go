package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors (Tokyo Night Theme)
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#7aa2f7", Dark: "#7aa2f7"}
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#bb9af7", Dark: "#bb9af7"}
	ColorAccent    = lipgloss.AdaptiveColor{Light: "#ff007c", Dark: "#ff007c"}
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "#9ece6a", Dark: "#9ece6a"}
	ColorBg        = lipgloss.AdaptiveColor{Light: "#1a1b26", Dark: "#1a1b26"}
	ColorFg        = lipgloss.AdaptiveColor{Light: "#c0caf5", Dark: "#c0caf5"}
	ColorDim       = lipgloss.AdaptiveColor{Light: "#565f89", Dark: "#565f89"}

	// Styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Background(ColorPrimary).
			Foreground(ColorBg)

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			MarginLeft(1).
			Bold(true)

	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorDim).
			MarginLeft(1)

	StyleSelectedItem = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorAccent).
				PaddingLeft(1).
				Foreground(ColorPrimary)

	StyleDimText = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleTag = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(1).
			Bold(true).
			Foreground(ColorBg)

	StyleTagPrivate = StyleTag.Copy().Background(ColorDim)
	StyleTagPublic  = StyleTag.Copy().Background(ColorSuccess)
)
