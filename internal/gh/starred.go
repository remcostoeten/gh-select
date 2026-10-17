package gh

import (
	"fmt"
	"strings"
	"sync"
)

// forkBatch is how many repositories one aliased GraphQL lookup asks about.
const forkBatch = 100

// FetchStarred returns every repository the authenticated user has starred,
// most recently starred first. The REST pages load concurrently with the
// viewer's login; starred forks then get their parent and language, which the
// REST listing leaves out, in one batched lookup.
func (c *Client) FetchStarred() ([]Repo, error) {
	var (
		wg        sync.WaitGroup
		viewer    string
		viewerErr error
	)
	wg.Go(func() { viewer, viewerErr = c.viewerLogin() })
	repos, err := c.fetchRepoPages(starredPath)
	wg.Wait()
	if err != nil {
		return nil, err
	}
	if viewerErr != nil {
		return nil, viewerErr
	}
	forks, err := c.lookupForks(repos)
	if err != nil {
		return nil, err
	}
	return mergeForks(repos, viewer, forks), nil
}

func (c *Client) viewerLogin() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	err := c.rest.Get("user", &user)
	return user.Login, err
}

type forkNode struct {
	NameWithOwner string `json:"nameWithOwner"`
	Parent        *struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"parent"`
	PrimaryLanguage struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
}

// lookupForks fetches the parent and language of every fork in repos with
// one aliased query per forkBatch forks.
func (c *Client) lookupForks(repos []Repo) (map[string]forkFacts, error) {
	var names []string
	for _, r := range repos {
		if r.IsFork {
			names = append(names, r.NameWithOwner)
		}
	}
	forks := map[string]forkFacts{}
	for start := 0; start < len(names); start += forkBatch {
		var q strings.Builder
		q.WriteString("query {")
		for i, name := range names[start:min(start+forkBatch, len(names))] {
			owner, repo, _ := strings.Cut(name, "/")
			fmt.Fprintf(&q, " r%d: repository(owner: %q, name: %q) { nameWithOwner parent { nameWithOwner } primaryLanguage { name } }", i, owner, repo)
		}
		q.WriteString(" }")
		var resp map[string]*forkNode
		if err := c.gql.Do(q.String(), nil, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp {
			if n != nil {
				forks[n.NameWithOwner] = forkFacts{parent: parentName(n.Parent), language: n.PrimaryLanguage.Name}
			}
		}
	}
	return forks, nil
}
