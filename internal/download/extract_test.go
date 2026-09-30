package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	f.Close()
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	f.Close()
}

func TestExtractTarWithSingleRoot(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "tool_1.0_linux_amd64.tar.gz")
	writeTarGz(t, archive, map[string]string{"tool/bin/tool": "#!/bin/sh\n", "tool/README": "hi"})

	got, err := Extract(archive, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "tool") {
		t.Errorf("dest = %s, want the archive's own root folder", got)
	}
	info, err := os.Stat(filepath.Join(got, "bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Error("executable bit lost")
	}
}

func TestExtractZipLooseFilesGetOwnFolder(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "tool.zip")
	writeZip(t, archive, map[string]string{"tool.exe": "x", "LICENSE": "y"})

	got, err := Extract(archive, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "tool") {
		t.Errorf("dest = %s", got)
	}
	if b, _ := os.ReadFile(filepath.Join(got, "LICENSE")); string(b) != "y" {
		t.Errorf("LICENSE = %q", b)
	}
}

func TestExtractRefusesEscapes(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.tar.gz")
	writeTarGz(t, archive, map[string]string{"../escape": "x"})
	if _, err := Extract(archive, filepath.Join(dir, "out")); err == nil {
		t.Fatal("extracted an entry that climbs out of the destination")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape")); err == nil {
		t.Fatal("escape file written")
	}
}

func TestIsArchive(t *testing.T) {
	for name, want := range map[string]bool{"a.tar.gz": true, "a.TGZ": true, "a.zip": true, "a.tar.bz2": true, "a.deb": false, "a": false} {
		if IsArchive(name) != want {
			t.Errorf("IsArchive(%q) != %v", name, want)
		}
	}
}
