package ui

import (
	"testing"

	"github.com/remcostoeten/gh-select/internal/gh"
)

func TestBestFor(t *testing.T) {
	assets := []gh.Asset{
		{Name: "gh_2.0_checksums.txt"},
		{Name: "gh_2.0_linux_386.tar.gz"},
		{Name: "gh_2.0_linux_amd64.deb"},
		{Name: "gh_2.0_linux_amd64.tar.gz"},
		{Name: "gh_2.0_linux_amd64.tar.gz.sig"},
		{Name: "gh_2.0_macOS_arm64.zip"},
		{Name: "gh_2.0_macOS_universal.pkg"},
		{Name: "gh_2.0_windows_amd64.msi"},
		{Name: "tool-x86_64-unknown-linux-musl.tar.gz"},
	}
	cases := []struct{ goos, goarch, want string }{
		{"linux", "amd64", "gh_2.0_linux_amd64.tar.gz"},
		{"linux", "386", "gh_2.0_linux_386.tar.gz"},
		{"darwin", "arm64", "gh_2.0_macOS_arm64.zip"},
		{"darwin", "amd64", "gh_2.0_macOS_universal.pkg"},
		{"windows", "amd64", "gh_2.0_windows_amd64.msi"},
		{"freebsd", "amd64", ""},
	}
	for _, c := range cases {
		got := ""
		if i := bestFor(assets, c.goos, c.goarch); i >= 0 {
			got = assets[i].Name
		}
		if got != c.want {
			t.Errorf("%s/%s: got %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
	if i := bestFor([]gh.Asset{{Name: "tool-x86_64-unknown-linux-musl.tar.gz"}}, "linux", "amd64"); i != 0 {
		t.Error("x86_64 spelling not recognized")
	}
}
