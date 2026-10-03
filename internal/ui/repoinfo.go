package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var langColors = map[string]string{
	"Astro":            "#ff5a03",
	"C":                "#555555",
	"C#":               "#178600",
	"C++":              "#f34b7d",
	"CSS":              "#563d7c",
	"Clojure":          "#db5855",
	"Dart":             "#00B4AB",
	"Dockerfile":       "#384d54",
	"Elixir":           "#6e4a7e",
	"Go":               "#00ADD8",
	"HTML":             "#e34c26",
	"Haskell":          "#5e5086",
	"Java":             "#b07219",
	"JavaScript":       "#f1e05a",
	"Jupyter Notebook": "#DA5B0B",
	"Kotlin":           "#A97BFF",
	"Lua":              "#000080",
	"MDX":              "#fcb32c",
	"Makefile":         "#427819",
	"Nix":              "#7e7eff",
	"OCaml":            "#ef7a08",
	"Objective-C":      "#438eff",
	"PHP":              "#4F5D95",
	"Perl":             "#0298c3",
	"PowerShell":       "#012456",
	"Python":           "#3572A5",
	"R":                "#198CE7",
	"Ruby":             "#701516",
	"Rust":             "#dea584",
	"SCSS":             "#c6538c",
	"Scala":            "#c22d40",
	"Shell":            "#89e051",
	"Svelte":           "#ff3e00",
	"Swift":            "#F05138",
	"TypeScript":       "#3178c6",
	"Vim Script":       "#199f4b",
	"Vue":              "#41b883",
	"Zig":              "#ec915c",
}

func langLabel(lang string) string {
	if lang == "" {
		return ""
	}
	dot := dimStyle.Render("●")
	if c, ok := langColors[lang]; ok {
		dot = lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("●")
	}
	return dot + " " + dimStyle.Render(lang)
}

func starLabel(n int) string {
	if n <= 0 {
		return ""
	}
	if nerdIcons {
		return iconStar + " " + shortCount(n)
	}
	return "★" + shortCount(n)
}

func shortCount(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprint(n)
	case n < 10_000:
		return trimZero(fmt.Sprintf("%.1f", float64(n)/1000)) + "k"
	case n < 1_000_000:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return trimZero(fmt.Sprintf("%.1f", float64(n)/1_000_000)) + "M"
	}
}

func trimZero(s string) string {
	if len(s) > 2 && s[len(s)-2:] == ".0" {
		return s[:len(s)-2]
	}
	return s
}

func age(t, now time.Time) string {
	d := now.Sub(t)
	unit := func(n int, name string) string {
		if n == 1 {
			return "1 " + name + " ago"
		}
		return fmt.Sprintf("%d %ss ago", n, name)
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return unit(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return unit(int(d.Hours()), "hour")
	case d < 30*24*time.Hour:
		return unit(int(d.Hours()/24), "day")
	case d < 365*24*time.Hour:
		return unit(int(d.Hours()/24/30), "month")
	default:
		return unit(int(d.Hours()/24/365), "year")
	}
}
