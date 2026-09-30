package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

// readmeModel is the rendered-markdown reader for a repository's readme.
type readmeModel struct {
	client *gh.Client
	repo   gh.Repo

	view    viewport.Model
	path    string
	raw     []byte
	loading bool
	err     error
}

type readmeLoadedMsg struct {
	path    string
	content []byte
	err     error
}

func newReadmeModel(client *gh.Client, repo gh.Repo, w, h int) *readmeModel {
	r := &readmeModel{client: client, repo: repo, loading: true}
	r.view = viewport.New(readmeSize(w, h))
	return r
}

// readmeSize is the viewport size inside the reader panel.
func readmeSize(w, h int) (int, int) { return contentWidth(w) - 4, h - chromeLines - 2 }

func (r *readmeModel) setSize(w, h int) {
	r.view.Width, r.view.Height = readmeSize(w, h)
	if !r.loading && r.err == nil {
		r.view.SetContent(renderMarkdown(string(r.raw), r.view.Width))
	}
}

func (r *readmeModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		path, content, err := r.client.FetchReadme(r.repo.NameWithOwner)
		return readmeLoadedMsg{path: path, content: content, err: err}
	}
}

func (r *readmeModel) loaded(msg readmeLoadedMsg) {
	r.loading = false
	if msg.err != nil {
		r.err = msg.err
		return
	}
	r.path, r.raw = msg.path, msg.content
	body := renderMarkdown(string(msg.content), r.view.Width)
	if !isMarkdown(msg.path) {
		body = string(msg.content) // a plain-text or rst readme stays verbatim
	}
	r.view.SetContent(body)
	r.view.GotoTop()
}

func readmeFooter() string {
	return keyHint(
		[2]string{"↑↓", "scroll"},
		[2]string{"esc", "back"},
		[2]string{"^C", "quit"},
	)
}

// chromeParts returns the header context, body, status line, and footer for the
// readme screen, sized to innerH body rows.
func (r *readmeModel) chromeParts(spinnerFrame string, w, innerH int) (context, body, status, keys string) {
	cw := contentWidth(w)
	title := r.repo.NameWithOwner
	switch {
	case r.err != nil:
		return title,
			panel("readme", errStyle.Render("Error: "+r.err.Error()), cw, innerH, false),
			"", readmeFooter()
	case r.loading:
		return title,
			panel("readme", "", cw, innerH, false),
			spinnerFrame + statusStyle.Render("Loading readme…"),
			readmeFooter()
	}
	pct := dimStyle.Render(fmt.Sprintf("%3.0f%%", r.view.ScrollPercent()*100))
	return title + "/" + r.path,
		panel("readme", r.view.View(), cw, innerH, true),
		pct, readmeFooter()
}
