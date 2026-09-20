package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// palette is the set of semantic colors a theme provides. Every color adapts
// to the terminal background (Light/Dark) so themes stay readable everywhere.
type palette struct {
	bg, fg, dim, hl, cyan, blue, green, pink, yellow, red lipgloss.AdaptiveColor
}

func ac(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

// themes maps a selectable name to its palette. Each pairs a canonical dark
// variant with the scheme's light counterpart (or a hand-tuned equivalent).
var themes = map[string]palette{
	"tokyonight": {
		bg:     ac("#e1e2e7", "#1a1b26"),
		fg:     ac("#343b58", "#c0caf5"),
		dim:    ac("#787c99", "#565f89"),
		hl:     ac("#9854f1", "#bb9af7"),
		cyan:   ac("#007197", "#7dcfff"),
		blue:   ac("#2e7de9", "#7aa2f7"),
		green:  ac("#587539", "#9ece6a"),
		pink:   ac("#d20065", "#ff007c"),
		yellow: ac("#8c6c3e", "#e0af68"),
		red:    ac("#f52a65", "#f7768e"),
	},
	"catppuccin": {
		bg:     ac("#eff1f5", "#1e1e2e"),
		fg:     ac("#4c4f69", "#cdd6f4"),
		dim:    ac("#7c7f93", "#6c7086"),
		hl:     ac("#8839ef", "#cba6f7"),
		cyan:   ac("#04a5e5", "#89dceb"),
		blue:   ac("#1e66f5", "#89b4fa"),
		green:  ac("#40a02b", "#a6e3a1"),
		pink:   ac("#ea76cb", "#f5c2e7"),
		yellow: ac("#df8e1d", "#f9e2af"),
		red:    ac("#d20f39", "#f38ba8"),
	},
	"gruvbox": {
		bg:     ac("#fbf1c7", "#282828"),
		fg:     ac("#3c3836", "#ebdbb2"),
		dim:    ac("#7c6f64", "#928374"),
		hl:     ac("#8f3f71", "#d3869b"),
		cyan:   ac("#427b58", "#8ec07c"),
		blue:   ac("#076678", "#83a598"),
		green:  ac("#79740e", "#b8bb26"),
		pink:   ac("#af3a03", "#fe8019"), // gruvbox has no pink; orange is its accent
		yellow: ac("#b57614", "#fabd2f"),
		red:    ac("#9d0006", "#fb4934"),
	},
	"dracula": {
		bg:     ac("#f8f8f2", "#282a36"),
		fg:     ac("#282a36", "#f8f8f2"),
		dim:    ac("#3e4a89", "#6272a4"),
		hl:     ac("#7c43c0", "#bd93f9"),
		cyan:   ac("#037a9b", "#8be9fd"),
		blue:   ac("#3e4a89", "#6272a4"),
		green:  ac("#2f9e44", "#50fa7b"),
		pink:   ac("#d6409f", "#ff79c6"),
		yellow: ac("#b8860b", "#f1fa8c"),
		red:    ac("#cc3333", "#ff5555"),
	},
	"nord": {
		bg:     ac("#eceff4", "#2e3440"),
		fg:     ac("#2e3440", "#d8dee9"),
		dim:    ac("#7b88a1", "#616e88"),
		hl:     ac("#8f5f80", "#b48ead"),
		cyan:   ac("#3b8ba5", "#88c0d0"),
		blue:   ac("#5e81ac", "#81a1c1"),
		green:  ac("#5e7a4c", "#a3be8c"),
		pink:   ac("#9e3a44", "#bf616a"), // nord has no pink; aurora red stands in
		yellow: ac("#92722a", "#ebcb8b"),
		red:    ac("#a1333e", "#bf616a"),
	},
}

const defaultTheme = "tokyonight"

// borders maps a selectable name to the chrome's box border style.
var borders = map[string]lipgloss.Border{
	"rounded": lipgloss.RoundedBorder(),
	"sharp":   lipgloss.NormalBorder(),
	"double":  lipgloss.DoubleBorder(),
	"thick":   lipgloss.ThickBorder(),
	"hidden":  lipgloss.HiddenBorder(), // flat look: same layout, no visible frame
}

const defaultBorder = "rounded"

// ThemeNames lists the selectable theme names, sorted.
func ThemeNames() []string { return sortedKeys(themes) }

// BorderNames lists the selectable border styles, sorted.
func BorderNames() []string { return sortedKeys(borders) }

func sortedKeys[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// SetTheme switches the active palette and rebuilds every style from it.
// Call before constructing the App; an empty name keeps the default.
func SetTheme(name string) error {
	if name == "" {
		name = defaultTheme
	}
	p, ok := themes[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown theme %q (valid: %s)", name, strings.Join(ThemeNames(), ", "))
	}
	applyPalette(p)
	return nil
}

// SetBorder switches the chrome's box border style. Call before constructing
// the App; an empty name keeps the default.
func SetBorder(name string) error {
	if name == "" {
		name = defaultBorder
	}
	b, ok := borders[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown border %q (valid: %s)", name, strings.Join(BorderNames(), ", "))
	}
	chromeBorder = b
	return nil
}

// The active colors and styles. Everything below is (re)assigned by
// applyPalette so a theme switch takes effect across the whole UI.
var (
	colBg, colFg, colDim, colHL, colCyan, colBlue lipgloss.AdaptiveColor
	colGreen, colPink, colYellow, colRed          lipgloss.AdaptiveColor

	titleStyle lipgloss.Style

	// App identity and screen context shown in the header chrome.
	appStyle     lipgloss.Style
	contextStyle lipgloss.Style

	dimStyle lipgloss.Style

	privateBadge lipgloss.Style
	publicBadge  lipgloss.Style

	selectedStyle lipgloss.Style

	headerStyle lipgloss.Style

	statusStyle lipgloss.Style

	errStyle lipgloss.Style

	markedStyle lipgloss.Style

	keyStyle lipgloss.Style
)

func applyPalette(p palette) {
	colBg = p.bg
	colFg, colDim, colHL = p.fg, p.dim, p.hl
	colCyan, colBlue, colGreen = p.cyan, p.blue, p.green
	colPink, colYellow, colRed = p.pink, p.yellow, p.red

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colCyan)
	appStyle = lipgloss.NewStyle().Bold(true).Foreground(colPink)
	contextStyle = lipgloss.NewStyle().Foreground(colCyan)
	dimStyle = lipgloss.NewStyle().Foreground(colDim)
	privateBadge = lipgloss.NewStyle().Foreground(colYellow)
	publicBadge = lipgloss.NewStyle().Foreground(colGreen)
	selectedStyle = lipgloss.NewStyle().Foreground(colPink).Bold(true)
	headerStyle = lipgloss.NewStyle().Foreground(colGreen).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(colBlue)
	errStyle = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	markedStyle = lipgloss.NewStyle().Foreground(colGreen)
	keyStyle = lipgloss.NewStyle().Foreground(colCyan).Bold(true)
}

func init() {
	applyPalette(themes[defaultTheme])
}
