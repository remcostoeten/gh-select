// Package gh wraps the GitHub API (via go-gh) for the data gh-select needs:
// the viewer's repositories and a repository's file tree.
package gh

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client bundles the REST and GraphQL clients, both authenticated with the
// user's existing `gh auth login` credentials.
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
}

// NewClient builds API clients from the active gh authentication. It returns a
// helpful error when the user is not logged in.
func NewClient() (*Client, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, authError(err)
	}
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, authError(err)
	}
	return &Client{rest: rest, gql: gql}, nil
}

func authError(err error) error {
	return fmt.Errorf("could not authenticate with GitHub (run `gh auth login`): %w", err)
}
