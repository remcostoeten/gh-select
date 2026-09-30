package gh

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// DeleteRepo permanently deletes a repository on GitHub. There is no undo, and
// the token needs the delete_repo scope, which `gh auth login` does not grant
// by default — deleteError turns that refusal into the command that fixes it.
func (c *Client) DeleteRepo(nameWithOwner string) error {
	owner, name, ok := strings.Cut(nameWithOwner, "/")
	if !ok || owner == "" || name == "" {
		return fmt.Errorf("invalid repository %q", nameWithOwner)
	}
	if err := c.rest.Delete("repos/"+owner+"/"+name, nil); err != nil {
		return deleteError(nameWithOwner, err)
	}
	return nil
}

func deleteError(nameWithOwner string, err error) error {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case 403:
			return fmt.Errorf("%s: refused — the token is missing the delete_repo scope; run: gh auth refresh -s delete_repo",
				nameWithOwner)
		case 404:
			return fmt.Errorf("%s: not found, or your token has no admin access to it", nameWithOwner)
		}
	}
	return fmt.Errorf("%s: %w", nameWithOwner, err)
}
