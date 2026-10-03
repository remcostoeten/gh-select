package gh

import "time"

const starredQuery = `
query($cursor: String) {
  viewer {
    login
    starredRepositories(first: 100, after: $cursor, orderBy: {field: STARRED_AT, direction: DESC}) {
      nodes {
        nameWithOwner
        description
        isPrivate
        stargazerCount
        updatedAt
        primaryLanguage { name }
        owner { login }
        isFork
        parent { nameWithOwner }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type starredResponse struct {
	Viewer struct {
		Login               string `json:"login"`
		StarredRepositories struct {
			Nodes []struct {
				NameWithOwner   string    `json:"nameWithOwner"`
				Description     string    `json:"description"`
				IsPrivate       bool      `json:"isPrivate"`
				StargazerCount  int       `json:"stargazerCount"`
				UpdatedAt       time.Time `json:"updatedAt"`
				PrimaryLanguage struct {
					Name string `json:"name"`
				} `json:"primaryLanguage"`
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
				IsFork bool `json:"isFork"`
				Parent *struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"parent"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"starredRepositories"`
	} `json:"viewer"`
}

// FetchStarred returns every repository the authenticated user has starred,
// most recently starred first.
func (c *Client) FetchStarred() ([]Repo, error) {
	var repos []Repo
	var cursor *string
	for {
		var resp starredResponse
		if err := c.gql.Do(starredQuery, map[string]interface{}{"cursor": cursor}, &resp); err != nil {
			return nil, err
		}
		page := resp.Viewer.StarredRepositories
		for _, n := range page.Nodes {
			repos = append(repos, Repo{
				NameWithOwner:  n.NameWithOwner,
				Description:    n.Description,
				IsPrivate:      n.IsPrivate,
				StargazerCount: n.StargazerCount,
				UpdatedAt:      n.UpdatedAt,
				Language:       n.PrimaryLanguage.Name,
				OwnerLogin:     n.Owner.Login,
				IsOwner:        n.Owner.Login == resp.Viewer.Login,
				IsFork:         n.IsFork,
				Parent:         parentName(n.Parent),
			})
		}
		if !page.PageInfo.HasNextPage {
			return repos, nil
		}
		end := page.PageInfo.EndCursor
		cursor = &end
	}
}
