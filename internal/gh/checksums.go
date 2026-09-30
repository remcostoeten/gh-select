package gh

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"path"
	"regexp"
	"strings"
)

var checksumExts = []string{".sha256", ".sha512", ".sha1", ".md5", ".sha256sum", ".sha512sum"}

// IsChecksumFile reports whether an asset carries digests of other assets:
// a combined list such as checksums.txt or SHA256SUMS, or a per-file
// companion such as tool.tar.gz.sha256.
func IsChecksumFile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range checksumExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return strings.Contains(lower, "checksum") || strings.Contains(lower, "sha256sums") ||
		strings.Contains(lower, "sha512sums") || lower == "sums.txt"
}

// IsSignatureFile reports whether an asset is a detached signature or
// attestation rather than something to install.
func IsSignatureFile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range []string{".sig", ".asc", ".pem", ".sbom", ".intoto.jsonl", ".bundle", ".cert"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

var bsdLine = regexp.MustCompile(`^[A-Za-z0-9-]+ \((.+)\) = ([0-9a-fA-F]+)$`)

// ParseChecksums reads digests out of a checksum file, keyed by file name.
// It understands GNU coreutils lines ("hex  name", "hex *name"), BSD lines
// ("SHA256 (name) = hex"), and a bare digest, which belongs to fileName minus
// its checksum extension.
func ParseChecksums(fileName string, content []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := bsdLine.FindStringSubmatch(line); m != nil {
			out[path.Base(m[1])] = strings.ToLower(m[2])
			continue
		}
		fields := strings.Fields(line)
		if !isHex(fields[0]) {
			continue
		}
		if len(fields) == 1 {
			if target := trimChecksumExt(fileName); target != fileName {
				out[target] = strings.ToLower(fields[0])
			}
			continue
		}
		name := strings.TrimPrefix(strings.Join(fields[1:], " "), "*")
		out[path.Base(name)] = strings.ToLower(fields[0])
	}
	return out
}

func trimChecksumExt(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range checksumExts {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

func isHex(s string) bool {
	switch len(s) {
	case 32, 40, 64, 128:
		_, err := hex.DecodeString(s)
		return err == nil
	}
	return false
}
