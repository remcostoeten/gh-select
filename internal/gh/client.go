// Package gh wraps the GitHub API (via go-gh) for the data gh-select needs:
// the viewer's repositories and a repository's file tree.
package gh

import (
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
)

// Client bundles the REST, GraphQL and raw HTTP clients, both authenticated with the
// user's existing `gh auth login` credentials.
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
	http *http.Client
}

// NewClient builds API clients from the active gh authentication. It returns a
// helpful error when the user is not logged in.
//
// The token is resolved once and shared: each api.Default*Client looks it up
// on its own, which can mean a keyring read per client.
func NewClient() (*Client, error) {
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	if token == "" {
		return nil, authError(fmt.Errorf("authentication token not found for host %s", host))
	}
	opts := api.ClientOptions{Host: host, AuthToken: token}
	rest, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, authError(err)
	}
	gql, err := api.NewGraphQLClient(opts)
	if err != nil {
		return nil, authError(err)
	}
	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, authError(err)
	}
	return &Client{rest: rest, gql: gql, http: httpClient}, nil
}

func authError(err error) error {
	return fmt.Errorf("could not authenticate with GitHub (run `gh auth login`): %w", err)
}
