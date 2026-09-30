package download

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var archiveExts = []string{".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".zip"}

// IsArchive reports whether Extract can unpack a file with this name.
func IsArchive(name string) bool {
	return archiveBase(name) != name
}

func archiveBase(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range archiveExts {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

// entry is one file or directory inside an archive, format-independent.
type entry struct {
	name string
	mode os.FileMode
	dir  bool
	link bool
	open func() (io.ReadCloser, error)
}

// Extract unpacks archive under parent and returns the directory it created.
// An archive whose entries all sit in one top-level folder unpacks as that
// folder; any other gets a folder named after the archive, so nothing ever
// spills loose into parent. Entries escaping the destination are refused and
// links are skipped, since an archive from the internet shouldn't be able to
// write outside the folder it was asked to fill.
func Extract(archive, parent string) (string, error) {
	entries, closer, err := readEntries(archive)
	if err != nil {
		return "", err
	}
	defer closer()

	for _, e := range entries {
		if !safePath(e.name) {
			return "", fmt.Errorf("refusing unsafe path %q", e.name)
		}
	}

	dest := filepath.Join(parent, filepath.Base(archiveBase(archive)))
	strip := ""
	if root := commonRoot(entries); root != "" {
		dest = filepath.Join(parent, root)
		strip = root + "/"
	}
	if exists(dest) {
		return "", fmt.Errorf("%s already exists", dest)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}

	skipped := 0
	place := func(e entry, src io.Reader) error {
		rel := strings.TrimPrefix(cleanName(e.name), strip)
		if rel == "" || rel == "." {
			return nil
		}
		out := filepath.Join(dest, filepath.FromSlash(rel))
		switch {
		case e.link:
			skipped++
			return nil
		case e.dir:
			return os.MkdirAll(out, 0o755)
		}
		return writeFile(out, e.mode, src)
	}
	if isZip(archive) {
		for _, e := range entries {
			if err := placeZip(e, place); err != nil {
				return dest, err
			}
		}
	} else if err := walkTar(archive, isBzip(archive), func(h *tar.Header, r io.Reader) error {
		return place(tarEntry(h), r)
	}); err != nil {
		return dest, err
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "  skipped %d link(s) in the archive\n", skipped)
	}
	return dest, nil
}

func placeZip(e entry, place func(entry, io.Reader) error) error {
	if e.dir || e.link {
		return place(e, nil)
	}
	src, err := e.open()
	if err != nil {
		return err
	}
	defer src.Close()
	return place(e, src)
}

func writeFile(out string, mode os.FileMode, src io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func cleanName(name string) string {
	return strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(name, "\\", "/")), "/")
}

// safePath rejects absolute names and any ".." that would climb out.
func safePath(name string) bool {
	n := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(n, "/") || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return false
	}
	clean := path.Clean(n)
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// commonRoot is the single top-level directory every entry lives under, or ""
// when entries are spread across the archive root.
func commonRoot(entries []entry) string {
	root := ""
	for _, e := range entries {
		name := cleanName(e.name)
		if name == "" {
			continue
		}
		first, _, nested := strings.Cut(name, "/")
		if !nested && !e.dir {
			return ""
		}
		if root == "" {
			root = first
		} else if root != first {
			return ""
		}
	}
	return root
}

func isZip(name string) bool { return strings.HasSuffix(strings.ToLower(name), ".zip") }

func isBzip(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".bz2") || strings.HasSuffix(lower, ".tbz2")
}

func readEntries(archive string) ([]entry, func(), error) {
	if isZip(archive) {
		return zipEntries(archive)
	}
	var entries []entry
	err := walkTar(archive, isBzip(archive), func(h *tar.Header, _ io.Reader) error {
		entries = append(entries, tarEntry(h))
		return nil
	})
	return entries, func() {}, err
}

func zipEntries(archive string) ([]entry, func(), error) {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return nil, nil, err
	}
	var entries []entry
	for _, f := range zr.File {
		entries = append(entries, entry{
			name: f.Name,
			mode: f.Mode(),
			dir:  f.FileInfo().IsDir(),
			link: f.Mode()&os.ModeSymlink != 0,
			open: f.Open,
		})
	}
	return entries, func() { zr.Close() }, nil
}

func tarEntry(h *tar.Header) entry {
	return entry{
		name: h.Name,
		mode: os.FileMode(h.Mode),
		dir:  h.Typeflag == tar.TypeDir,
		link: h.Typeflag == tar.TypeSymlink || h.Typeflag == tar.TypeLink,
	}
}

func walkTar(archive string, bz bool, fn func(*tar.Header, io.Reader) error) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	tr, err := tarReader(f, bz)
	if err != nil {
		return err
	}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if err := fn(h, tr); err != nil {
			return err
		}
	}
}

func tarReader(r io.Reader, bz bool) (*tar.Reader, error) {
	if bz {
		return tar.NewReader(bzip2.NewReader(r)), nil
	}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return tar.NewReader(gz), nil
}
