package ui

import (
	"runtime"
	"strings"

	"github.com/remcostoeten/gh-select/internal/gh"
)

var osAliases = map[string][]string{
	"linux":   {"linux"},
	"darwin":  {"darwin", "macos", "mac", "osx", "apple"},
	"windows": {"windows", "win64", "win32", "win"},
	"freebsd": {"freebsd"},
}

var archAliases = map[string][]string{
	"amd64": {"amd64", "x64", "64bit"},
	"arm64": {"arm64", "aarch64"},
	"386":   {"386", "i386", "i686", "x86", "32bit"},
	"arm":   {"arm", "armv6", "armv7", "armhf"},
}

// assetTokens splits an asset name into lowercase words, normalizing the
// spellings of x86-64 first because they straddle the separators.
func assetTokens(name string) map[string]bool {
	lower := strings.ToLower(name)
	lower = strings.NewReplacer("x86_64", "amd64", "x86-64", "amd64").Replace(lower)
	tokens := map[string]bool{}
	for _, t := range strings.FieldsFunc(lower, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' '
	}) {
		tokens[t] = true
	}
	return tokens
}

func hasAny(tokens map[string]bool, words []string) bool {
	for _, w := range words {
		if tokens[w] {
			return true
		}
	}
	return false
}

// platformScore rates how well an asset fits goos/goarch: 0 when it's for
// another platform or isn't installable at all, higher for a better fit.
// Portable archives beat bare binaries, which beat OS installers.
func platformScore(a gh.Asset, goos, goarch string) int {
	if a.Source || gh.IsChecksumFile(a.Name) || gh.IsSignatureFile(a.Name) {
		return 0
	}
	tokens := assetTokens(a.Name)
	if !hasAny(tokens, osAliases[goos]) {
		return 0
	}
	score := 10
	switch {
	case hasAny(tokens, archAliases[goarch]):
		score += 10
	case goos == "darwin" && tokens["universal"]:
		score += 9
	default:
		for arch, words := range archAliases {
			if arch != goarch && hasAny(tokens, words) {
				return 0
			}
		}
	}
	lower := strings.ToLower(a.Name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"), strings.HasSuffix(lower, ".zip"):
		score += 3
	case strings.HasSuffix(lower, ".deb"), strings.HasSuffix(lower, ".rpm"), strings.HasSuffix(lower, ".msi"),
		strings.HasSuffix(lower, ".pkg"), strings.HasSuffix(lower, ".dmg"), strings.HasSuffix(lower, ".apk"):
		score++
	default:
		score += 2
	}
	return score
}

// bestForPlatform returns the index of the asset that best fits this machine,
// or -1 when none does.
func bestForPlatform(assets []gh.Asset) int {
	return bestFor(assets, runtime.GOOS, runtime.GOARCH)
}

func bestFor(assets []gh.Asset, goos, goarch string) int {
	best, bestScore := -1, 0
	for i, a := range assets {
		if s := platformScore(a, goos, goarch); s > bestScore {
			best, bestScore = i, s
		}
	}
	return best
}
