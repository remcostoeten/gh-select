package gitx

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fakeGit(t *testing.T) func() []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake git is a shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	SetOutput(io.Discard)
	t.Cleanup(func() { SetOutput(os.Stdout) })
	return func() []string {
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
}

func TestSparseCloneFoldersUsesConeMode(t *testing.T) {
	calls := fakeGit(t)
	if err := SparseClone("acme/mono", []string{"apps/web", "packages/ui"}, nil, "", ""); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"clone --filter=blob:none --sparse https://github.com/acme/mono.git mono",
		"-C mono sparse-checkout set apps/web packages/ui",
	}
	assertCalls(t, calls(), want)
}

func TestSparseCloneWithFilesAnchorsNonConePatterns(t *testing.T) {
	calls := fakeGit(t)
	if err := SparseClone("acme/mono", []string{"docs"}, []string{"README.md"}, "dev", "out"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"clone --filter=blob:none --sparse --branch dev https://github.com/acme/mono.git out",
		"-C out sparse-checkout set --no-cone /docs /README.md",
	}
	assertCalls(t, calls(), want)
}

func TestSparseCloneWithoutPathsRunsNothing(t *testing.T) {
	fakeGit(t)
	if err := SparseClone("acme/mono", nil, nil, "", ""); err == nil {
		t.Fatal("expected an error for an empty selection")
	}
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("git ran %d times, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d:\n got  %q\n want %q", i, got[i], want[i])
		}
	}
}
