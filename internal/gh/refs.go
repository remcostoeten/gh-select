package gh

import (
	"fmt"
	"net/url"
)

const maxTagPages = 3

// FetchTags returns up to 300 of the repository's tags, newest first.
func (c *Client) FetchTags(nameWithOwner string) ([]string, error) {
	var names []string
	for page := 1; page <= maxTagPages; page++ {
		var resp []struct {
			Name string `json:"name"`
		}
		if err := c.rest.Get(fmt.Sprintf("repos/%s/tags?per_page=100&page=%d", nameWithOwner, page), &resp); err != nil {
			return nil, err
		}
		for _, t := range resp {
			names = append(names, t.Name)
		}
		if len(resp) < 100 {
			break
		}
	}
	return names, nil
}

// CommitSHA resolves a branch, tag or "" (the default branch) to the commit it
// points at, for building permalinks.
func (c *Client) CommitSHA(nameWithOwner, ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	var resp struct {
		SHA string `json:"sha"`
	}
	if err := c.rest.Get(fmt.Sprintf("repos/%s/commits/%s", nameWithOwner, url.PathEscape(ref)), &resp); err != nil {
		return "", err
	}
	return resp.SHA, nil
}
