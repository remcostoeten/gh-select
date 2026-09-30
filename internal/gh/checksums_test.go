package gh

import "testing"

func TestParseChecksums(t *testing.T) {
	gnu := ParseChecksums("checksums.txt", []byte(
		"# comment\n"+
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  tool_linux.tar.gz\n"+
			"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB *dist/tool.zip\n"+
			"not a digest line\n"))
	if gnu["tool_linux.tar.gz"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("gnu line: %v", gnu)
	}
	if gnu["tool.zip"] != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Errorf("binary-mode line: %v", gnu)
	}
	if len(gnu) != 2 {
		t.Errorf("parsed %d entries, want 2", len(gnu))
	}

	bsd := ParseChecksums("SHA256SUMS", []byte("SHA256 (tool.tar.gz) = cccccccccccccccccccccccccccccccc\n"))
	if bsd["tool.tar.gz"] != "cccccccccccccccccccccccccccccccc" {
		t.Errorf("bsd line: %v", bsd)
	}

	bare := ParseChecksums("tool.tar.gz.sha256", []byte("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n"))
	if bare["tool.tar.gz"] == "" {
		t.Errorf("bare digest: %v", bare)
	}
}

func TestIsChecksumFile(t *testing.T) {
	for name, want := range map[string]bool{
		"checksums.txt": true, "gh_2.0_checksums.txt": true, "SHA256SUMS": true,
		"tool.tar.gz.sha256": true, "tool.tar.gz": false, "tool.sig": false,
	} {
		if got := IsChecksumFile(name); got != want {
			t.Errorf("IsChecksumFile(%q) = %v", name, got)
		}
	}
}
