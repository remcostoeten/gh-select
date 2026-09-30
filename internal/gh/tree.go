package gh

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
)

// TreeEntry is one node in a repository's git tree.
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Size int    `json:"size"`
	SHA  string `json:"sha"`
}

// IsDir reports whether the entry is a directory (git tree object).
func (e TreeEntry) IsDir() bool { return e.Type == "tree" }

type treeResponse struct {
	Tree      []TreeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

// Tree is a repository's full file listing.
type Tree struct {
	Entries   []TreeEntry
	Truncated bool // GitHub caps very large trees; true means some entries are missing
}

// FetchTree returns the recursive file tree for ref (e.g. a branch name or
// "HEAD") in a single API call. Entries are sorted by path.
func (c *Client) FetchTree(nameWithOwner, ref string) (*Tree, error) {
	if ref == "" {
		ref = "HEAD"
	}
	path := fmt.Sprintf("repos/%s/git/trees/%s?recursive=1", nameWithOwner, url.PathEscape(ref))

	var resp treeResponse
	if err := c.rest.Get(path, &resp); err != nil {
		return nil, err
	}

	sort.Slice(resp.Tree, func(i, j int) bool {
		return resp.Tree[i].Path < resp.Tree[j].Path
	})
	return &Tree{Entries: resp.Tree, Truncated: resp.Truncated}, nil
}

type contentResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	SHA      string `json:"sha"`
}

// FetchReadme returns the repository's readme and its path. GitHub resolves
// which file that is (any casing, any extension, docs/ or .github/ fallbacks),
// so no guessing is needed here.
func (c *Client) FetchReadme(nameWithOwner string) (path string, content []byte, err error) {
	var resp contentResponse
	if err := c.rest.Get(fmt.Sprintf("repos/%s/readme", nameWithOwner), &resp); err != nil {
		return "", nil, err
	}
	if resp.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(resp.Content)
		return resp.Path, decoded, err
	}
	return resp.Path, []byte(resp.Content), nil
}

// FetchFile returns the decoded contents of a single file at ref.
func (c *Client) FetchFile(nameWithOwner, ref, filePath string) ([]byte, error) {
	p := fmt.Sprintf("repos/%s/contents/%s", nameWithOwner, filePath)
	if ref != "" {
		p += "?ref=" + url.QueryEscape(ref)
	}

	var resp contentResponse
	if err := c.rest.Get(p, &resp); err != nil {
		return nil, err
	}
	switch resp.Encoding {
	case "base64":
		return base64.StdEncoding.DecodeString(resp.Content)
	case "none":
		// The contents API leaves files over 1 MB empty; the blob API serves up to 100 MB.
		return c.fetchBlob(nameWithOwner, resp.SHA)
	}
	return []byte(resp.Content), nil
}

func (c *Client) fetchBlob(nameWithOwner, sha string) ([]byte, error) {
	var resp contentResponse
	if err := c.rest.Get(fmt.Sprintf("repos/%s/git/blobs/%s", nameWithOwner, sha), &resp); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(resp.Content)
}
