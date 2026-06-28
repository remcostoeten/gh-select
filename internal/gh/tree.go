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
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
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
	if resp.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(resp.Content)
	}
	return []byte(resp.Content), nil
}
