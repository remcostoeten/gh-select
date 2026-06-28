// Package gitx shells out to git for clone operations, including sparse
// (partial) clones that fetch only selected folders.
package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Clone performs a full clone of nameWithOwner into dir (or the default
// directory when dir is empty). When branch is non-empty only that branch is
// cloned. Output is streamed to the user's terminal.
func Clone(nameWithOwner, branch, dir string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL(nameWithOwner))
	if dir != "" {
		args = append(args, dir)
	}
	return run(args...)
}

// SparseClone clones only the selected folders and/or files using a partial
// clone (--filter=blob:none, so blobs are fetched lazily) plus sparse-checkout.
//
// With folders only, it uses cone mode (fast, the common case). When individual
// files are selected, cone mode can't express them, so it falls back to a
// non-cone pattern set anchored at the repo root. branch, when set, restricts
// the clone to that branch. dir defaults to the repo name when empty.
func SparseClone(nameWithOwner string, folders, files []string, branch, dir string) error {
	if len(folders) == 0 && len(files) == 0 {
		return fmt.Errorf("no paths selected")
	}
	if dir == "" {
		dir = repoName(nameWithOwner)
	}

	// Partial clone with no checkout yet; blobs are lazy.
	cloneArgs := []string{"clone", "--filter=blob:none", "--sparse"}
	if branch != "" {
		cloneArgs = append(cloneArgs, "--branch", branch)
	}
	cloneArgs = append(cloneArgs, repoURL(nameWithOwner), dir)
	if err := run(cloneArgs...); err != nil {
		return err
	}

	// Folders only: cone mode materializes exactly the listed directories.
	if len(files) == 0 {
		setArgs := append([]string{"-C", dir, "sparse-checkout", "set"}, folders...)
		return run(setArgs...)
	}

	// File selection: non-cone patterns. A leading "/" anchors each path to the
	// repo root; matching a directory includes its contents.
	patterns := make([]string, 0, len(folders)+len(files))
	for _, f := range folders {
		patterns = append(patterns, "/"+f)
	}
	for _, f := range files {
		patterns = append(patterns, "/"+f)
	}
	setArgs := append([]string{"-C", dir, "sparse-checkout", "set", "--no-cone"}, patterns...)
	return run(setArgs...)
}

func run(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func repoURL(nameWithOwner string) string {
	return "https://github.com/" + nameWithOwner + ".git"
}

func repoName(nameWithOwner string) string {
	if i := strings.LastIndex(nameWithOwner, "/"); i >= 0 {
		return nameWithOwner[i+1:]
	}
	return nameWithOwner
}
