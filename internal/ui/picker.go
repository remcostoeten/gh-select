package ui

import (
	"fmt"
	"strings"

	"github.com/sahilm/fuzzy"
)

// pickerModel is a lightweight, scrollable single-choice list of strings with
// always-on fuzzy search — reused for the branch chooser. It renders into the
// shared chrome like the other screens.
type pickerModel struct {
	all    []string
	query  string
	cursor int
}

func newPicker(items []string) *pickerModel {
	return &pickerModel{all: items}
}

// rows returns the items matching the current query (fuzzy-ranked), or all
// items when the query is empty.
func (p *pickerModel) rows() []string {
	if p.query == "" {
		return p.all
	}
	matches := fuzzy.Find(p.query, p.all)
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = p.all[m.Index]
	}
	return out
}

func (p *pickerModel) clampCursor() {
	n := len(p.rows())
	if p.cursor >= n {
		p.cursor = n - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// pickerOutcome reports what a keypress decided.
type pickerOutcome int

const (
	pickerContinue pickerOutcome = iota
	pickerChosen
	pickerBack
	pickerQuit
)

// handleKey advances the picker. When it returns pickerChosen, selection() holds
// the chosen value.
func (p *pickerModel) handleKey(keyStr string, runes []rune, isRunes bool) pickerOutcome {
	switch keyStr {
	case "ctrl+c":
		return pickerQuit
	case "esc":
		if p.query != "" {
			p.query = ""
			p.cursor = 0
			return pickerContinue
		}
		return pickerBack
	case "enter":
		if p.selection() != "" {
			return pickerChosen
		}
		return pickerContinue
	case "up":
		if p.cursor > 0 {
			p.cursor--
		}
		return pickerContinue
	case "down":
		if p.cursor < len(p.rows())-1 {
			p.cursor++
		}
		return pickerContinue
	case "backspace":
		if p.query != "" {
			r := []rune(p.query)
			p.query = string(r[:len(r)-1])
			p.cursor = 0
		}
		return pickerContinue
	}
	if isRunes {
		p.query += string(runes)
		p.cursor = 0
	}
	return pickerContinue
}

func (p *pickerModel) selection() string {
	rows := p.rows()
	if p.cursor < 0 || p.cursor >= len(rows) {
		return ""
	}
	return rows[p.cursor]
}

// context is the header's right-hand text: a branch count. The live query lives
// in the search box rendered below the header.
func (p *pickerModel) context() string {
	if p.query == "" {
		return fmt.Sprintf("%d branches", len(p.all))
	}
	return fmt.Sprintf("%d/%d branches", len(p.rows()), len(p.all))
}

// body renders the visible window of rows for innerH body lines.
func (p *pickerModel) body(innerH int) string {
	rows := p.rows()
	if len(rows) == 0 {
		return "\n  " + dimStyle.Render("no branches match “"+p.query+"”")
	}

	visible := innerH - 1
	if visible < 1 {
		visible = 1
	}
	start := 0
	if p.cursor >= visible {
		start = p.cursor - visible + 1
	}
	end := start + visible
	if end > len(rows) {
		end = len(rows)
	}

	var b strings.Builder
	b.WriteString("\n")
	for i := start; i < end; i++ {
		pointer := "  "
		name := rows[i]
		if i == p.cursor {
			pointer = selectedStyle.Render(" ▸")
			name = selectedStyle.Render(name)
		} else {
			name = keyStyle.Render(name)
		}
		fmt.Fprintf(&b, "%s %s\n", pointer, name)
	}
	return b.String()
}

var pickerFooter = keyHint(
	[2]string{"↑↓", "move"},
	[2]string{"type", "search"},
	[2]string{"⏎", "clone"},
	[2]string{"esc", "back"},
	[2]string{"^C", "quit"},
)
