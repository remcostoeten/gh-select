package gh

import (
	"fmt"
)

// branchesPerPage is GitHub's maximum page size for the branches endpoint.
const branchesPerPage = 100

// FetchBranches returns every branch name for nameWithOwner, paging through the
// REST API. The default branch, when identifiable, is hoisted to the front so
// it's the first pick.
func (c *Client) FetchBranches(nameWithOwner string) ([]string, error) {
	def, _ := c.defaultBranch(nameWithOwner) // best-effort; empty on failure

	var names []string
	for page := 1; ; page++ {
		path := fmt.Sprintf("repos/%s/branches?per_page=%d&page=%d",
			nameWithOwner, branchesPerPage, page)

		var resp []struct {
			Name string `json:"name"`
		}
		if err := c.rest.Get(path, &resp); err != nil {
			return nil, err
		}
		for _, b := range resp {
			names = append(names, b.Name)
		}
		if len(resp) < branchesPerPage {
			break
		}
	}

	return hoistDefault(names, def), nil
}

// defaultBranch resolves the repository's default branch name.
func (c *Client) defaultBranch(nameWithOwner string) (string, error) {
	var resp struct {
		DefaultBranch string `json:"default_branch"`
	}
	// nameWithOwner already contains the "owner/repo" path; escaping the slash
	// would break it, so concatenate directly like the other endpoints.
	if err := c.rest.Get("repos/"+nameWithOwner, &resp); err != nil {
		return "", err
	}
	return resp.DefaultBranch, nil
}

// hoistDefault moves def to the front of names if present, preserving the order
// of the rest. If def isn't in names, names is returned untouched.
func hoistDefault(names []string, def string) []string {
	if def == "" {
		return names
	}
	rest := make([]string, 0, len(names))
	found := false
	for _, n := range names {
		if n == def {
			found = true
			continue
		}
		rest = append(rest, n)
	}
	if !found {
		return names
	}
	return append([]string{def}, rest...)
}
