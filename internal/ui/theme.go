package ui

import "github.com/charmbracelet/lipgloss"

// Tokyo Night palette, carried over from the original fzf color scheme.
var (
	colFg     = lipgloss.Color("#c0caf5")
	colDim    = lipgloss.Color("#565f89")
	colHL     = lipgloss.Color("#bb9af7")
	colCyan   = lipgloss.Color("#7dcfff")
	colBlue   = lipgloss.Color("#7aa2f7")
	colGreen  = lipgloss.Color("#9ece6a")
	colPink   = lipgloss.Color("#ff007c")
	colYellow = lipgloss.Color("#e0af68")
	colRed    = lipgloss.Color("#f7768e")
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colCyan)

	// App identity and screen context shown in the header chrome.
	appStyle     = lipgloss.NewStyle().Bold(true).Foreground(colPink)
	contextStyle = lipgloss.NewStyle().Foreground(colCyan)

	dimStyle = lipgloss.NewStyle().Foreground(colDim)

	privateBadge = lipgloss.NewStyle().Foreground(colYellow)
	publicBadge  = lipgloss.NewStyle().Foreground(colGreen)

	selectedStyle = lipgloss.NewStyle().Foreground(colPink).Bold(true)

	headerStyle = lipgloss.NewStyle().Foreground(colGreen).Bold(true)

	statusStyle = lipgloss.NewStyle().Foreground(colBlue)

	errStyle = lipgloss.NewStyle().Foreground(colRed).Bold(true)

	markedStyle = lipgloss.NewStyle().Foreground(colGreen)

	keyStyle = lipgloss.NewStyle().Foreground(colCyan).Bold(true)
)
