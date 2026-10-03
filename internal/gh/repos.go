package gh

import (
	"strings"
	"time"
)

// Repo is a single GitHub repository with the fields gh-select displays and
// acts on. JSON tags double as the cache serialization format.
type Repo struct {
	NameWithOwner  string    `json:"nameWithOwner"`
	Description    string    `json:"description"`
	IsPrivate      bool      `json:"isPrivate"`
	StargazerCount int       `json:"stargazerCount"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Language       string    `json:"language"`
	OwnerLogin     string    `json:"ownerLogin"`
	IsOwner        bool      `json:"isOwner"`
	IsFork         bool      `json:"isFork"`
	Parent         string    `json:"parent,omitempty"` // "owner/repo" a fork was made from, when known
}

// URL returns the canonical https URL for the repository.
func (r Repo) URL() string {
	return "https://github.com/" + r.NameWithOwner
}

// Name returns the repository name without its owner prefix.
func (r Repo) Name() string {
	if i := strings.LastIndex(r.NameWithOwner, "/"); i >= 0 {
		return r.NameWithOwner[i+1:]
	}
	return r.NameWithOwner
}

// reposQuery fetches a page of the repositories the viewer can reach, newest
// first: their own, those of organizations they belong to, and those they were
// added to as a collaborator.
//
// Both affiliation arguments are needed and they are not interchangeable:
// affiliations filters on the viewer's relationship to the repository, while
// ownerAffiliations filters on the owner's — and its default of
// [OWNER, COLLABORATOR] would drop every organization repository regardless of
// what affiliations says. The two are intersected, so both list all three.
//
// defaultBranchRef is deliberately omitted: it roughly triples per-page latency
// (it resolves a ref server-side), and we don't need it — tree/clone operations
// default to HEAD. Keeping the query to cheap fields holds each page near ~1.4s;
// owner { login } resolves from the node itself and costs nothing extra.
const reposQuery = `
query($cursor: String) {
  viewer {
    login
    repositories(first: 100, after: $cursor, affiliations: [OWNER, ORGANIZATION_MEMBER, COLLABORATOR], ownerAffiliations: [OWNER, ORGANIZATION_MEMBER, COLLABORATOR], orderBy: {field: UPDATED_AT, direction: DESC}) {
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

type reposResponse struct {
	Viewer struct {
		Login        string `json:"login"`
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
		} `json:"repositories"`
	} `json:"viewer"`
}

// FetchRepos returns every repository the authenticated user can reach: their
// own, their organizations', and those they collaborate on. It pages through the
// GraphQL API 100 repos at a time (one round-trip per page) which is far fewer
// requests and less data than the REST list endpoint.
func (c *Client) FetchRepos() ([]Repo, error) {
	var repos []Repo
	var cursor *string

	for {
		vars := map[string]interface{}{"cursor": cursor}

		var resp reposResponse
		if err := c.gql.Do(reposQuery, vars, &resp); err != nil {
			return nil, err
		}

		viewer := resp.Viewer.Login
		page := resp.Viewer.Repositories
		for _, n := range page.Nodes {
			repos = append(repos, Repo{
				NameWithOwner:  n.NameWithOwner,
				Description:    n.Description,
				IsPrivate:      n.IsPrivate,
				StargazerCount: n.StargazerCount,
				UpdatedAt:      n.UpdatedAt,
				Language:       n.PrimaryLanguage.Name,
				OwnerLogin:     n.Owner.Login,
				IsOwner:        n.Owner.Login == viewer,
				IsFork:         n.IsFork,
				Parent:         parentName(n.Parent),
			})
		}

		if !page.PageInfo.HasNextPage {
			break
		}
		end := page.PageInfo.EndCursor
		cursor = &end
	}

	return dedupeRepos(repos), nil
}

func parentName(p *struct {
	NameWithOwner string `json:"nameWithOwner"`
}) string {
	if p == nil {
		return ""
	}
	return p.NameWithOwner
}

// dedupeRepos drops repeated NameWithOwner entries, keeping the first occurrence
// so the server's UPDATED_AT DESC ordering survives. Broadening the affiliation
// filters lets the same repository arrive under more than one affiliation.
func dedupeRepos(repos []Repo) []Repo {
	seen := make(map[string]struct{}, len(repos))
	out := make([]Repo, 0, len(repos))
	for _, r := range repos {
		if _, dup := seen[r.NameWithOwner]; dup {
			continue
		}
		seen[r.NameWithOwner] = struct{}{}
		out = append(out, r)
	}
	return out
}
