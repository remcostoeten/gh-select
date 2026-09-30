// Package download saves release assets to disk: streamed with progress,
// verified against the release's checksum files, and optionally unpacked.
package download

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/remcostoeten/gh-select/internal/gh"
	"golang.org/x/term"
)

// checksumLimit caps how much of a checksum file is read; real ones are a few
// kilobytes, so anything past this isn't a checksum list.
const checksumLimit = 1 << 20

// Options configures a batch of downloads.
type Options struct {
	Dir       string     // where files land; "" means the current directory
	Extract   bool       // unpack .tar.gz / .tgz / .tar.bz2 / .zip after downloading
	Checksums []gh.Asset // the release's checksum files
}

// Assets downloads every asset, reporting each as it goes. One failure never
// abandons the rest of the batch.
func Assets(client *gh.Client, assets []gh.Asset, opts Options) error {
	if opts.Dir != "" {
		if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
			return err
		}
	}
	digests := loadDigests(client, opts.Checksums)

	stop := cleanupOnInterrupt()
	defer stop()

	var failed []string
	for _, a := range assets {
		if err := one(client, a, digests, opts); err != nil {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
			failed = append(failed, a.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d downloads failed: %s",
			len(failed), len(assets), strings.Join(failed, ", "))
	}
	return nil
}

// loadDigests fetches and merges every checksum file. A checksum file that
// can't be read only costs verification, never the download itself.
func loadDigests(client *gh.Client, files []gh.Asset) map[string]string {
	digests := map[string]string{}
	for _, f := range files {
		body, _, err := client.OpenAsset(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: couldn't read %s, skipping it for verification: %v\n", f.Name, err)
			continue
		}
		content, err := io.ReadAll(io.LimitReader(body, checksumLimit))
		body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: couldn't read %s, skipping it for verification: %v\n", f.Name, err)
			continue
		}
		for name, sum := range gh.ParseChecksums(f.Name, content) {
			digests[name] = sum
		}
	}
	return digests
}

var (
	partMu      sync.Mutex
	currentPart string
)

func setPart(path string) {
	partMu.Lock()
	currentPart = path
	partMu.Unlock()
}

// cleanupOnInterrupt removes the in-progress .part file when the user presses
// Ctrl+C, so an abandoned download leaves nothing behind.
func cleanupOnInterrupt() func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-ch:
			partMu.Lock()
			if currentPart != "" {
				_ = os.Remove(currentPart)
			}
			partMu.Unlock()
			fmt.Fprintln(os.Stderr, "\r\033[KDownload interrupted.")
			os.Exit(130)
		case <-done:
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}

func hasherFor(digest string) hash.Hash {
	switch len(digest) {
	case 32:
		return md5.New()
	case 40:
		return sha1.New()
	case 64:
		return sha256.New()
	case 128:
		return sha512.New()
	}
	return nil
}

// one streams an asset to a .part file, verifies it, and only then gives it
// its real name, so a failed or corrupt download never looks finished.
func one(client *gh.Client, a gh.Asset, digests map[string]string, opts Options) error {
	target := filepath.Join(opts.Dir, filepath.Base(a.Name))
	if exists(target) {
		return fmt.Errorf("%s already exists", target)
	}
	body, size, err := client.OpenAsset(a)
	if err != nil {
		return err
	}
	defer body.Close()

	part := target + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	setPart(part)
	defer setPart("")

	want := digests[a.Name]
	h := hasherFor(want)
	var sink io.Writer = f
	if h != nil {
		sink = io.MultiWriter(f, h)
	}
	p := &progress{name: a.Name, total: size, lastPct: -1, live: term.IsTerminal(int(os.Stderr.Fd()))}
	n, err := io.Copy(sink, io.TeeReader(body, p))
	p.finish()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(part)
		return fmt.Errorf("%s: %w", a.Name, err)
	}

	verdict := ""
	switch {
	case h != nil:
		if got := hex.EncodeToString(h.Sum(nil)); got != want {
			_ = os.Remove(part)
			return fmt.Errorf("%s: checksum mismatch (want %s, got %s) — file discarded", a.Name, want, got)
		}
		verdict = "  ✓ checksum"
	case len(digests) > 0 && !gh.IsChecksumFile(a.Name) && !a.Source:
		verdict = "  (not listed in the checksum file — unverified)"
	}
	if err := os.Rename(part, target); err != nil {
		return err
	}
	fmt.Printf("  %s  %s%s\n", target, humanSize(n), verdict)

	if opts.Extract {
		if !IsArchive(a.Name) {
			fmt.Printf("  %s isn't an archive gh-select can unpack; kept as is\n", a.Name)
			return nil
		}
		dir, err := Extract(target, opts.Dir)
		if err != nil {
			return fmt.Errorf("extracting %s: %w", a.Name, err)
		}
		fmt.Printf("  → unpacked into %s\n", dir)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// progress redraws a single stderr line as an asset downloads.
type progress struct {
	name    string
	total   int64
	done    int64
	lastPct int64
	drawn   bool
	live    bool // stderr is a terminal that can redraw a line in place
}

func (p *progress) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if !p.live {
		return len(b), nil
	}
	if p.total > 0 {
		pct := p.done * 100 / p.total
		if pct == p.lastPct {
			return len(b), nil
		}
		p.lastPct = pct
		fmt.Fprintf(os.Stderr, "\r\033[K  %s  %3d%%  %s / %s", p.name, pct, humanSize(p.done), humanSize(p.total))
	} else {
		fmt.Fprintf(os.Stderr, "\r\033[K  %s  %s", p.name, humanSize(p.done))
	}
	p.drawn = true
	return len(b), nil
}

func (p *progress) finish() {
	if p.drawn {
		fmt.Fprint(os.Stderr, "\r\033[K")
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
