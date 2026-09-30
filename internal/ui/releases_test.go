package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/remcostoeten/gh-select/internal/gh"
)

var sampleReleases = []gh.Release{
	{
		TagName:     "v1.2.0",
		Name:        "Faster",
		Body:        "## Changes\n\n- quicker",
		PublishedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Assets: []gh.Asset{
			{Name: "tool_linux_amd64.tar.gz", Size: 3 << 20, URL: "https://api.github.com/a/1"},
			{Name: "checksums.txt", Size: 512, URL: "https://api.github.com/a/2"},
		},
		TarballURL: "https://api.github.com/t",
		ZipballURL: "https://api.github.com/z",
	},
	{TagName: "v1.1.0-rc1", Prerelease: true, PublishedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
}

func openReleases(t *testing.T) *App {
	t.Helper()
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, tea.WindowSizeMsg{Width: 120, Height: 30})
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionReleases))
	if a.screen != screenReleases {
		t.Fatalf("screen = %v, want releases", a.screen)
	}
	mustRender(t, a, "releases-loading")
	return send(t, a, releasesLoadedMsg{releases: sampleReleases})
}

func TestReleasesDownloadMarked(t *testing.T) {
	a := openReleases(t)
	if v := a.View(); !strings.Contains(v, "v1.2.0") || !strings.Contains(v, "pre-release") {
		t.Fatalf("releases view missing tags/badges:\n%s", v)
	}

	a = send(t, a, key("enter"))
	mustRender(t, a, "assets")
	if got := len(a.releases.assets); got != 4 {
		t.Fatalf("downloads = %d, want 2 assets + 2 source archives", got)
	}

	a = send(t, a, key("space"))
	a = send(t, a, key("space"))
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionDownload || a.Result.Release != "v1.2.0" {
		t.Fatalf("result = %+v", a.Result)
	}
	var names []string
	for _, as := range a.Result.Assets {
		names = append(names, as.Name)
	}
	if strings.Join(names, ",") != "tool_linux_amd64.tar.gz,checksums.txt" {
		t.Errorf("downloaded %v", names)
	}
}

func TestReleasesDownloadHighlightedWhenNothingMarked(t *testing.T) {
	a := openReleases(t)
	a = send(t, a, key("enter"))
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, key("enter"))
	if len(a.Result.Assets) != 1 || a.Result.Assets[0].Name != "alpha-1.2.0.tar.gz" || !a.Result.Assets[0].Source {
		t.Errorf("assets = %+v, want the tarball", a.Result.Assets)
	}
}

func TestReleasesNotesAndPreview(t *testing.T) {
	a := openReleases(t)
	a = send(t, a, key("n"))
	mustRender(t, a, "notes")
	if !strings.Contains(a.View(), "quicker") {
		t.Error("notes reader missing release body")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.releases.level != levelReleases {
		t.Fatalf("esc from notes went to level %v", a.releases.level)
	}

	a = send(t, a, key("enter"))
	a = send(t, a, tea.KeyMsg{Type: tea.KeyDown})
	a = send(t, a, key("v"))
	mustRender(t, a, "preview-loading")
	a = send(t, a, assetPreviewMsg{name: "checksums.txt", content: []byte("abc123  tool.tar.gz\n")})
	if !strings.Contains(a.View(), "abc123") {
		t.Error("asset preview missing contents")
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.screen != screenActions {
		t.Errorf("screen = %v, want actions", a.screen)
	}
}

func TestReleasesEmptyAndError(t *testing.T) {
	a := NewApp(nil, sampleRepos, false, nil, "test")
	a = send(t, a, key("enter"))
	a = send(t, a, actionKey(t, a, ActionReleases))
	a = send(t, a, releasesLoadedMsg{})
	if !strings.Contains(a.View(), "no releases") {
		t.Error("empty state missing")
	}
	a = send(t, a, key("enter"))
	if a.Result.Action != ActionNone {
		t.Errorf("enter with no releases produced %v", a.Result.Action)
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1536: "1.5 KB", 3 << 20: "3.0 MB"} {
		if got := HumanSize(n); got != want {
			t.Errorf("HumanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestReleasesFilterSurvivesDownloadsRoundTrip(t *testing.T) {
	a := openReleases(t)
	a = send(t, a, key("/"))
	for _, r := range "rc1" {
		a = send(t, a, key(string(r)))
	}
	a = send(t, a, key("enter"))
	if rel, _ := a.releases.current(); rel.TagName != "v1.1.0-rc1" {
		t.Fatalf("filtered selection = %q", rel.TagName)
	}
	a = send(t, a, key("enter"))
	if a.releases.level != levelAssets || a.releases.filter != "" {
		t.Fatalf("level %v, filter %q", a.releases.level, a.releases.filter)
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyLeft})
	if rel, _ := a.releases.current(); rel.TagName != "v1.1.0-rc1" || a.releases.filter != "rc1" {
		t.Errorf("back from downloads: %q with filter %q", rel.TagName, a.releases.filter)
	}
}

func TestReleasesFilterAssetsAndExtract(t *testing.T) {
	a := openReleases(t)
	a = send(t, a, key("enter"))
	a = send(t, a, key("/"))
	for _, r := range "zip" {
		a = send(t, a, key(string(r)))
	}
	a = send(t, a, key("enter"))
	mustRender(t, a, "assets-filtered")
	a = send(t, a, key("x"))
	if a.Result.Action != ActionDownload || !a.Result.Extract {
		t.Fatalf("result = %+v", a.Result)
	}
	if len(a.Result.Assets) != 1 || a.Result.Assets[0].Name != "alpha-1.2.0.zip" {
		t.Errorf("assets = %+v", a.Result.Assets)
	}
	if len(a.Result.Checksums) != 1 || a.Result.Checksums[0].Name != "checksums.txt" {
		t.Errorf("checksums = %+v", a.Result.Checksums)
	}
}
