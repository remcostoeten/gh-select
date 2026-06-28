package gh

import "time"

// Repo is a single GitHub repository with the fields gh-select displays and
// acts on. JSON tags double as the cache serialization format.
type Repo struct {
	NameWithOwner  string    `json:"nameWithOwner"`
	Description    string    `json:"description"`
	IsPrivate      bool      `json:"isPrivate"`
	StargazerCount int       `json:"stargazerCount"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Language       string    `json:"language"`
}

// URL returns the canonical https URL for the repository.
func (r Repo) URL() string {
	return "https://github.com/" + r.NameWithOwner
}

// reposQuery fetches a page of the viewer's owned repositories, newest first.
//
// defaultBranchRef is deliberately omitted: it roughly triples per-page latency
// (it resolves a ref server-side), and we don't need it — tree/clone operations
// default to HEAD. Keeping the query to cheap fields holds each page near ~1.4s.
const reposQuery = `
query($cursor: String) {
  viewer {
    repositories(first: 100, after: $cursor, ownerAffiliations: [OWNER], orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        nameWithOwner
        description
        isPrivate
        stargazerCount
        updatedAt
        primaryLanguage { name }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type reposResponse struct {
	Viewer struct {
		Repositories struct {
			Nodes []struct {
				NameWithOwner   string    `json:"nameWithOwner"`
				Description     string    `json:"description"`
				IsPrivate       bool      `json:"isPrivate"`
				StargazerCount  int       `json:"stargazerCount"`
				UpdatedAt       time.Time `json:"updatedAt"`
				PrimaryLanguage struct {
					Name string `json:"name"`
				} `json:"primaryLanguage"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"repositories"`
	} `json:"viewer"`
}

// FetchRepos returns all repositories owned by the authenticated user. It pages
// through the GraphQL API 100 repos at a time (one round-trip per page) which is
// far fewer requests and less data than the REST list endpoint.
func (c *Client) FetchRepos() ([]Repo, error) {
	var repos []Repo
	var cursor *string

	for {
		vars := map[string]interface{}{"cursor": cursor}

		var resp reposResponse
		if err := c.gql.Do(reposQuery, vars, &resp); err != nil {
			return nil, err
		}

		page := resp.Viewer.Repositories
		for _, n := range page.Nodes {
			repos = append(repos, Repo{
				NameWithOwner:  n.NameWithOwner,
				Description:    n.Description,
				IsPrivate:      n.IsPrivate,
				StargazerCount: n.StargazerCount,
				UpdatedAt:      n.UpdatedAt,
				Language:       n.PrimaryLanguage.Name,
			})
		}

		if !page.PageInfo.HasNextPage {
			break
		}
		end := page.PageInfo.EndCursor
		cursor = &end
	}

	return repos, nil
}
